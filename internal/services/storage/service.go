package storage

import (
	"time"

	"github.com/phambaophuc/image-resize/internal/config"
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
