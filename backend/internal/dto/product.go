package dto

import "github.com/lp/campus-market/internal/model"

// CreateProductRequest is the payload for publishing a second-hand item.
type CreateProductRequest struct {
	Title         string            `json:"title" binding:"required,min=2,max=64"`
	Description   string            `json:"description"`
	Price         float64           `json:"price" binding:"required,gt=0"`
	Category      string            `json:"category" binding:"required"`
	Condition     string            `json:"condition" binding:"required"`
	Campus        string            `json:"campus" binding:"required,max=64"`
	TradeLocation string            `json:"trade_location" binding:"required,max=128"`
	Images        string            `json:"images"`
	Slots         []CreateSlotInput `json:"slots" binding:"dive"`
}

// ListProductQuery adds filters to pagination.
type ListProductQuery struct {
	PageQuery
	Category string `form:"category"`
	Campus   string `form:"campus"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

// ProductDetailView augments a product with its handover slot availability.
type ProductDetailView struct {
	model.Product
	Slots          []SlotView `json:"slots"`
	AvailableSlots int        `json:"available_slots"`
	TotalSlots     int        `json:"total_slots"`
	HasSlots       bool       `json:"has_slots"`
}
