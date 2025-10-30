package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/phambaophuc/image-resize/internal/models"
	"go.uber.org/zap"
)

// === REQUEST PARSING ===

func (h *ImageHandler) parseResizeParams(c *gin.Context) (*models.AdvancedProcessingRequest, error) {
	width, err := h.parsePositiveInt(c.PostForm("width"), "width")
	if err != nil {
		return nil, err
	}

	height, err := h.parsePositiveInt(c.PostForm("height"), "height")
	if err != nil {
		return nil, err
	}

	quality := h.parseQuality(c.PostForm("quality"))
	format := c.PostForm("format")

	return &models.AdvancedProcessingRequest{
		Resize: &models.ResizeRequest{
			Width:   width,
			Height:  height,
			Quality: quality,
			Format:  format,
		},
	}, nil
}

func (h *ImageHandler) parseAdvancedParams(c *gin.Context) (*models.AdvancedProcessingRequest, error) {
	jsonStr := c.PostForm("payload")
	if jsonStr == "" {
		return nil, fmt.Errorf("missing payload parameter")
	}

	var req models.AdvancedProcessingRequest
	if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
		return nil, fmt.Errorf("invalid processing request: %v", err)
	}

	return &req, nil
}

func (h *ImageHandler) parseMultipartFiles(c *gin.Context) ([]*multipart.FileHeader, error) {
	if err := c.Request.ParseMultipartForm(h.config.Storage.MaxFileSize * 10); err != nil {
		return nil, fmt.Errorf("failed to parse form data: %v", err)
	}

	files := c.Request.MultipartForm.File[imagesParamKey]
	if len(files) == 0 {
		return nil, fmt.Errorf("no images provided")
	}

	return files, nil
}

func (h *ImageHandler) parsePositiveInt(value, fieldName string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is required", fieldName)
	}

	num, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: must be a number", fieldName)
	}

	if num <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", fieldName)
	}

	return num, nil
}

func (h *ImageHandler) parseQuality(value string) int {
	if value == "" {
		return defaultQuality
	}

	quality, err := strconv.Atoi(value)
	if err != nil || quality < 1 || quality > 100 {
		return defaultQuality
	}

	return quality
}

// === FILE OPERATIONS ===

func (h *ImageHandler) getUploadedFile(c *gin.Context, paramKey string) (multipart.File, *multipart.FileHeader, error) {
	return c.Request.FormFile(paramKey)
}

func (h *ImageHandler) openFiles(files []*multipart.FileHeader) ([]multipart.File, error) {
	var openedFiles []multipart.File

	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			h.closeFiles(openedFiles)
			return nil, err
		}
		openedFiles = append(openedFiles, f)
	}

	return openedFiles, nil
}

func (h *ImageHandler) closeFiles(files []multipart.File) {
	for _, file := range files {
		if file != nil {
			file.Close()
		}
	}
}

// === RESPONSE HANDLING ===

func (h *ImageHandler) respondError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, models.APIResponse{
		Success: false,
		Error:   message,
	})
}

func (h *ImageHandler) respondWithURL(
	c *gin.Context,
	buffer *bytes.Buffer,
	header *multipart.FileHeader,
	format string,
	req *models.AdvancedProcessingRequest,
	img image.Image,
) {
	imageURL := h.uploadToStorage(c.Request.Context(), buffer, header, format)

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	size := models.ResizeSize{
		Width:   width,
		Height:  height,
		Quality: 0,
		Format:  format,
	}

	if req != nil && req.Resize != nil {
		size.Width = req.Resize.Width
		size.Height = req.Resize.Height
		size.Quality = req.Resize.Quality
		size.Format = req.Resize.Format
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data: models.ProcessedImage{
			ID:          uuid.New().String(),
			OriginalURL: header.Filename,
			URL:         imageURL,
			FileSize:    int64(buffer.Len()),
			ProcessedAt: time.Now(),
			Size:        size,
		},
	})
}

// === PROCESSING LOGIC ===

func (h *ImageHandler) processAndRespond(c *gin.Context, file multipart.File, header *multipart.FileHeader, req *models.AdvancedProcessingRequest) {
	if err := h.processor.ValidateImage(file, h.config.Storage.MaxFileSize); err != nil {
		h.respondError(c, http.StatusBadRequest, fmt.Sprintf("Invalid image: %v", err))
		return
	}

	// cacheKey := h.storage.GenerateCacheKey(header.Filename, req)
	// if cachedData, found := h.tryGetFromCache(c.Request.Context(), cacheKey); found {
	// 	h.respondWithImage(c, cachedData, req.Resize.Format)
	// 	return
	// }

	if _, err := file.Seek(0, 0); err != nil {
		h.logger.Error("Failed to reset file pointer", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, "Internal file error")
		return
	}

	buffer, format, processedImg, err := h.processor.ProcessImage(file, req)
	if err != nil {
		h.logger.Error("Processing failed", zap.Error(err))
		h.respondError(c, http.StatusInternalServerError, "Failed to process image")
		return
	}

	// Cache the result
	// h.setCacheData(c.Request.Context(), cacheKey, buffer.Bytes())

	h.respondWithURL(c, buffer, header, format, req, processedImg)
}

// === UTILITY METHODS ===

func (h *ImageHandler) calculateOverallHealth(services map[string]string) string {
	for _, status := range services {
		if status != "healthy" && status != "not configured" {
			return "unhealthy"
		}
	}
	return "healthy"
}

func (h *ImageHandler) generateNewFilename(originalFilename, format string) string {
	ext := "." + format
	return strings.TrimSuffix(originalFilename, filepath.Ext(originalFilename)) + ext
}

func (h *ImageHandler) buildBatchResponse(ctx context.Context, images []models.BatchImage, files []*multipart.FileHeader, format string) models.BatchResponse {
	var batchResponse models.BatchResponse

	for i, img := range images {
		if img.Buffer == nil {
			continue
		}

		url := h.uploadToStorage(ctx, img.Buffer, files[i], format)
		batchResponse.Images = append(batchResponse.Images, models.ImageResponse{
			URL:         url,
			FileSize:    img.FileSize,
			ProcessedAt: time.Now(),
		})
	}

	return batchResponse
}

// === STORAGE OPERATIONS ===

func (h *ImageHandler) uploadToStorage(ctx context.Context, buffer *bytes.Buffer, header *multipart.FileHeader, format string) string {
	if h.storage == nil {
		return ""
	}

	newFilename := h.generateNewFilename(header.Filename, format)
	url, err := h.storage.Upload(ctx, buffer, newFilename, "image/"+format)
	if err != nil {
		h.logger.Warn("Failed to upload to Storage", zap.Error(err))
		return ""
	}

	return url
}

// func (h *ImageHandler) tryGetFromCache(ctx context.Context, cacheKey string) ([]byte, bool) {
// 	cachedData, err := h.storage.GetFromCache(ctx, cacheKey)
// 	if err != nil || cachedData == nil {
// 		return nil, false
// 	}

// 	h.logger.Info("Cache hit", zap.String("cache_key", cacheKey))
// 	return cachedData, true
// }

// func (h *ImageHandler) setCacheData(ctx context.Context, cacheKey string, data []byte) {
// 	if err := h.storage.SetCache(ctx, cacheKey, data); err != nil {
// 		h.logger.Warn("Failed to cache data", zap.String("cache_key", cacheKey), zap.Error(err))
// 	}
// }
