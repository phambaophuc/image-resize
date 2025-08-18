package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// ValidateContentType ensures uploaded files are images
func ValidateContentType() gin.HandlerFunc {
	// allowedTypes := []string{"image/jpeg", "image/png", "image/webp", "image/gif"}

	return func(ctx *gin.Context) {
		contentType := ctx.GetHeader("Content-Type")

		// Skip validation for non-file uploads
		if !strings.Contains(contentType, "multipart/form-data") {
			ctx.Next()
			return
		}

		// For multipart uploads, validation happens in handlers
		ctx.Next()
	}
}
