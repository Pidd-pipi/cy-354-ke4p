package dto

// CreateTradeOrderRequest creates a purchase intent for a product.
type CreateTradeOrderRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
	// SlotID is required when the product defines face-to-face handover slots.
	SlotID *uint `json:"slot_id"`
}
