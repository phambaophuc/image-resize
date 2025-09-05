package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"sync"

	"github.com/disintegration/imaging"
	"github.com/phambaophuc/image-resize/internal/models"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	DefaultQuality   = 85
	DefaultWorkers   = 5
	WatermarkPadding = 10
	MaxFileSize      = 10 << 20 // 10MB
)

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

// ProcessImage handles single image processing with all operations
func (p *ImageProcessor) ProcessImage(file multipart.File, request *models.AdvancedProcessingRequest) (*bytes.Buffer, string, error) {
	// Validate image first
	if err := p.ValidateImage(file, MaxFileSize); err != nil {
		return nil, "", err
	}

	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Apply all transformations in sequence
	processedImg := p.applyTransformations(img, request)

	// Determine output format
	outputFormat := p.getOutputFormat(format, request)

	// Encode and return
	buffer := &bytes.Buffer{}
	if err := p.encodeImage(buffer, processedImg, outputFormat, p.getQuality(request)); err != nil {
		return nil, "", fmt.Errorf("failed to encode image: %w", err)
	}

	return buffer, outputFormat, nil
}

// BatchResize processes multiple images concurrently
func (p *ImageProcessor) BatchResize(files []multipart.File, req *models.AdvancedProcessingRequest) []models.BatchImage {
	results := make([]models.BatchImage, len(files))
	jobs := make(chan int, len(files))

	// Determine optimal worker count
	numWorkers := DefaultWorkers
	if len(files) < numWorkers {
		numWorkers = len(files)
	}

	var wg sync.WaitGroup

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				p.processImageJob(i, files, req, results)
			}
		}()
	}

	// Queue jobs
	for i := range files {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	return results
}

// validateImage validates file size and format
func (p *ImageProcessor) ValidateImage(file multipart.File, maxSize int64) error {
	// Check file size
	if size := p.getFileSize(file); size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, maxSize)
	}

	// Validate image format
	if _, _, err := image.Decode(file); err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	p.resetFilePointer(file)
	return nil
}

// applyTransformations applies all image transformations in sequence
func (p *ImageProcessor) applyTransformations(img image.Image, request *models.AdvancedProcessingRequest) image.Image {
	result := img

	// Apply crop first (order matters)
	if request.Crop != nil {
		result = p.cropImage(result, request.Crop)
	}

	// Then resize
	if request.Resize != nil {
		result = p.resizeImage(result, request.Resize)
	}

	// Finally watermark
	if request.Watermark != nil {
		result = p.addWatermark(result, request.Watermark)
	}

	return result
}

// cropImage crops the image based on the crop request
func (p *ImageProcessor) cropImage(img image.Image, req *models.CropRequest) image.Image {
	bounds := img.Bounds()

	// Validate crop boundaries
	x := max(0, min(req.X, bounds.Dx()))
	y := max(0, min(req.Y, bounds.Dy()))
	width := min(req.Width, bounds.Dx()-x)
	height := min(req.Height, bounds.Dy()-y)

	cropBounds := image.Rect(x, y, x+width, y+height)
	return imaging.Crop(img, cropBounds)
}

// resizeImage resizes the image using Lanczos resampling
func (p *ImageProcessor) resizeImage(img image.Image, req *models.ResizeRequest) image.Image {
	// Validate dimensions
	width := max(1, req.Width)
	height := max(1, req.Height)

	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// addWatermark adds text watermark to the image
func (p *ImageProcessor) addWatermark(img image.Image, req *models.WatermarkRequest) image.Image {
	if req.Text == "" {
		return img
	}

	bounds := img.Bounds()
	watermarked := image.NewRGBA(bounds)
	draw.Draw(watermarked, bounds, img, bounds.Min, draw.Src)

	p.drawTextWatermark(watermarked, req)
	return watermarked
}

// drawTextWatermark draws text watermark at specified position
func (p *ImageProcessor) drawTextWatermark(img *image.RGBA, req *models.WatermarkRequest) {
	bounds := img.Bounds()

	// Calculate position using map for cleaner code
	positions := map[string]struct{ x, y int }{
		"top-left":     {WatermarkPadding, 30},
		"top-right":    {bounds.Dx() - 100, 30},
		"bottom-left":  {WatermarkPadding, bounds.Dy() - WatermarkPadding},
		"bottom-right": {bounds.Dx() - 100, bounds.Dy() - WatermarkPadding},
		"center":       {bounds.Dx()/2 - 50, bounds.Dy() / 2},
	}

	pos, exists := positions[req.Position]
	if !exists {
		pos = positions["bottom-right"] // Default position
	}

	// Draw text
	opacity := min(1.0, max(0.0, req.Opacity))
	textColor := image.NewUniform(color.RGBA{200, 200, 200, uint8(255 * opacity)})

	d := &font.Drawer{
		Dst:  img,
		Src:  textColor,
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(pos.x), Y: fixed.I(pos.y)},
	}
	d.DrawString(req.Text)
}

// encodeImage encodes image to specified format
func (p *ImageProcessor) encodeImage(w io.Writer, img image.Image, format string, quality int) error {
	switch format {
	case "jpeg", "jpg":
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	case "png":
		return png.Encode(w, img)
	case "webp":
		// For now, fallback to PNG (WebP support would require additional dependency)
		return png.Encode(w, img)
	default:
		return jpeg.Encode(w, img, &jpeg.Options{Quality: quality})
	}
}

// Helper functions
func (p *ImageProcessor) processImageJob(i int, files []multipart.File, req *models.AdvancedProcessingRequest, results []models.BatchImage) {
	if i >= len(files) {
		return
	}

	buffer, _, err := p.ProcessImage(files[i], req)
	if err != nil {
		results[i] = models.BatchImage{
			Error: fmt.Sprintf("failed to process image %d: %v", i, err),
		}
		return
	}

	results[i] = models.BatchImage{
		Buffer:   buffer,
		FileSize: int64(buffer.Len()),
	}
}

func (p *ImageProcessor) getFileSize(file multipart.File) int64 {
	current, _ := file.Seek(0, 1) // Get current position
	size, _ := file.Seek(0, 2)    // Seek to end to get size
	file.Seek(current, 0)         // Restore original position
	return size
}

func (p *ImageProcessor) resetFilePointer(file multipart.File) {
	file.Seek(0, 0)
}

func (p *ImageProcessor) getOutputFormat(originalFormat string, request *models.AdvancedProcessingRequest) string {
	if request.Resize != nil && request.Resize.Format != "" {
		return request.Resize.Format
	}
	return originalFormat
}

func (p *ImageProcessor) getQuality(req *models.AdvancedProcessingRequest) int {
	if req.Resize != nil && req.Resize.Quality > 0 {
		return min(100, max(1, req.Resize.Quality))
	}
	return DefaultQuality
}
