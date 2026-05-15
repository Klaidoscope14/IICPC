package storage

import (
	"context"
	"fmt"
	"io"
	"os"
)

// LocalStorageClient implements service.StorageClient by reading directly from a shared filesystem.
type LocalStorageClient struct {
}

func NewLocalStorageClient() *LocalStorageClient {
	return &LocalStorageClient{}
}

// DownloadArchive copies the file from the shared storage path to the local destination path.
func (c *LocalStorageClient) DownloadArchive(ctx context.Context, storagePath string, destinationPath string) error {
	src, err := os.Open(storagePath)
	if err != nil {
		return fmt.Errorf("failed to open source archive at %s: %w", storagePath, err)
	}
	defer src.Close()

	dst, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file at %s: %w", destinationPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy archive: %w", err)
	}

	return nil
}
