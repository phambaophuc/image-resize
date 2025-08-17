package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/phambaophuc/image-resizing/internal/config"
	"github.com/phambaophuc/image-resizing/internal/models"
	"github.com/phambaophuc/image-resizing/internal/service"
	"go.uber.org/zap"
)

type ImageHandler struct {
	processor *service.ImageProcessor
	storage   *service.StorageService
	queue     *service.QueueService
	logger    *zap.Logger
	config    *config.Config
}

func NewImageHandler(
	processor *service.ImageProcessor,
	storage *service.StorageService,
	queue *service.QueueService,
	logger *zap.Logger,
	config *config.Config,
) *ImageHandler {
	return &ImageHandler{
		processor: processor,
		storage:   storage,
		queue:     queue,
		logger:    logger,
		config:    config,
	}
}

// ResizeImage handles single image resize
func (h *ImageHandler) ResizeImage(c *gin.Context) {
	file, header, err := h.getUploadedFile(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "No image file provided",
		})
		return
	}
	defer file.Close()

	width, _ := strconv.Atoi(c.PostForm("width"))
	height, _ := strconv.Atoi(c.PostForm("height"))
	quality, _ := strconv.Atoi(c.PostForm("quality"))
	format := c.PostForm("format")

	if width <= 0 || height <= 0 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Width and height must be positive integers",
		})
		return
	}

	if quality == 0 {
		quality = 85
	}

	req := &models.AdvancedProcessingRequest{
		Resize: &models.ResizeRequest{
			Width:   width,
			Height:  height,
			Quality: quality,
			Format:  format,
		},
	}

	h.processAndRespond(c, file, header, req)
}

// AdvancedProcess handles advanced image processing
func (h *ImageHandler) AdvancedProcess(c *gin.Context) {
	file, header, err := h.getUploadedFile(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "No image file provided",
		})
		return
	}
	defer file.Close()

	jsonStr := c.PostForm("payload")
	var req models.AdvancedProcessingRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Invalid processing request",
		})
		return
	}

	h.processAndRespond(c, file, header, &req)
}

func (h *ImageHandler) HealthCheck(c *gin.Context) {
	storageStatus := h.storage.HealthCheck(c.Request.Context())
	queueStatus := h.queue.HealthCheck()

	services := map[string]string{
		"queue": queueStatus,
	}

	// Add storage status
	for k, v := range storageStatus {
		services[k] = v
	}

	overall := "healthy"
	for _, status := range services {
		if status != "healthy" && status != "not configured" {
			overall = "unhealthy"
			break
		}
	}

	statusCode := http.StatusOK
	if overall == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, models.APIResponse{
		Success: overall == "healthy",
		Data: models.HealthCheck{
			Status:    overall,
			Timestamp: time.Now(),
			Services:  services,
		},
	})
}

// GetStats returns API statistics
func (h *ImageHandler) GetStats(c *gin.Context) {
	queueStats, err := h.queue.GetQueueStats()
	if err != nil {
		h.logger.Error("Failed to get queue stats", zap.Error(err))
	}

	cacheStats, err := h.storage.GetCacheStats(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get cache stats", zap.Error(err))
	}

	stats := map[string]interface{}{
		"queue":     queueStats,
		"cache":     cacheStats,
		"timestamp": time.Now(),
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    stats,
	})
}

func (h *ImageHandler) uploadToStorage(
	ctx context.Context,
	buffer *bytes.Buffer,
	header *multipart.FileHeader,
	format string,
) string {
	// Rename file
	ext := "." + format
	newFilename := strings.TrimSuffix(header.Filename, filepath.Ext(header.Filename)) + ext

	// Upload to Supabase if configured
	if h.storage != nil {
		url, err := h.storage.Upload(ctx, buffer, newFilename, "image/"+format)
		if err != nil {
			h.logger.Warn("Failed to upload to Storage", zap.Error(err))
			return ""
		}
		return url
	}

	return ""
}

func (h *ImageHandler) getUploadedFile(c *gin.Context) (multipart.File, *multipart.FileHeader, error) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		return nil, nil, err
	}
	return file, header, nil
}

func (h *ImageHandler) processAndRespond(
	c *gin.Context,
	file multipart.File,
	header *multipart.FileHeader,
	req *models.AdvancedProcessingRequest,
) {
	// Validate
	if err := h.processor.ValidateImage(file, h.config.Storage.MaxFileSize); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid image: %v", err),
		})
		return
	}

	// Cache key
	cacheKey := h.storage.GenerateCacheKey(header.Filename, req)
	cachedData, err := h.storage.GetFromCache(c.Request.Context(), cacheKey)
	if err == nil && cachedData != nil {
		h.logger.Info("Cache hit", zap.String("filename", header.Filename))
		c.Header("Content-Type", "image/"+req.Resize.Format)
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, "image/"+req.Resize.Format, cachedData)
		return
	}

	// Process
	buffer, format, err := h.processor.ProcessImage(file, req)
	if err != nil {
		h.logger.Error("Processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to process image",
		})
		return
	}

	// Cache result
	h.storage.SetCache(c.Request.Context(), cacheKey, buffer.Bytes())

	// Upload
	imageURL := h.uploadToStorage(c.Request.Context(), buffer, header, format)

	// Return processed image
	if c.Query("return_url") == "true" && imageURL != "" {
		c.JSON(http.StatusOK, models.APIResponse{
			Success: true,
			Data: models.ProcessedImage{
				ID:          uuid.New().String(),
				OriginalURL: header.Filename,
				URL:         imageURL,
				FileSize:    int64(buffer.Len()),
				ProcessedAt: time.Now(),
				Size: models.ResizeSize{
					Width:   req.Resize.Width,
					Height:  req.Resize.Height,
					Quality: req.Resize.Quality,
					Format:  format,
				},
			},
		})
	} else {
		c.Header("Content-Type", "image/"+format)
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, "image/"+format, buffer.Bytes())
	}
}
