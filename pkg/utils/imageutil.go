package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

func DownloadImage(ctx context.Context, imageURL string, maxSize int64) ([]byte, string, error) {
	fmt.Printf("ImageURL: %s", imageURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read image data: %w", err)
	}

	contentType := http.DetectContentType(imageData)
	if !IsValidImageType(contentType) {
		return nil, "", fmt.Errorf("invalid content type: %s", contentType)
	}

	if len(imageData) == 0 {
		return nil, "", fmt.Errorf("empty image data")
	}

	return imageData, contentType, nil
}

// IsValidImageType checks if content type is a valid image type
func IsValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
		"image/bmp",
		"image/tiff",
	}

	ct := strings.ToLower(contentType)
	for _, validType := range validTypes {
		if strings.Contains(ct, validType) {
			return true
		}
	}
	return false
}

// GenerateFilename generates a unique filename for processed image
func GenerateFilename(jobID, format string) string {
	timestamp := time.Now().Unix()
	if format == "" {
		format = "jpeg"
	}
	return fmt.Sprintf("processed_%s_%d.%s", jobID, timestamp, format)
}

func GenerateStorageKey(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	timestamp := time.Now().Unix()
	uuid := uuid.New().String()[:8]

	return fmt.Sprintf("processed/%s_%d_%s%s", name, timestamp, uuid, ext)
}
