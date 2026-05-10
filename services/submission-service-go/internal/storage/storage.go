package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Storage defines the interface for object storage (e.g., S3, local disk).
type Storage interface {
	// Save stores a file and returns its URI/path.
	Save(ctx context.Context, key string, reader io.Reader) (string, error)
	// Get retrieves a file.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
}

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a LocalStorage saving to baseDir.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

func (s *LocalStorage) Save(ctx context.Context, key string, reader io.Reader) (string, error) {
	path := filepath.Join(s.baseDir, key)

	// Ensure subdirectories exist if key contains slashes.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	out, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer out.Close()

	if _, err := io.Copy(out, reader); err != nil {
		return "", err
	}

	return path, nil
}

func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path := filepath.Join(s.baseDir, key)
	return os.Open(path)
}
