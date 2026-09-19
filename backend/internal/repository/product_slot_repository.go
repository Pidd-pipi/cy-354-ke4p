package repository

import (
	"context"
	"time"

	"github.com/lp/campus-market/internal/constants"
	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/gorm"
)

// ProductSlotRepository persists meetup slot rows.
type ProductSlotRepository struct {
	db *gorm.DB
}

// NewProductSlotRepository builds a ProductSlotRepository.
func NewProductSlotRepository(db *gorm.DB) *ProductSlotRepository {
	return &ProductSlotRepository{db: db}
}

// Transaction runs fn inside a database transaction for cross-repository writes.
func (r *ProductSlotRepository) Transaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	return Transaction(ctx, r.db, fn)
}

// CreateBatch inserts all offered slots of a newly published product.
func (r *ProductSlotRepository) CreateBatch(ctx context.Context, slots []model.ProductSlot) error {
	if len(slots) == 0 {
		return nil
	}
	return db(ctx, r.db).Create(&slots).Error
}

// ListByProduct returns all slots of a product ordered chronologically.
func (r *ProductSlotRepository) ListByProduct(ctx context.Context, productID uint) ([]model.ProductSlot, error) {
	var slots []model.ProductSlot
	err := db(ctx, r.db).
		Where("product_id = ?", productID).
		Order("start_at ASC").
		Find(&slots).Error
	if err != nil {
		return nil, err
	}
	return slots, nil
}

// FindByProductAndStartForUpdate takes a row lock on one slot and returns it.
// Must be called inside a transaction.
func (r *ProductSlotRepository) FindByProductAndStartForUpdate(ctx context.Context, productID uint, startAt time.Time) (*model.ProductSlot, error) {
	var slot model.ProductSlot
	err := db(ctx, r.db).
		Clauses(ClauseLockingUpdate).
		Where("product_id = ? AND start_at = ?", productID, startAt).
		First(&slot).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &slot, nil
}

// Occupy atomically marks an open slot as occupied by orderID. Returns
// util.ErrConflict when the slot is no longer open, which is how concurrent
// contenders lose the race without creating a duplicate order.
func (r *ProductSlotRepository) Occupy(ctx context.Context, slotID, orderID uint) error {
	res := db(ctx, r.db).Model(&model.ProductSlot{}).
		Where("id = ? AND status = ?", slotID, constants.ProductSlotStatusOpen).
		Updates(map[string]interface{}{"status": constants.ProductSlotStatusOccupied, "order_id": orderID})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}

// Release returns a slot to the open pool after its order is cancelled.
func (r *ProductSlotRepository) Release(ctx context.Context, slotID uint) error {
	res := db(ctx, r.db).Model(&model.ProductSlot{}).
		Where("id = ? AND status = ?", slotID, constants.ProductSlotStatusOccupied).
		Updates(map[string]interface{}{"status": constants.ProductSlotStatusOpen, "order_id": nil})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrConflict
	}
	return nil
}

// LockAllForProduct permanently locks every slot of a completed product.
func (r *ProductSlotRepository) LockAllForProduct(ctx context.Context, productID, orderID uint) (int64, error) {
	res := db(ctx, r.db).Model(&model.ProductSlot{}).
		Where("product_id = ? AND status <> ?", productID, constants.ProductSlotStatusLocked).
		Updates(map[string]interface{}{"status": constants.ProductSlotStatusLocked, "order_id": gorm.Expr("COALESCE(order_id, ?)", orderID)})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
