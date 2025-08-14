package handlers

import (
	"fmt"
	"net/http"
	"strconv"
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
	// Parse form data
	err := c.Request.ParseMultipartForm(h.config.Storage.MaxFileSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "Failed to parse form data",
		})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "No image file provided",
		})
		return
	}
	defer file.Close()

	// Validate file
	if err := h.processor.ValidateImage(file, h.config.Storage.MaxFileSize); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid image: %v", err),
		})
		return
	}

	// Parse resize parameters
	var resizeReq models.ResizeRequest
	if err := c.ShouldBindJSON(&resizeReq); err != nil {
		// Try to get from form parameters
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

		resizeReq = models.ResizeRequest{
			Width:   width,
			Height:  height,
			Quality: quality,
			Format:  format,
		}
	}

	// Set default quality if not provided
	if resizeReq.Quality == 0 {
		resizeReq.Quality = 85
	}

	// Create processing request
	processReq := models.AdvancedProcessingRequest{
		Resize: &resizeReq,
	}

	// Check cache first
	cacheKey := h.storage.GenerateCacheKey(header.Filename, &processReq)
	cachedData, err := h.storage.GetFromCache(c.Request.Context(), cacheKey)

	if err == nil && cachedData != nil {
		h.logger.Info("Cache hit", zap.String("filename", header.Filename))
		c.Header("Content-Type", "image/"+resizeReq.Format)
		c.Header("Cache-Control", "public, max-age=3600")
		c.Data(http.StatusOK, "image/"+resizeReq.Format, cachedData)
		return
	}

	// Process image
	buffer, format, err := h.processor.ProcessImage(file, &processReq)
	if err != nil {
		h.logger.Error("Image processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Error:   "Failed to process image",
		})
		return
	}

	// Cache the result
	h.storage.SetCache(c.Request.Context(), cacheKey, buffer.Bytes())

	// Upload to Supabase if configured
	var imageURL string
	if h.storage != nil {
		url, err := h.storage.Upload(c.Request.Context(), buffer, header.Filename, "image/"+format)
		if err != nil {
			h.logger.Warn("Failed to upload to S3", zap.Error(err))
		} else {
			imageURL = url
		}
	}

	// Return processed image or URL
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
					Width:   resizeReq.Width,
					Height:  resizeReq.Height,
					Quality: resizeReq.Quality,
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
