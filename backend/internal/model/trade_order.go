package model

import "time"

// TradeOrder is an order created when a buyer intends to purchase a product.
type TradeOrder struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	ProductID         uint       `gorm:"index;not null" json:"product_id"`
	BuyerID           uint       `gorm:"index;not null" json:"buyer_id"`
	SellerID          uint       `gorm:"index;not null" json:"seller_id"`
	SlotID            *uint      `gorm:"index" json:"slot_id"`
	Status            string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	BuyerConfirmedAt  *time.Time `json:"buyer_confirmed_at"`
	SellerConfirmedAt *time.Time `json:"seller_confirmed_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	CreatedAt         time.Time  `json:"created_at"`

	// Slot is the reserved face-to-face handover window. It is populated by
	// the service layer for order detail/list responses and is not a column
	// nor a GORM-managed association (slots are preloaded manually).
	Slot *TradeSlot `gorm:"-" json:"slot,omitempty"`
}
