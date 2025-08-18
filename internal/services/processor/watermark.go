package processor

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/phambaophuc/image-resize/internal/models"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func (p *ImageProcessor) addWatermark(img image.Image, req *models.WatermarkRequest) image.Image {
	bounds := img.Bounds()
	watermarked := image.NewRGBA(bounds)
	draw.Draw(watermarked, bounds, img, bounds.Min, draw.Src)

	if req.Text != "" {
		p.addTextWatermark(watermarked, req.Text, req.Position, req.Opacity)
	}

	return watermarked
}

func (p *ImageProcessor) addTextWatermark(img *image.RGBA, text, position string, opacity float64) {
	bounds := img.Bounds()

	// Calculate position
	var x, y int
	switch position {
	case "top-left":
		x, y = 10, 30
	case "top-right":
		x, y = bounds.Dx()-100, 30
	case "bottom-left":
		x, y = 10, bounds.Dy()-10
	case "bottom-right":
		x, y = bounds.Dx()-100, bounds.Dy()-10
	case "center":
		x, y = bounds.Dx()/2-50, bounds.Dy()/2
	}

	// Add text (simplified - in production you'd use a better font rendering)
	textColor := image.NewUniform(color.RGBA{200, 200, 200, uint8(255 * opacity)})

	d := &font.Drawer{
		Dst:  img,
		Src:  textColor,
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(text)
}
