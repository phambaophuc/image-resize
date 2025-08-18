package storage

import "context"

func (s *StorageService) Download(ctx context.Context, path string) ([]byte, error) {
	data, err := s.sbClient.DownloadFile(s.bucket, path)
	if err != nil {
		return nil, err
	}
	return data, nil
}
