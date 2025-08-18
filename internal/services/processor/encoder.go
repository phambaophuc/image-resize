package processor

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"
)

func (p *ImageProcessor) encodeImage(w io.Writer, img image.Image, format string, quality int) error {
	switch format {
	case "jpeg", "jpg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	case "png":
		return png.Encode(w, img)
	case "webp":
		// Note: This requires the webp package to be properly imported
		// For simplicity, we'll fall back to PNG
		return png.Encode(w, img)
	default:
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	}
}
