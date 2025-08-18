package storage

import (
	"context"
	"fmt"

	storage_go "github.com/supabase-community/storage-go"
)

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
