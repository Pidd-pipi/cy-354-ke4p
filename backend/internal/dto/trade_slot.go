package dto

import "time"

// CreateSlotInput is one half-hour handover window attached at publish time.
type CreateSlotInput struct {
	StartTime time.Time `json:"start_time" binding:"required"`
}

// SlotView is a handover window returned to clients.
type SlotView struct {
	ID        uint      `json:"id"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Status    string    `json:"status"`
	Available bool      `json:"available"`
	Expired   bool      `json:"expired"`
}
