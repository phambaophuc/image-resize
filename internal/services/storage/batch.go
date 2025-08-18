package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/phambaophuc/image-resize/internal/models"
)

func (s *StorageService) UploadMultiple(ctx context.Context, files []models.UploadFile) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}

	urls := make([]string, len(files))
	errors := make([]error, len(files))

	numWorkers := 5
	if len(files) < numWorkers {
		numWorkers = len(files)
	}

	jobs := make(chan int, len(files))
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				buffer := bytes.NewBuffer(files[i].Data)
				url, err := s.Upload(ctx, buffer, files[i].Filename, files[i].ContentType)
				urls[i] = url
				errors[i] = err
			}
		}()
	}

	for i := range files {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	var failedUploads []string
	successUrls := make([]string, 0, len(files))

	for i, err := range errors {
		if err != nil {
			failedUploads = append(failedUploads, fmt.Sprintf("file %d: %v", i, err))
		} else {
			successUrls = append(successUrls, urls[i])
		}
	}

	if len(failedUploads) > 0 {
		return successUrls, fmt.Errorf("failed to upload %d files: %s",
			len(failedUploads), strings.Join(failedUploads, "; "))
	}

	return urls, nil
}
