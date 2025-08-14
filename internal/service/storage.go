package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/phambaophuc/image-resizing/internal/config"
	"github.com/phambaophuc/image-resizing/internal/models"
	"github.com/redis/go-redis/v9"
	storage_go "github.com/supabase-community/storage-go"
)

type StorageService struct {
	sbClient      *storage_go.Client
	redisClient   *redis.Client
	bucket        string
	cacheDuration time.Duration
}

func NewStorageService(cfg *config.Config) (*StorageService, error) {
	sbClient := storage_go.NewClient(cfg.Supabase.URL+"/storage/v1", cfg.Supabase.KEY, nil)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	return &StorageService{
		sbClient:      sbClient,
		redisClient:   redisClient,
		bucket:        cfg.Supabase.BUCKET,
		cacheDuration: 24 * time.Hour,
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

// Download downloads file from Supabase Storage
func (s *StorageService) Download(ctx context.Context, path string) ([]byte, error) {
	data, err := s.sbClient.DownloadFile(s.bucket, path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Delete removes file from Supabase Storage
func (s *StorageService) Delete(ctx context.Context, path string) error {
	_, err := s.sbClient.RemoveFile(s.bucket, []string{path})
	return err
}

// GetFromCache retrieves processed image from cache
func (s *StorageService) GetFromCache(ctx context.Context, cacheKey string) ([]byte, error) {
	data, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("cache get error: %w", err)
	}
	return data, nil
}

// SetCache stores processed image in cache
func (s *StorageService) SetCache(ctx context.Context, cacheKey string, data []byte) error {
	return s.redisClient.Set(ctx, cacheKey, data, s.cacheDuration).Err()
}

// GenerateCacheKey generates a unique cache key for the image and processing parameters
func (s *StorageService) GenerateCacheKey(originalFilename string, request *models.AdvancedProcessingRequest) string {
	hash := md5.New()

	// Include original filename
	hash.Write([]byte(originalFilename))

	// Include processing parameters
	if request.Resize != nil {
		hash.Write([]byte(fmt.Sprintf("resize_%d_%d_%d_%s",
			request.Resize.Width, request.Resize.Height, request.Resize.Quality, request.Resize.Format)))
	}

	if request.Crop != nil {
		hash.Write([]byte(fmt.Sprintf("crop_%d_%d_%d_%d",
			request.Crop.X, request.Crop.Y, request.Crop.Width, request.Crop.Height)))
	}

	if request.Watermark != nil {
		hash.Write([]byte(fmt.Sprintf("watermark_%s_%s_%.2f",
			request.Watermark.Text, request.Watermark.Position, request.Watermark.Opacity)))
	}

	if request.Compress {
		hash.Write([]byte("compress"))
	}

	return fmt.Sprintf("img_cache:%x", hash.Sum(nil))
}

// CleanupCache removes expired cache entries (can be called periodically)
func (s *StorageService) CleanupCache(ctx context.Context) error {
	// This is a simplified cleanup - in production you might want more sophisticated cleanup
	keys, err := s.redisClient.Keys(ctx, "img_cache:*").Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		ttl := s.redisClient.TTL(ctx, key).Val()
		if ttl <= 0 {
			s.redisClient.Del(ctx, key)
		}
	}

	return nil
}

// GetCacheStats returns cache statistics
func (s *StorageService) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	info, err := s.redisClient.Info(ctx, "memory").Result()
	if err != nil {
		return nil, err
	}

	dbSize, err := s.redisClient.DBSize(ctx).Result()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"db_keys": dbSize,
		"info":    info,
	}

	return stats, nil
}

// HealthCheck checks Redis + Supabase
func (s *StorageService) HealthCheck(ctx context.Context) map[string]string {
	status := make(map[string]string)

	// Redis
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		status["redis"] = "unhealthy: " + err.Error()
	} else {
		status["redis"] = "healthy"
	}

	// Supabase Storage check
	_, err := s.sbClient.ListFiles(s.bucket, "", storage_go.FileSearchOptions{})
	if err != nil {
		fmt.Printf("Raw error: %#v\n", err) // In ra struct error
		status["supabase"] = "unhealthy: " + fmt.Sprintf("%#v", err)
	} else {
		status["supabase"] = "healthy"
	}

	return status
}

// generateStorageKey creates unique storage key
func (s *StorageService) generateStorageKey(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	timestamp := time.Now().Unix()
	uuid := uuid.New().String()[:8]

	return fmt.Sprintf("processed/%s_%d_%s%s", name, timestamp, uuid, ext)
}
