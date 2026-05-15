package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Storage defines the interface for object storage (e.g., S3, local disk).
type Storage interface {
	// Save stores a file and returns its URI/path.
	Save(ctx context.Context, key string, reader io.Reader) (string, error)
	// Get retrieves a file.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes a file from storage.
	Delete(ctx context.Context, key string) error
	// Exists checks whether a file exists in storage.
	Exists(ctx context.Context, key string) (bool, error)
}

// LocalStorage implements Storage using the local filesystem.
type LocalStorage struct {
	baseDir string
}

// NewLocalStorage creates a LocalStorage saving to baseDir.
func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage directory: %w", err)
	}
	if err := os.MkdirAll(absBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}
	return &LocalStorage{baseDir: absBaseDir}, nil
}

func (s *LocalStorage) Save(ctx context.Context, key string, reader io.Reader) (string, error) {
	path, err := s.resolvePath(key)
	if err != nil {
		return "", err
	}

	// Ensure subdirectories exist if key contains slashes.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	committed := false
	defer func() {
		_ = tmp.Close()
		if !committed {
			_ = os.Remove(tmpPath)
		}
	}()

	buffer := make([]byte, 256*1024)
	if _, err := io.CopyBuffer(tmp, &contextReader{ctx: ctx, reader: reader}, buffer); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return "", err
	}
	committed = true

	return path, nil
}

func (s *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.resolvePath(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *LocalStorage) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.resolvePath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

func (s *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	path, err := s.resolvePath(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check file existence: %w", err)
}

func (s *LocalStorage) resolvePath(key string) (string, error) {
	cleanKey := filepath.Clean(key)
	if cleanKey == "." || cleanKey == ".." || strings.HasPrefix(cleanKey, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key: %s", key)
	}

	path := filepath.Join(s.baseDir, cleanKey)
	if filepath.IsAbs(cleanKey) {
		path = cleanKey
	}
	rel, err := filepath.Rel(s.baseDir, path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve storage path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("storage key escapes base directory: %s", key)
	}

	return path, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}
