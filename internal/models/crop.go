package models

type CropRequest struct {
	X      int `json:"x" binding:"min=0"`
	Y      int `json:"y" binding:"min=0"`
	Width  int `json:"width" binding:"required,min=1"`
	Height int `json:"height" binding:"required,min=1"`
}
