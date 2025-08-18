package processor

import (
	"bytes"
	"fmt"
	"image"
	"mime/multipart"

	"github.com/phambaophuc/image-resize/internal/models"
)

type ImageProcessor struct{}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{}
}

func (p *ImageProcessor) ProcessImage(file multipart.File, request *models.AdvancedProcessingRequest) (*bytes.Buffer, string, error) {
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

func (p *ImageProcessor) getQuality(req *models.AdvancedProcessingRequest) int {
	if req.Resize != nil && req.Resize.Quality > 0 {
		return req.Resize.Quality
	}
	return 85 // Default quality
}
