package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStorageClient implements service.StorageClient by reading directly from a shared filesystem.
type LocalStorageClient struct {
}

func NewLocalStorageClient() *LocalStorageClient {
	return &LocalStorageClient{}
}

// DownloadArchive copies the file from the shared storage path to the local destination path.
func (c *LocalStorageClient) DownloadArchive(ctx context.Context, storagePath string, destinationPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src, err := os.Open(storagePath)
	if err != nil {
		return fmt.Errorf("failed to open source archive at %s: %w", storagePath, err)
	}
	defer src.Close()

	if err := os.MkdirAll(filepath.Dir(destinationPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	dst, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file at %s: %w", destinationPath, err)
	}
	defer dst.Close()

	if err := copyWithContext(ctx, dst, src); err != nil {
		return fmt.Errorf("failed to copy archive: %w", err)
	}

	return nil
}

func copyWithContext(ctx context.Context, dst *os.File, src *os.File) error {
	buf := make([]byte, 256*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			written, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
			if written != n {
				return io.ErrShortWrite
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}
