package models

type AdvancedProcessingRequest struct {
	Resize    *ResizeRequest    `json:"resize,omitempty"`
	Crop      *CropRequest      `json:"crop,omitempty"`
	Watermark *WatermarkRequest `json:"watermark,omitempty"`
	Compress  bool              `json:"compress,omitempty"`
}
