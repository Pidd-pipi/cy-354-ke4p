package model

import "time"

// TradeSlot is an optional half-hour face-to-face handover window offered by
// the seller when publishing a product. Buyers must pick an unexpired,
// available slot when ordering a product that defines slots.
type TradeSlot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProductID uint      `gorm:"index;uniqueIndex:uk_trade_slots_product_start,priority:1;not null" json:"product_id"`
	StartTime time.Time `gorm:"type:datetime(3);uniqueIndex:uk_trade_slots_product_start,priority:2;not null" json:"start_time"`
	EndTime   time.Time `gorm:"type:datetime(3);not null" json:"end_time"`
	Status    string    `gorm:"size:16;index;not null;default:available" json:"status"`
	OrderID   *uint     `gorm:"uniqueIndex:uk_trade_slots_order" json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
