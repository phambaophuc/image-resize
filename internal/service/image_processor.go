package service

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

	"github.com/disintegration/imaging"
	"github.com/phambaophuc/image-resizing/internal/models"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

func (p *ImageProcessor) ProcessImage(
	file multipart.File,
	request *models.AdvancedProcessingRequest,
) (*bytes.Buffer, string, error) {
	// Decode image
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Reset file pointer
	file.Seek(0, 0)

	processedImg := img

	// Apply crop if specified
	if request.Crop != nil {
		processedImg = p.cropImage(processedImg, request.Crop)
	}

	// Apply resize if specified
	if request.Resize != nil {
		processedImg = p.resizeImage(processedImg, request.Resize)
	}

	// Apply watermark if specified
	if request.Watermark != nil {
		processedImg = p.addWatermark(processedImg, request.Watermark)
	}

	// Determine output format
	outputFormat := format
	if request.Resize != nil && request.Resize.Format != "" {
		outputFormat = request.Resize.Format
	}

	// Encode to buffer
	buffer := &bytes.Buffer{}
	if err := p.encodeImage(buffer, processedImg, outputFormat, p.getQuality(request)); err != nil {
		return nil, "", fmt.Errorf("failed to encode image: %w", err)
	}

	return buffer, outputFormat, nil
}

// BatchProcess handles multiple images concurrently
func (p *ImageProcessor) BatchProcess(files []multipart.File, requests []models.AdvancedProcessingRequest) []models.ProcessedImage {
	results := make([]models.ProcessedImage, len(files))

	// Process images concurrently
	jobs := make(chan int, len(files))

	// Worker pool
	numWorkers := 5
	if len(files) < numWorkers {
		numWorkers = len(files)
	}

	for w := 0; w < numWorkers; w++ {
		go func() {
			for i := range jobs {
				if i < len(files) && i < len(requests) {
					buffer, _, err := p.ProcessImage(files[i], &requests[i])
					if err != nil {
						results[i] = models.ProcessedImage{
							ID: fmt.Sprintf("error_%d", i),
							// Error handling
						}
						continue
					}

					results[i] = models.ProcessedImage{
						ID:       fmt.Sprintf("processed_%d", i),
						FileSize: int64(buffer.Len()),
						// Other fields will be populated by caller
					}
				}
			}
		}()
	}

	// Send jobs
	for i := range files {
		jobs <- i
	}
	close(jobs)

	return results
}

// ResizeImage resizes an image with high quality
func (p *ImageProcessor) resizeImage(img image.Image, req *models.ResizeRequest) image.Image {
	return imaging.Resize(img, req.Width, req.Height, imaging.Lanczos)
}

// CropImage crops an image
func (p *ImageProcessor) cropImage(img image.Image, req *models.CropRequest) image.Image {
	bounds := image.Rect(req.X, req.Y, req.X+req.Width, req.Y+req.Height)
	return imaging.Crop(img, bounds)
}

// AddWatermark adds watermark to image
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

	// Create text color with opacity
	textColor := color.RGBA{255, 255, 255, uint8(255 * opacity)}

	// Add text (simplified - in production you'd use a better font rendering)
	point := fixed.Point26_6{
		X: fixed.I(x),
		Y: fixed.I(y),
	}

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)
}

// EncodeImage encodes image to specified format
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

func (p *ImageProcessor) getQuality(req *models.AdvancedProcessingRequest) int {
	if req.Resize != nil && req.Resize.Quality > 0 {
		return req.Resize.Quality
	}
	return 85 // Default quality
}

// GetImageInfo returns basic information about an image
func (p *ImageProcessor) GetImageInfo(file multipart.File) (int, int, string, error) {
	img, format, err := image.Decode(file)
	if err != nil {
		return 0, 0, "", err
	}

	bounds := img.Bounds()
	return bounds.Dx(), bounds.Dy(), format, nil
}

// ValidateImage checks if the image is valid and within limits
func (p *ImageProcessor) ValidateImage(file multipart.File, maxSize int64) error {
	// Check file size
	file.Seek(0, 2)            // Seek to end
	size, _ := file.Seek(0, 0) // Get size and seek back to start

	if size > maxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", size, maxSize)
	}

	// Try to decode to ensure it's a valid image
	_, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("invalid image format: %w", err)
	}

	file.Seek(0, 0) // Reset for further processing
	return nil
}
