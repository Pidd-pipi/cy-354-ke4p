package model

import "time"

// Product is a second-hand item posted by a student seller.
type Product struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	SellerID      uint      `gorm:"index;not null" json:"seller_id"`
	Title         string    `gorm:"size:64;not null" json:"title"`
	Description   string    `gorm:"type:text" json:"description"`
	Price         float64   `gorm:"not null" json:"price"`
	Category      string    `gorm:"size:24;index;not null" json:"category"`
	Condition     string    `gorm:"size:16" json:"condition"`
	Campus        string    `gorm:"size:64;index" json:"campus"`
	TradeLocation string    `gorm:"size:128" json:"trade_location"`
	Images        string    `gorm:"type:text" json:"images"`
	Status        string    `gorm:"size:16;index;not null;default:on_sale" json:"status"`
	CreatedAt     time.Time `json:"created_at"`

	// HasSlots reports whether the seller offered meetup slots; OpenSlotCount
	// counts currently bookable (non-expired, open) slots. Both fields are
	// populated by the repository list queries and ignored on writes.
	HasSlots      bool `gorm:"-:all" json:"has_slots"`
	OpenSlotCount int  `gorm:"-:all" json:"open_slot_count"`
}
