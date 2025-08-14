package models

import "time"

type BatchResizeRequest struct {
	Sizes []ResizeSize `json:"sizes" binding:"required,min=1"`
}

type BatchResponse struct {
	JobID       string           `json:"job_id"`
	Status      string           `json:"status"`
	Images      []ProcessedImage `json:"images,omitempty"`
	ProcessedAt time.Time        `json:"processed_at,omitempty"`
	Error       string           `json:"error,omitempty"`
}
