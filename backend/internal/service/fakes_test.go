package service

import (
	"context"
	"sync"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// fakeDB is a shared in-memory store with commit/rollback semantics so that
// service-level transactions (order insert + slot occupy, cancel + release)
// behave like the real GORM database transaction used in production.
//
// Concurrency note: all mutating methods hold db.mu, and transaction control
// reuses the same mutex through internal non-locking helpers, avoiding
// re-entrant locking.
type fakeDB struct {
	// txMu serializes whole transactions, mimicking row-lock contention:
	// only one create/reserve (or cancel/release) sequence is in flight at a
	// time, so the conditional slot status check still admits exactly one
	// winner under concurrent buyers.
	txMu     sync.Mutex
	mu       sync.Mutex
	products map[uint]*model.Product
	orders   map[uint]*model.TradeOrder
	slots    map[uint]*model.TradeSlot
	nextPID  uint
	nextOID  uint
	nextSID  uint
	txDepth  int
	undo     []func()
}

func newFakeDB() *fakeDB {
	return &fakeDB{
		products: map[uint]*model.Product{},
		orders:   map[uint]*model.TradeOrder{},
		slots:    map[uint]*model.TradeSlot{},
		nextPID:  1, nextOID: 1, nextSID: 1,
	}
}

// begin pushes a rollback checkpoint (nil marker). Caller holds mu.
func (d *fakeDB) beginLocked() {
	d.txDepth++
	d.undo = append(d.undo, nil)
}

// commitLocked pops back to the latest marker, keeping the changes. Caller holds mu.
func (d *fakeDB) commitLocked() {
	for i := len(d.undo) - 1; i >= 0; i-- {
		if d.undo[i] == nil {
			d.undo = d.undo[:i]
			break
		}
	}
	d.txDepth--
}

// rollbackLocked replays undo callbacks back to the latest marker. Caller holds mu.
func (d *fakeDB) rollbackLocked() {
	for i := len(d.undo) - 1; i >= 0; i-- {
		if d.undo[i] == nil {
			d.undo = d.undo[:i]
			break
		}
		d.undo[i]()
		d.undo = d.undo[:i]
	}
	d.txDepth--
}

func (d *fakeDB) inTxLocked() bool { return d.txDepth > 0 }

// transaction runs fn with all-or-nothing semantics across the shared tables,
// serialized to emulate database row-level contention.
func (d *fakeDB) transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	d.txMu.Lock()
	defer d.txMu.Unlock()

	d.mu.Lock()
	d.beginLocked()
	d.mu.Unlock()

	err := fn(ctx)

	d.mu.Lock()
	if err != nil {
		d.rollbackLocked()
	} else {
		d.commitLocked()
	}
	d.mu.Unlock()
	return err
}

// ---- products ----

type fakeProductRepo struct{ db *fakeDB }

func newFakeProductRepo(db *fakeDB) *fakeProductRepo { return &fakeProductRepo{db: db} }

func (r *fakeProductRepo) Create(_ context.Context, p *model.Product) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	if p.ID == 0 {
		p.ID = r.db.nextPID
		r.db.nextPID++
	}
	id := p.ID
	cp := *p
	if r.db.inTxLocked() {
		_, existed := r.db.products[id]
		prev := r.db.products[id]
		r.db.undo = append(r.db.undo, func() {
			if existed {
				r.db.products[id] = prev
			} else {
				delete(r.db.products, id)
			}
		})
	}
	r.db.products[id] = &cp
	return nil
}

func (r *fakeProductRepo) FindByID(_ context.Context, id uint) (*model.Product, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	if p, ok := r.db.products[id]; ok {
		cp := *p
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (r *fakeProductRepo) List(_ context.Context, category, campus, keyword, status string, _, _ int) ([]model.Product, int64, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	var out []model.Product
	for _, p := range r.db.products {
		if category != "" && p.Category != category {
			continue
		}
		if campus != "" && p.Campus != campus {
			continue
		}
		if status != "" && p.Status != status {
			continue
		}
		out = append(out, *p)
	}
	return out, int64(len(out)), nil
}

func (r *fakeProductRepo) UpdateStatus(_ context.Context, id uint, status string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	p, ok := r.db.products[id]
	if !ok {
		return util.ErrNotFound
	}
	old := p.Status
	if r.db.inTxLocked() {
		r.db.undo = append(r.db.undo, func() { r.db.products[id].Status = old })
	}
	p.Status = status
	return nil
}

func (r *fakeProductRepo) Count(context.Context) (int64, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	return int64(len(r.db.products)), nil
}

// ---- trade orders ----

type fakeOrderRepo struct{ db *fakeDB }

func newFakeOrderRepo(db *fakeDB) *fakeOrderRepo { return &fakeOrderRepo{db: db} }

func (r *fakeOrderRepo) Create(_ context.Context, o *model.TradeOrder) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	o.ID = r.db.nextOID
	r.db.nextOID++
	id := o.ID
	cp := *o
	if r.db.inTxLocked() {
		r.db.undo = append(r.db.undo, func() { delete(r.db.orders, id) })
	}
	r.db.orders[id] = &cp
	return nil
}

func (r *fakeOrderRepo) FindByID(_ context.Context, id uint) (*model.TradeOrder, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	if o, ok := r.db.orders[id]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (r *fakeOrderRepo) FindByProductAndBuyer(_ context.Context, productID, buyerID uint) (*model.TradeOrder, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	for _, o := range r.db.orders {
		if o.ProductID == productID && o.BuyerID == buyerID &&
			(o.Status == constants.TradeStatusPending || o.Status == constants.TradeStatusConfirmed) {
			cp := *o
			return &cp, nil
		}
	}
	return nil, util.ErrNotFound
}

func (r *fakeOrderRepo) ListByUser(_ context.Context, userID uint, _, _ int) ([]model.TradeOrder, int64, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	var out []model.TradeOrder
	for _, o := range r.db.orders {
		if o.BuyerID == userID || o.SellerID == userID {
			out = append(out, *o)
		}
	}
	return out, int64(len(out)), nil
}

func (r *fakeOrderRepo) UpdateStatus(_ context.Context, id uint, status string) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	o, ok := r.db.orders[id]
	if !ok {
		return util.ErrNotFound
	}
	old := o.Status
	if r.db.inTxLocked() {
		r.db.undo = append(r.db.undo, func() { r.db.orders[id].Status = old })
	}
	o.Status = status
	return nil
}

func (r *fakeOrderRepo) UpdateBuyerConfirmed(_ context.Context, id uint, _ interface{}) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	o, ok := r.db.orders[id]
	if !ok || o.Status != constants.TradeStatusPending {
		return util.ErrConflict
	}
	o.Status = constants.TradeStatusConfirmed
	return nil
}

func (r *fakeOrderRepo) UpdateSellerConfirmed(_ context.Context, id uint, _ interface{}) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	o, ok := r.db.orders[id]
	if !ok || o.Status != constants.TradeStatusConfirmed {
		return util.ErrConflict
	}
	o.Status = constants.TradeStatusCompleted
	return nil
}

func (r *fakeOrderRepo) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return r.db.transaction(ctx, fn)
}

// ---- trade slots ----

type fakeSlotRepo struct{ db *fakeDB }

func newFakeSlotRepo(db *fakeDB) *fakeSlotRepo { return &fakeSlotRepo{db: db} }

func (r *fakeSlotRepo) CreateBatch(_ context.Context, slots []model.TradeSlot) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	for i := range slots {
		for _, ex := range r.db.slots {
			if ex.ProductID == slots[i].ProductID && ex.StartTime.Equal(slots[i].StartTime) {
				return util.ErrConflict
			}
		}
		slots[i].ID = r.db.nextSID
		r.db.nextSID++
		cp := slots[i]
		id := cp.ID
		if r.db.inTxLocked() {
			r.db.undo = append(r.db.undo, func() { delete(r.db.slots, id) })
		}
		r.db.slots[id] = &cp
	}
	return nil
}

func (r *fakeSlotRepo) ListByProduct(_ context.Context, productID uint) ([]model.TradeSlot, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	var out []model.TradeSlot
	for _, s := range r.db.slots {
		if s.ProductID == productID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (r *fakeSlotRepo) FindByID(_ context.Context, id uint) (*model.TradeSlot, error) {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	if s, ok := r.db.slots[id]; ok {
		cp := *s
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (r *fakeSlotRepo) TryOccupy(_ context.Context, slotID, orderID uint, now time.Time) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	s, ok := r.db.slots[slotID]
	if !ok {
		return util.ErrConflict
	}
	if s.Status != constants.SlotStatusAvailable && s.Status != constants.SlotStatusReleased {
		return util.ErrConflict
	}
	if !s.StartTime.After(now) {
		return util.ErrConflict
	}
	oldStatus, oldOrder := s.Status, s.OrderID
	if r.db.inTxLocked() {
		r.db.undo = append(r.db.undo, func() { s.Status = oldStatus; s.OrderID = oldOrder })
	}
	s.Status = constants.SlotStatusLocked
	s.OrderID = &orderID
	return nil
}

func (r *fakeSlotRepo) Release(_ context.Context, slotID uint, _ time.Time) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	s, ok := r.db.slots[slotID]
	if !ok || s.Status != constants.SlotStatusLocked {
		return util.ErrConflict
	}
	oldStatus, oldOrder := s.Status, s.OrderID
	if r.db.inTxLocked() {
		r.db.undo = append(r.db.undo, func() { s.Status = oldStatus; s.OrderID = oldOrder })
	}
	s.Status = constants.SlotStatusReleased
	s.OrderID = nil
	return nil
}

func (r *fakeSlotRepo) PreloadOrders(_ context.Context, orders []model.TradeOrder) error {
	r.db.mu.Lock()
	defer r.db.mu.Unlock()
	for i := range orders {
		if orders[i].SlotID != nil {
			if s, ok := r.db.slots[*orders[i].SlotID]; ok {
				cp := *s
				orders[i].Slot = &cp
			}
		}
	}
	return nil
}

func (r *fakeSlotRepo) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return r.db.transaction(ctx, fn)
}
