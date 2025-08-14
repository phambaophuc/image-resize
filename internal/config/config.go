package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Supabase SupabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Storage  StorageConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type SupabaseConfig struct {
	URL    string
	KEY    string
	BUCKET string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RabbitMQConfig struct {
	URL string
}

type StorageConfig struct {
	MaxFileSize   int64
	AllowedTypes  []string
	UploadPath    string
	CacheDuration time.Duration
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getDuration("READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDuration("WRITE_TIMEOUT", 10*time.Second),
		},
		Supabase: SupabaseConfig{
			URL:    getEnv("SUPABASE_URL", ""),
			KEY:    getEnv("SUPABASE_KEY", ""),
			BUCKET: getEnv("SUPABASE_BUCKET", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		},
		Storage: StorageConfig{
			MaxFileSize:   getEnvAsInt64("MAX_FILE_SIZE", 10*1024*1024), // 10MB
			AllowedTypes:  []string{"image/jpeg", "image/png", "image/webp"},
			UploadPath:    getEnv("UPLOAD_PATH", "./uploads"),
			CacheDuration: getDuration("CACHE_DURATION", 24*time.Hour),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvAsInt64(key string, defaultVal int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getDuration(key string, defaultVal time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultVal
}
