package model

import "time"

// ProductSlot is a half-hour meetup time window a seller offers on a product.
type ProductSlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"uniqueIndex:uniq_product_slot_start,priority:1;not null" json:"product_id"`
	StartAt   time.Time `gorm:"column:start_at;uniqueIndex:uniq_product_slot_start,priority:2;not null" json:"start_at"`
	Status    string    `gorm:"size:16;index;not null;default:open" json:"status"`
	OrderID   *uint     `gorm:"index" json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

// TableName pins the table name for the product slot entity.
func (ProductSlot) TableName() string { return "product_slots" }
