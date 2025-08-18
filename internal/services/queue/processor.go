package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/phambaophuc/image-resize/internal/models"
	"github.com/phambaophuc/image-resize/pkg/utils"
	"go.uber.org/zap"
)

func (q *QueueService) processJob(ctx context.Context, job *models.ProcessingJob) (*models.ProcessedImage, error) {
	// Generate cache key
	cacheKey := q.storage.GenerateCacheKey(job.ImageURL, &job.Request)

	// Check cache first
	cachedData, err := q.storage.GetFromCache(ctx, cacheKey)
	if err == nil && cachedData != nil {
		var cachedResult models.ProcessedImage
		if err := json.Unmarshal(cachedData, &cachedResult); err == nil {
			return &cachedResult, nil
		}
		q.logger.Warn("Failed to unmarshal cached data", zap.Error(err))
	}

	// Download image from URL
	imageData, contentType, err := utils.DownloadImage(ctx, job.ImageURL, 10*1024*1024)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Process the image using ImageProcessor
	processedBuffer, processedContentType, err := q.processor.ProcessImage(
		&readerFile{
			reader:      bytes.NewReader(imageData),
			size:        int64(len(imageData)),
			contentType: contentType,
		},
		&job.Request,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	// Generate filename and save processed image
	filename := utils.GenerateFilename(job.ID, job.Request.Resize.Format)
	processedURL, err := q.storage.SaveFile(ctx, processedBuffer.Bytes(), filename, processedContentType)
	if err != nil {
		return nil, fmt.Errorf("failed to save processed image: %w", err)
	}

	// Create result
	result := &models.ProcessedImage{
		ID:          job.ID,
		OriginalURL: job.ImageURL,
		ProcessedAt: time.Now(),
		Size: models.ResizeSize{
			Width:  job.Request.Resize.Width,
			Height: job.Request.Resize.Height,
		},
		URL:      processedURL,
		FileSize: int64(processedBuffer.Len()),
	}

	// Cache the result
	resultBytes, _ := json.Marshal(result)
	if err := q.storage.SetCache(ctx, cacheKey, resultBytes); err != nil {
		q.logger.Warn("Failed to cache result", zap.Error(err))
	}

	return result, nil
}
