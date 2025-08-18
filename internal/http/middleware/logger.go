package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(params gin.LogFormatterParams) string {
		logger.Info("HTTP Request",
			zap.String("method", params.Method),
			zap.String("path", params.Path),
			zap.Int("status", params.StatusCode),
			zap.Duration("latency", params.Latency),
			zap.String("client_ip", params.ClientIP),
			zap.String("user_agent", params.Request.UserAgent()),
		)
		return ""
	})
}
