package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/phambaophuc/image-resize/internal/config"
	"github.com/phambaophuc/image-resize/internal/models"
	"github.com/phambaophuc/image-resize/internal/services"
	"go.uber.org/zap"
)

const (
	defaultQuality = 75
	maxCacheAge    = 3600
	imageParamKey  = "image"
	imagesParamKey = "images"
)

type ImageHandler struct {
	processor *services.ImageProcessor
	storage   *services.StorageService
	logger    *zap.Logger
	config    *config.Config
}

func NewImageHandler(
	processor *services.ImageProcessor,
	storage *services.StorageService,
	logger *zap.Logger,
	config *config.Config,
) *ImageHandler {
	return &ImageHandler{
		processor: processor,
		storage:   storage,
		logger:    logger,
		config:    config,
	}
}

// === MAIN API ENDPOINTS ===

func (h *ImageHandler) ResizeImage(c *gin.Context) {
	file, header, err := h.getUploadedFile(c, imageParamKey)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "No image file provided")
		return
	}
	defer file.Close()

	req, err := h.parseResizeParams(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.processAndRespond(c, file, header, req)
}

func (h *ImageHandler) AdvancedProcess(c *gin.Context) {
	file, header, err := h.getUploadedFile(c, imageParamKey)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, "No image file provided")
		return
	}
	defer file.Close()

	req, err := h.parseAdvancedParams(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	h.processAndRespond(c, file, header, req)
}

func (h *ImageHandler) BatchResize(c *gin.Context) {
	files, err := h.parseMultipartFiles(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	req, err := h.parseResizeParams(c)
	if err != nil {
		h.respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	openedFiles, err := h.openFiles(files)
	if err != nil {
		h.respondError(c, http.StatusInternalServerError, "Failed to open files: "+err.Error())
		return
	}
	defer h.closeFiles(openedFiles)

	images := h.processor.BatchResize(openedFiles, req)
	response := h.buildBatchResponse(c.Request.Context(), images, files, req.Resize.Format)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    response,
	})
}

// HealthCheck
func (h *ImageHandler) HealthCheck(c *gin.Context) {
	storageStatus := h.storage.HealthCheck(c.Request.Context())
	overall := h.calculateOverallHealth(storageStatus)

	statusCode := http.StatusOK
	if overall == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, models.APIResponse{
		Success: overall == "healthy",
		Data: models.HealthCheck{
			Status:    overall,
			Timestamp: time.Now(),
			Services:  storageStatus,
		},
	})
}

// func (h *ImageHandler) GetStats(c *gin.Context) {
// 	cacheStats, err := h.storage.GetCacheStats(c.Request.Context())
// 	if err != nil {
// 		h.logger.Error("Failed to get cache stats", zap.Error(err))
// 	}

// 	stats := map[string]interface{}{
// 		"cache":     cacheStats,
// 		"timestamp": time.Now(),
// 	}

// 	c.JSON(http.StatusOK, models.APIResponse{
// 		Success: true,
// 		Data:    stats,
// 	})
// }
