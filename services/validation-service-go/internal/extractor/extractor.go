package extractor

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/iicpc/pkg/security"
	"github.com/iicpc/validation-service-go/internal/domain"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

// Extractor handles secure ZIP extraction with safety guards.
type Extractor struct {
	maxBytes int64
	maxFiles int
}

// NewExtractor creates an Extractor with the given size and file count limits.
func NewExtractor(maxBytes int64, maxFiles int) *Extractor {
	return &Extractor{
		maxBytes: maxBytes,
		maxFiles: maxFiles,
	}
}

// ExtractResult holds the extraction outcome.
type ExtractResult struct {
	RootDir   string   // Absolute path to the extracted root directory
	FileCount int      // Total number of files extracted
	TotalSize int64    // Total bytes extracted
	Files     []string // Relative paths of all extracted files
}

// Extract unpacks a ZIP file to a temp directory and returns the root path.
// It enforces: path traversal prevention, symlink blocking, size bomb detection,
// and file count limits. The caller is responsible for cleaning up RootDir.
func (e *Extractor) Extract(zipPath string) (*ExtractResult, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open zip: %v", domain.ErrExtractionFailed, err)
	}
	defer reader.Close()

	limits := security.DefaultArchiveLimits()
	limits.MaxFiles = e.maxFiles
	limits.MaxUncompressedBytes = e.maxBytes
	if _, err := security.ValidateZipReader(&reader.Reader, limits); err != nil {
		return nil, mapArchiveSecurityError(err)
	}

	// Create temp directory for extraction.
	tempDir, err := os.MkdirTemp("", "iicpc-validation-*")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create temp dir: %v", domain.ErrExtractionFailed, err)
	}

	result := &ExtractResult{
		RootDir: tempDir,
		Files:   make([]string, 0, len(reader.File)),
	}

	for _, file := range reader.File {
		if err := e.extractFile(file, tempDir, result); err != nil {
			// Clean up on failure.
			os.RemoveAll(tempDir)
			return nil, err
		}
	}

	// Detect if all files are nested in a single top-level directory
	// (common when zipping a directory: trade-engine/src/main.cpp instead of src/main.cpp).
	result.RootDir = detectActualRoot(tempDir)

	return result, nil
}

// extractFile extracts a single file from the archive with all safety checks.
func (e *Extractor) extractFile(file *zip.File, destDir string, result *ExtractResult) error {
	limits := security.DefaultArchiveLimits()
	limits.MaxFiles = e.maxFiles
	limits.MaxUncompressedBytes = e.maxBytes

	cleanName, err := security.SafeArchivePath(file.Name, limits)
	if err != nil {
		return fmt.Errorf("%w: entry '%s': %v", domain.ErrPathTraversal, file.Name, err)
	}
	if err := security.ValidateArchiveEntry(cleanName, file.Mode()); err != nil {
		return fmt.Errorf("%w: entry '%s': %v", domain.ErrDangerousFile, file.Name, err)
	}

	destPath, err := security.SafeJoin(destDir, cleanName)
	if err != nil {
		return fmt.Errorf("%w: entry '%s' resolves outside destination", domain.ErrPathTraversal, file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, 0755)
	}

	// Size bomb detection.
	if result.TotalSize+int64(file.UncompressedSize64) > e.maxBytes {
		return fmt.Errorf("%w: would exceed %d bytes", domain.ErrSizeBomb, e.maxBytes)
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("%w: failed to create dir for '%s': %v", domain.ErrExtractionFailed, cleanName, err)
	}

	// Extract file with size-limited reader.
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("%w: failed to open entry '%s': %v", domain.ErrExtractionFailed, cleanName, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, safeExtractPerm(file.Mode()))
	if err != nil {
		return fmt.Errorf("%w: failed to create '%s': %v", domain.ErrExtractionFailed, cleanName, err)
	}
	defer dst.Close()

	// Use LimitedReader as a second defense against size bombs (in case
	// UncompressedSize64 was spoofed in the ZIP header).
	remaining := e.maxBytes - result.TotalSize
	limitedReader := io.LimitReader(src, remaining+1) // +1 to detect overflow

	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	written, err := io.CopyBuffer(dst, limitedReader, buf)
	if err != nil {
		return fmt.Errorf("%w: failed to write '%s': %v", domain.ErrExtractionFailed, cleanName, err)
	}

	if written > remaining {
		return fmt.Errorf("%w: would exceed %d bytes", domain.ErrSizeBomb, e.maxBytes)
	}

	result.TotalSize += written
	result.FileCount++
	result.Files = append(result.Files, cleanName)

	return nil
}

func mapArchiveSecurityError(err error) error {
	switch {
	case errors.Is(err, security.ErrTooManyArchiveFiles):
		return fmt.Errorf("%w: %v", domain.ErrTooManyFiles, err)
	case errors.Is(err, security.ErrArchiveTooLarge):
		return fmt.Errorf("%w: %v", domain.ErrSizeBomb, err)
	case errors.Is(err, security.ErrDangerousFile):
		return fmt.Errorf("%w: %v", domain.ErrDangerousFile, err)
	case errors.Is(err, security.ErrUnsafeArchiveEntry):
		return fmt.Errorf("%w: %v", domain.ErrPathTraversal, err)
	default:
		return fmt.Errorf("%w: %v", domain.ErrExtractionFailed, err)
	}
}

func safeExtractPerm(mode os.FileMode) os.FileMode {
	perm := mode.Perm()
	if perm == 0 {
		return 0600
	}
	perm &^= 0111
	return perm
}

// detectActualRoot handles the common case where all files are nested in a
// single top-level directory (e.g., trade-engine/src/main.cpp). If so, it
// returns that nested directory as the root instead of the temp dir.
func detectActualRoot(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return dir
	}
	return filepath.Join(dir, entries[0].Name())
}

// Cleanup removes the extracted directory tree.
func Cleanup(dir string) {
	if dir != "" {
		os.RemoveAll(dir)
	}
}
