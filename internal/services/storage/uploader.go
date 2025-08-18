package storage

import (
	"bytes"
	"context"
	"fmt"

	"github.com/phambaophuc/image-resize/pkg/utils"
)

func (s *StorageService) SaveFile(ctx context.Context, data []byte, filename, contentType string) (string, error) {
	buffer := bytes.NewBuffer(data)
	return s.Upload(ctx, buffer, filename, contentType)
}

// Upload uploads file to Supabase Storage
func (s *StorageService) Upload(ctx context.Context, buffer *bytes.Buffer, filename, contentType string) (string, error) {
	key := utils.GenerateStorageKey(filename)

	_, err := s.sbClient.UploadFile(s.bucket, key, bytes.NewReader(buffer.Bytes()))
	if err != nil {
		return "", fmt.Errorf("failed to upload to supabase: %w", err)
	}

	publicURL := s.sbClient.GetPublicUrl(s.bucket, key)
	return publicURL.SignedURL, nil
}

// Delete removes file from Supabase Storage
func (s *StorageService) Delete(ctx context.Context, path string) error {
	_, err := s.sbClient.RemoveFile(s.bucket, []string{path})
	return err
}
