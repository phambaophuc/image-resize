package models

import (
	"bytes"
	"time"
)

type BatchImage struct {
	Buffer   *bytes.Buffer
	Error    string
	FileSize int64
}

type ImageResponse struct {
	ProcessedAt time.Time `json:"processed_at"`
	URL         string    `json:"url"`
	FileSize    int64     `json:"file_size"`
}

type BatchResponse struct {
	Images      []ImageResponse `json:"images,omitempty"`
	ProcessedAt time.Time       `json:"processed_at,omitempty"`
	Error       string          `json:"error,omitempty"`
}
