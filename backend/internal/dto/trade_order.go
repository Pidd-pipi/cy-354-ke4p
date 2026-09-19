package dto

// CreateTradeOrderRequest creates a purchase intent for a product. For
// products with meetup slots the buyer must pick a non-expired, open slot.
type CreateTradeOrderRequest struct {
	ProductID uint   `json:"product_id" binding:"required"`
	SlotStart string `json:"slot_start"`
}
