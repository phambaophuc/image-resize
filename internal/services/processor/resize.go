package processor

import (
	"image"

	"github.com/disintegration/imaging"
	"github.com/phambaophuc/image-resize/internal/models"
)

func (p *ImageProcessor) resizeImage(img image.Image, req *models.ResizeRequest) image.Image {
	return imaging.Resize(img, req.Width, req.Height, imaging.Lanczos)
}
