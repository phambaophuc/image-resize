package processor

import (
	"image"

	"github.com/disintegration/imaging"
	"github.com/phambaophuc/image-resize/internal/models"
)

func (p *ImageProcessor) cropImage(img image.Image, req *models.CropRequest) image.Image {
	bounds := image.Rect(req.X, req.Y, req.X+req.Width, req.Y+req.Height)
	return imaging.Crop(img, bounds)
}
