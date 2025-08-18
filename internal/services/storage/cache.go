package storage

import (
	"context"
	"crypto/md5"
	"fmt"

	"github.com/phambaophuc/image-resize/internal/models"
	"github.com/redis/go-redis/v9"
)

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

func (s *StorageService) SetCache(ctx context.Context, cacheKey string, data []byte) error {
	return s.redisClient.Set(ctx, cacheKey, data, s.cacheDuration).Err()
}

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
