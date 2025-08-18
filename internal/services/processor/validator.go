package processor

import (
	"fmt"
	"image"
	"mime/multipart"
)

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
