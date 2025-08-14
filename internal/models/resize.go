package models

type ResizeRequest struct {
	Width   int    `json:"width" binding:"required,min=1"`
	Height  int    `json:"height" binding:"required,min=1"`
	Quality int    `json:"quality" binding:"min=1,max=100"`
	Format  string `json:"format" binding:"omitempty,oneof=jpeg png webp"`
}

type ResizeSize struct {
	Name    string `json:"name" binding:"required"`
	Width   int    `json:"width" binding:"required,min=1"`
	Height  int    `json:"height" binding:"required,min=1"`
	Quality int    `json:"quality" binding:"min=1,max=100"`
	Format  string `json:"format" binding:"omitempty,oneof=jpeg png webp"`
}

const (
	FormatJPEG = "jpeg"
	FormatPNG  = "png"
	FormatWebP = "webp"
)
