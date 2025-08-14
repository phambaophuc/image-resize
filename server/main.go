package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/phambaophuc/image-resizing/internal/config"
	"github.com/phambaophuc/image-resizing/internal/handlers"
	"github.com/phambaophuc/image-resizing/internal/service"
	"github.com/phambaophuc/image-resizing/server/routes"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize services
	processor := service.NewImageProcessor()

	storage, err := service.NewStorageService(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize storage service", zap.Error(err))
	}

	queue, err := service.NewQueueService(cfg.RabbitMQ.URL, processor, storage, logger)
	if err != nil {
		logger.Warn("Failed to initialize queue service", zap.Error(err))
		// Continue without queue service for basic functionality
	}

	// Initialize handlers
	imageHandler := handlers.NewImageHandler(processor, storage, queue, logger, cfg)

	router := routes.NewRouter(imageHandler, logger)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		Handler:      router.SetupRoutes(),
	}

	// Start server
	go func() {
		logger.Info("Starting server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
}
