package service

import (
	"context"
	"sort"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
)

// fakeSlotStore is an in-memory SlotStore used by service tests.
type fakeSlotStore struct {
	slots  map[uint]*model.ProductSlot
	nextID uint
}

func newFakeSlotStore() *fakeSlotStore {
	return &fakeSlotStore{slots: map[uint]*model.ProductSlot{}, nextID: 1}
}

func (f *fakeSlotStore) CreateBatch(_ context.Context, slots []model.ProductSlot) error {
	for i := range slots {
		slots[i].ID = f.nextID
		cp := slots[i]
		f.slots[cp.ID] = &cp
		f.nextID++
	}
	return nil
}

func (f *fakeSlotStore) ListByProduct(_ context.Context, productID uint) ([]model.ProductSlot, error) {
	var out []model.ProductSlot
	for _, s := range f.slots {
		if s.ProductID == productID {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartAt.Before(out[j].StartAt) })
	return out, nil
}

func (f *fakeSlotStore) FindByProductAndStartForUpdate(_ context.Context, productID uint, startAt time.Time) (*model.ProductSlot, error) {
	for _, s := range f.slots {
		if s.ProductID == productID && s.StartAt.Equal(startAt) {
			cp := *s
			return &cp, nil
		}
	}
	return nil, util.ErrNotFound
}

func (f *fakeSlotStore) Occupy(_ context.Context, slotID, orderID uint) error {
	s, ok := f.slots[slotID]
	if !ok {
		return util.ErrNotFound
	}
	if s.Status != constants.ProductSlotStatusOpen {
		return util.ErrConflict
	}
	s.Status = constants.ProductSlotStatusOccupied
	oid := orderID
	s.OrderID = &oid
	return nil
}

func (f *fakeSlotStore) Release(_ context.Context, slotID uint) error {
	s, ok := f.slots[slotID]
	if !ok {
		return util.ErrNotFound
	}
	if s.Status != constants.ProductSlotStatusOccupied {
		return util.ErrConflict
	}
	s.Status = constants.ProductSlotStatusOpen
	s.OrderID = nil
	return nil
}

func (f *fakeSlotStore) LockAllForProduct(_ context.Context, productID, orderID uint) (int64, error) {
	var n int64
	for _, s := range f.slots {
		if s.ProductID == productID && s.Status != constants.ProductSlotStatusLocked {
			s.Status = constants.ProductSlotStatusLocked
			oid := orderID
			s.OrderID = &oid
			n++
		}
	}
	return n, nil
}

// fakeTxRunner executes the callback without a real database transaction.
type fakeTxRunner struct{}

func (fakeTxRunner) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

// fakeOrderStore is an in-memory orderStore used by trade order service tests.
type fakeOrderStore struct {
	orders  map[uint]*model.TradeOrder
	nextID  uint
	created []uint
}

func newFakeOrderStore() *fakeOrderStore {
	return &fakeOrderStore{orders: map[uint]*model.TradeOrder{}, nextID: 1}
}

func (f *fakeOrderStore) Create(_ context.Context, o *model.TradeOrder) error {
	o.ID = f.nextID
	f.nextID++
	cp := *o
	f.orders[cp.ID] = &cp
	f.created = append(f.created, cp.ID)
	return nil
}

func (f *fakeOrderStore) FindByID(_ context.Context, id uint) (*model.TradeOrder, error) {
	if o, ok := f.orders[id]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, util.ErrNotFound
}

func (f *fakeOrderStore) FindByProductAndBuyer(_ context.Context, productID, buyerID uint) (*model.TradeOrder, error) {
	for _, o := range f.orders {
		if o.ProductID == productID && o.BuyerID == buyerID &&
			(o.Status == constants.TradeStatusPending || o.Status == constants.TradeStatusConfirmed) {
			cp := *o
			return &cp, nil
		}
	}
	return nil, util.ErrNotFound
}

func (f *fakeOrderStore) ListByUser(_ context.Context, userID uint, _, _ int) ([]model.TradeOrder, int64, error) {
	var out []model.TradeOrder
	for _, o := range f.orders {
		if o.BuyerID == userID || o.SellerID == userID {
			out = append(out, *o)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeOrderStore) UpdateStatus(_ context.Context, id uint, status string) error {
	o, ok := f.orders[id]
	if !ok {
		return util.ErrNotFound
	}
	o.Status = status
	return nil
}

func (f *fakeOrderStore) UpdateStatusFromPending(_ context.Context, id uint, status string) error {
	o, ok := f.orders[id]
	if !ok {
		return util.ErrNotFound
	}
	if o.Status != constants.TradeStatusPending {
		return util.ErrConflict
	}
	o.Status = status
	return nil
}

func (f *fakeOrderStore) UpdateBuyerConfirmed(_ context.Context, id uint, _ interface{}) error {
	o, ok := f.orders[id]
	if !ok || o.Status != constants.TradeStatusPending {
		return util.ErrConflict
	}
	o.Status = constants.TradeStatusConfirmed
	return nil
}

func (f *fakeOrderStore) UpdateSellerConfirmed(_ context.Context, id uint, _ interface{}) error {
	o, ok := f.orders[id]
	if !ok || o.Status != constants.TradeStatusConfirmed {
		return util.ErrConflict
	}
	o.Status = constants.TradeStatusCompleted
	return nil
}

func (f *fakeOrderStore) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}
