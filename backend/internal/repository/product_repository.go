package repository

import (
	"context"
	"time"

	"github.com/lp/campus-market/internal/model"
	"github.com/lp/campus-market/internal/util"
	"gorm.io/gorm"
)

// ProductRepository persists product rows.
type ProductRepository struct {
	db *gorm.DB
}

// NewProductRepository builds a ProductRepository.
func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// Create inserts a new product.
func (r *ProductRepository) Create(ctx context.Context, p *model.Product) error {
	return db(ctx, r.db).Create(p).Error
}

// FindByID returns a product by id.
func (r *ProductRepository) FindByID(ctx context.Context, id uint) (*model.Product, error) {
	var p model.Product
	err := db(ctx, r.db).First(&p, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &p, nil
}

// FindByIDForUpdate returns a product with a row lock; use inside a
// transaction to serialize concurrent booking attempts.
func (r *ProductRepository) FindByIDForUpdate(ctx context.Context, id uint) (*model.Product, error) {
	var p model.Product
	err := db(ctx, r.db).Clauses(ClauseLockingUpdate).First(&p, id).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return &p, nil
}

// List filters products by category/campus/keyword/status with pagination.
func (r *ProductRepository) List(ctx context.Context, category, campus, keyword, status string, page, pageSize int) ([]model.Product, int64, error) {
	q := db(ctx, r.db).Model(&model.Product{})
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if campus != "" {
		q = q.Where("campus = ?", campus)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if keyword != "" {
		q = q.Where("title LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.Product
	err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	if err != nil {
		return nil, 0, err
	}
	if err := r.hydrateSlotCounts(ctx, items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// slotCountRow holds grouped slot counts for a product.
type slotCountRow struct {
	ProductID uint
	Total     int64
	Open      int64
}

func (r *ProductRepository) hydrateSlotCounts(ctx context.Context, items []model.Product) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(items))
	for _, p := range items {
		ids = append(ids, p.ID)
	}
	var rows []slotCountRow
	err := db(ctx, r.db).Model(&model.ProductSlot{}).
		Select("product_id AS product_id, COUNT(*) AS total, "+
			"SUM(CASE WHEN status = ? AND start_at > ? THEN 1 ELSE 0 END) AS open",
			"open", time.Now()).
		Where("product_id IN ?", ids).
		Group("product_id").
		Scan(&rows).Error
	if err != nil {
		return err
	}
	counts := make(map[uint]slotCountRow, len(rows))
	for _, row := range rows {
		counts[row.ProductID] = row
	}
	for i := range items {
		if c, ok := counts[items[i].ID]; ok {
			items[i].HasSlots = c.Total > 0
			items[i].OpenSlotCount = int(c.Open)
		}
	}
	return nil
}

// UpdateStatus sets the product status.
func (r *ProductRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	res := db(ctx, r.db).Model(&model.Product{}).Where("id = ?", id).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return util.ErrNotFound
	}
	return nil
}

// Count returns the total product count.
func (r *ProductRepository) Count(ctx context.Context) (int64, error) {
	var n int64
	err := db(ctx, r.db).Model(&model.Product{}).Count(&n).Error
	return n, err
}
