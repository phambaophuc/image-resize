package models

import "time"

type AdvancedProcessingRequest struct {
	Resize    *ResizeRequest    `json:"resize,omitempty"`
	Crop      *CropRequest      `json:"crop,omitempty"`
	Watermark *WatermarkRequest `json:"watermark,omitempty"`
	Compress  bool              `json:"compress,omitempty"`
}

type ProcessingJob struct {
	ID        string                    `json:"id"`
	ImageURL  string                    `json:"image_url"`
	Request   AdvancedProcessingRequest `json:"request"`
	Status    string                    `json:"status"`
	CreatedAt time.Time                 `json:"created_at"`
	Result    *ProcessedImage           `json:"result,omitempty"`
	Error     string                    `json:"error,omitempty"`
}

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)
