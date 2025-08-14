package models

type WatermarkRequest struct {
	Text     string  `json:"text,omitempty"`
	ImageURL string  `json:"image_url,omitempty"`
	Position string  `json:"position" binding:"required,oneof=top-left top-right bottom-left bottom-right center"`
	Opacity  float64 `json:"opacity" binding:"min=0,max=1"`
}
