package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler handles panics and errors
func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered",
			zap.Any("panic", recovered),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("method", ctx.Request.Method),
		)

		ctx.JSON(500, gin.H{
			"success": false,
			"error":   "Internal server error",
		})
	})
}
