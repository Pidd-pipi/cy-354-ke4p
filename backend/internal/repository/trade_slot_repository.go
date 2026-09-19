package repository

import (
	"context"
	"errors"
	"time"

	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/gorm"
)

// TradeSlotRepository persists face-to-face handover slot rows.
type TradeSlotRepository struct {
	db *gorm.DB
}

// NewTradeSlotRepository builds a TradeSlotRepository.
func NewTradeSlotRepository(db *gorm.DB) *TradeSlotRepository {
	return &TradeSlotRepository{db: db}
}

// Transaction runs fn inside a database transaction for cross-repository writes.
func (r *TradeSlotRepository) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return Transaction(ctx, r.db, fn)
}

// CreateBatch inserts all slots for a product in one statement.
func (r *TradeSlotRepository) CreateBatch(ctx context.Context, slots []model.TradeSlot) error {
	if len(slots) == 0 {
		return nil
	}
	return db(ctx, r.db).Create(&slots).Error
}

// ListByProduct returns every slot of a product ordered by start time.
func (r *TradeSlotRepository) ListByProduct(ctx context.Context, productID uint) ([]model.TradeSlot, error) {
	var slots []model.TradeSlot
	err := db(ctx, r.db).Where("product_id = ?", productID).
		Order("start_time ASC").Find(&slots).Error
	if err != nil {
		return nil, err
	}
	return slots, nil
}

// FindByID returns a slot by id.
func (r *TradeSlotRepository) FindByID(ctx context.Context, id uint) (*model.TradeSlot, error) {
	var s model.TradeSlot
	err := db(ctx, r.db).First(&s, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &s, nil
}

// TryOccupy atomically reserves an unexpired, free slot (available or
// released by a previous cancellation) for an order. The conditional UPDATE
// is the core guard against concurrent grabs: InnoDB serializes row writes so
// at most one transaction gets RowsAffected == 1 for the same slot.
func (r *TradeSlotRepository) TryOccupy(ctx context.Context, slotID, orderID uint, now time.Time) error {
	res := db(ctx, r.db).Model(&model.TradeSlot{}).
		Where("id = ? AND status IN ? AND start_time > ?", slotID, []string{"available", "released"}, now).
		Updates(map[string]interface{}{"status": "locked", "order_id": orderID, "updated_at": now})
	if res.Error != nil {
		if errors.Is(res.Error, gorm.ErrDuplicatedKey) {
			return util.ErrConflict
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}

// Release frees a locked slot after a pending order is cancelled. The slot
// moves to "released" (still bookable) and is detached from the cancelled
// order; expired windows are simply skipped by later bookings.
func (r *TradeSlotRepository) Release(ctx context.Context, slotID uint, now time.Time) error {
	res := db(ctx, r.db).Model(&model.TradeSlot{}).
		Where("id = ? AND status = ?", slotID, "locked").
		Updates(map[string]interface{}{"status": "released", "order_id": nil, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}

// PreloadOrders attaches the slot of each order in place.
func (r *TradeSlotRepository) PreloadOrders(ctx context.Context, orders []model.TradeOrder) error {
	ids := make([]uint, 0, len(orders))
	for i := range orders {
		if orders[i].SlotID != nil {
			ids = append(ids, *orders[i].SlotID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var slots []model.TradeSlot
	if err := db(ctx, r.db).Where("id IN ?", ids).Find(&slots).Error; err != nil {
		return err
	}
	byID := make(map[uint]*model.TradeSlot, len(slots))
	for i := range slots {
		byID[slots[i].ID] = &slots[i]
	}
	for i := range orders {
		if orders[i].SlotID != nil {
			orders[i].Slot = byID[*orders[i].SlotID]
		}
	}
	return nil
}
