package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/phambaophuc/image-resize/internal/http/handlers"
	"github.com/phambaophuc/image-resize/internal/http/middleware"
	"go.uber.org/zap"
)

type Router struct {
	imageHandler *handlers.ImageHandler
	logger       *zap.Logger
}

func NewRouter(
	imageHandler *handlers.ImageHandler,
	logger *zap.Logger,
) *Router {
	return &Router{
		imageHandler: imageHandler,
		logger:       logger,
	}
}

func (r *Router) SetupRoutes() *gin.Engine {
	if gin.Mode() == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	router.Use(middleware.Logger(r.logger))
	router.Use(middleware.ErrorHandler(r.logger))
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())

	// API version 1
	v1 := router.Group("/api/v1")
	{
		v1.GET("/health", r.imageHandler.HealthCheck)
		// v1.GET("/stats", r.imageHandler.GetStats)

		images := v1.Group("/images")
		{
			images.POST("/resize", r.imageHandler.ResizeImage)
			images.POST("/batch/resize", r.imageHandler.BatchResize)
			images.POST("/process", r.imageHandler.AdvancedProcess)
		}
	}

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"status":  "OK",
			"message": "Image resizing is running",
		})
	})

	return router
}
