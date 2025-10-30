package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phambaophuc/image-resize/internal/config"
	"github.com/phambaophuc/image-resize/internal/models"
	"github.com/redis/go-redis/v9"
	storage_go "github.com/supabase-community/storage-go"
)

type StorageService struct {
	sbClient      *storage_go.Client
	redisClient   *redis.Client
	bucket        string
	cacheDuration time.Duration
	maxRetries    int
	timeout       time.Duration
}

type ServiceOptions struct {
	CacheDuration time.Duration
	MaxRetries    int
	Timeout       time.Duration
}

var DefaultOptions = ServiceOptions{
	CacheDuration: 24 * time.Hour,
	MaxRetries:    3,
	Timeout:       30 * time.Second,
}

const (
	CacheKeyPrefix = "img_cache:"
)

func NewStorageService(cfg *config.Config, opts ...ServiceOptions) (*StorageService, error) {
	options := DefaultOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	sbClient := storage_go.NewClient(cfg.Supabase.URL+"/storage/v1", cfg.Supabase.KEY, nil)

	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     10, // Connection pool
		MinIdleConns: 5,
		MaxRetries:   options.MaxRetries,
		DialTimeout:  options.Timeout,
		ReadTimeout:  options.Timeout,
		WriteTimeout: options.Timeout,
	})

	return &StorageService{
		sbClient:      sbClient,
		redisClient:   redisClient,
		bucket:        cfg.Supabase.BUCKET,
		cacheDuration: options.CacheDuration,
		maxRetries:    options.MaxRetries,
		timeout:       options.Timeout,
	}, nil
}

// Upload uploads file to Supabase Storage
func (s *StorageService) Upload(ctx context.Context, buffer *bytes.Buffer, filename, contentType string) (string, error) {
	key := s.generateStorageKey(filename)

	_, err := s.sbClient.UploadFile(s.bucket, key, bytes.NewReader(buffer.Bytes()))
	if err != nil {
		return "", fmt.Errorf("failed to upload to supabase: %w", err)
	}

	publicURL := s.sbClient.GetPublicUrl(s.bucket, key)
	return publicURL.SignedURL, nil
}

func (s *StorageService) SetCache(ctx context.Context, cacheKey string, data []byte) error {
	return s.redisClient.Set(ctx, cacheKey, data, s.cacheDuration).Err()
}

func (s *StorageService) GetFromCache(ctx context.Context, cacheKey string) ([]byte, error) {
	data, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("cache get error: %w", err)
	}
	return data, nil
}

func (s *StorageService) GenerateCacheKey(originalFilename string, request *models.AdvancedProcessingRequest) string {
	var keyParts []string
	keyParts = append(keyParts, originalFilename)

	// Processing parameters - more efficient concatenation
	if request.Resize != nil {
		keyParts = append(keyParts, fmt.Sprintf("resize_%d_%d_%d_%s",
			request.Resize.Width, request.Resize.Height, request.Resize.Quality, request.Resize.Format))
	}

	if request.Crop != nil {
		keyParts = append(keyParts, fmt.Sprintf("crop_%d_%d_%d_%d",
			request.Crop.X, request.Crop.Y, request.Crop.Width, request.Crop.Height))
	}

	if request.Watermark != nil {
		keyParts = append(keyParts, fmt.Sprintf("watermark_%s_%s_%.2f",
			request.Watermark.Text, request.Watermark.Position, request.Watermark.Opacity))
	}

	combined := strings.Join(keyParts, "_")

	hash := sha256.Sum256([]byte(combined))
	return fmt.Sprintf("%s%x", CacheKeyPrefix, hash)
}

func (s *StorageService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	pipeline := s.redisClient.Pipeline()

	infoCmd := pipeline.Info(ctx, "memory")
	dbSizeCmd := pipeline.DBSize(ctx)

	_, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline error: %w", err)
	}

	stats := map[string]interface{}{
		"db_keys": dbSizeCmd.Val(),
		"info":    infoCmd.Val(),
	}

	return stats, nil
}

// HealthCheck checks Redis + Supabase
func (s *StorageService) HealthCheck(ctx context.Context) map[string]string {
	status := make(map[string]string)

	// Redis
	// if err := s.redisClient.Ping(ctx).Err(); err != nil {
	// 	status["redis"] = "unhealthy: " + err.Error()
	// } else {
	// 	status["redis"] = "healthy"
	// }

	// Supabase Storage check
	_, err := s.sbClient.ListFiles(s.bucket, "", storage_go.FileSearchOptions{})
	if err != nil {
		fmt.Printf("Raw error: %#v\n", err)
		status["supabase"] = "unhealthy: " + fmt.Sprintf("%#v", err)
	} else {
		status["supabase"] = "healthy"
	}

	return status
}

func (s *StorageService) generateStorageKey(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	timestamp := time.Now().Unix()
	uuid := uuid.New().String()[:8]

	return fmt.Sprintf("processed/%s_%d_%s%s", name, timestamp, uuid, ext)
}
