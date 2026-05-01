package usecase

import (
	"context"
	"io"
	"time"

	"github.com/Nanyak/thangdq-lab/internal/entity"
)

// FileStorage defines the interface for cloud storage operations
type FileStorage interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) error
	Download(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]*entity.File, error)
	GetURL(ctx context.Context, key string, expiration time.Duration) (string, error)
}

// Storage wraps the FileStorage interface for business logic
type Storage struct {
	storage FileStorage
}

// NewStorage creates a new Storage instance
func NewStorage(storage FileStorage) *Storage {
	return &Storage{
		storage: storage,
	}
}

// Upload uploads a file to cloud storage
func (s *Storage) Upload(ctx context.Context, key string, reader io.Reader, contentType string) error {
	return s.storage.Upload(ctx, key, reader, contentType)
}

// Download downloads a file from cloud storage
func (s *Storage) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.storage.Download(ctx, key)
}

// Delete removes a file from cloud storage
func (s *Storage) Delete(ctx context.Context, key string) error {
	return s.storage.Delete(ctx, key)
}

// List returns files with the given prefix
func (s *Storage) List(ctx context.Context, prefix string) ([]*entity.File, error) {
	return s.storage.List(ctx, prefix)
}

// GetURL generates a presigned URL for file access
func (s *Storage) GetURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	return s.storage.GetURL(ctx, key, expiration)
}
