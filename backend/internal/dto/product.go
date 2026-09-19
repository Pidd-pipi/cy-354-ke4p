package dto

// CreateProductRequest is the payload for publishing a second-hand item.
type CreateProductRequest struct {
	Title         string   `json:"title" binding:"required,min=2,max=64"`
	Description   string   `json:"description"`
	Price         float64  `json:"price" binding:"required,gt=0"`
	Category      string   `json:"category" binding:"required"`
	Condition     string   `json:"condition" binding:"required"`
	Campus        string   `json:"campus" binding:"required,max=64"`
	TradeLocation string   `json:"trade_location" binding:"required,max=128"`
	Images        string   `json:"images"`
	SlotStarts    []string `json:"slot_starts"`
}

// ListProductQuery adds filters to pagination.
type ListProductQuery struct {
	PageQuery
	Category string `form:"category"`
	Campus   string `form:"campus"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

// ProductSlotView is one meetup slot as shown on the product detail page.
type ProductSlotView struct {
	ID        uint   `json:"id"`
	StartAt   string `json:"start_at"`
	Status    string `json:"status"`
	Available bool   `json:"available"`
}

// ProductDetailResponse embeds the product and its meetup slot availability.
type ProductDetailResponse struct {
	Product    interface{}       `json:"product"`
	Slots      []ProductSlotView `json:"slots"`
	TotalSlots int               `json:"total_slots"`
	OpenSlots  int               `json:"open_slots"`
	Bookable   bool              `json:"bookable"`
}
