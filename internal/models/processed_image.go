package models

import "time"

type ProcessedImage struct {
	ID          string     `json:"id"`
	OriginalURL string     `json:"original_url"`
	ProcessedAt time.Time  `json:"processed_at"`
	Size        ResizeSize `json:"size"`
	URL         string     `json:"url"`
	FileSize    int64      `json:"file_size"`
}
