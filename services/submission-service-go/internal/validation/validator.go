package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/iicpc/submission-service-go/internal/domain"
)

// zipMagicBytes is the ZIP file signature (PK\x03\x04).
var zipMagicBytes = []byte{0x50, 0x4B, 0x03, 0x04}

// safeFilenameRegex allows only alphanumeric, underscores, hyphens, and dots.
var safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)

// defaultAllowedMIMETypes is the set of MIME types accepted for uploads.
var defaultAllowedMIMETypes = map[string]bool{
	"application/zip":                true,
	"application/x-zip-compressed":   true,
	"application/octet-stream":       true, // Many ZIP files are detected as this
	"application/x-zip":              true,
	"multipart/x-zip":                true,
}

// UploadValidator validates uploaded files using streaming checks.
// All checks operate on io.Reader — never buffering the full file.
type UploadValidator struct {
	maxUploadBytes   int64
	allowedMIMETypes map[string]bool
}

// NewUploadValidator creates a validator with the given size limit and optional MIME overrides.
func NewUploadValidator(maxUploadBytes int64, allowedMIME map[string]bool) *UploadValidator {
	if allowedMIME == nil {
		allowedMIME = defaultAllowedMIMETypes
	}
	return &UploadValidator{
		maxUploadBytes:   maxUploadBytes,
		allowedMIMETypes: allowedMIME,
	}
}

// ValidateFileSize checks if the declared file size is within limits.
// This is a pre-flight check using Content-Length / multipart header, before reading any bytes.
func (v *UploadValidator) ValidateFileSize(fileSize int64) error {
	if fileSize > v.maxUploadBytes {
		return fmt.Errorf("%w: %d bytes exceeds maximum of %d bytes",
			domain.ErrFileTooLarge, fileSize, v.maxUploadBytes)
	}
	return nil
}

// ValidateAndHash reads the header bytes from the reader to validate ZIP magic bytes and MIME type,
// then streams the remainder through a SHA-256 hasher into the destination writer.
// Returns the hex-encoded checksum and total bytes written.
//
// This is a single-pass operation: validate + hash + write in one stream.
func (v *UploadValidator) ValidateAndHash(src io.Reader, dst io.Writer) (checksum string, bytesWritten int64, err error) {
	// Read the first 512 bytes for validation (ZIP magic + MIME sniffing).
	// 512 bytes is what net/http.DetectContentType requires.
	header := make([]byte, 512)
	n, err := io.ReadAtLeast(src, header, len(zipMagicBytes))
	if err != nil {
		return "", 0, fmt.Errorf("%w: file too small or unreadable", domain.ErrInvalidArchive)
	}
	header = header[:n]

	// Check ZIP magic bytes (PK\x03\x04).
	if len(header) < 4 || header[0] != zipMagicBytes[0] || header[1] != zipMagicBytes[1] ||
		header[2] != zipMagicBytes[2] || header[3] != zipMagicBytes[3] {
		return "", 0, fmt.Errorf("%w: file does not have ZIP signature", domain.ErrInvalidArchive)
	}

	// MIME sniff using the header bytes.
	detectedMIME := http.DetectContentType(header)
	if !v.allowedMIMETypes[detectedMIME] {
		return "", 0, fmt.Errorf("%w: detected '%s'", domain.ErrUnsupportedMIME, detectedMIME)
	}

	// Set up streaming pipeline: hash the entire content while writing to dst.
	hasher := sha256.New()
	multiWriter := io.MultiWriter(dst, hasher)

	// Write the header bytes we already read.
	headerWritten, err := multiWriter.Write(header)
	if err != nil {
		return "", 0, fmt.Errorf("failed to write header bytes: %w", err)
	}

	// Stream the rest of the file through hash + destination.
	remaining, err := io.Copy(multiWriter, src)
	if err != nil {
		return "", 0, fmt.Errorf("failed to stream file: %w", err)
	}

	totalBytes := int64(headerWritten) + remaining
	checksumHex := hex.EncodeToString(hasher.Sum(nil))

	return checksumHex, totalBytes, nil
}

// SanitizeFilename cleans a filename to prevent path traversal and injection attacks.
// It strips directory components, removes Unicode control characters,
// and collapses the name to [a-zA-Z0-9_.-] characters.
func SanitizeFilename(filename string) string {
	// Strip any directory path components.
	filename = filepath.Base(filename)

	// Remove Unicode control characters.
	filename = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, filename)

	// Replace unsafe characters with underscores.
	filename = safeFilenameRegex.ReplaceAllString(filename, "_")

	// Collapse consecutive underscores.
	for strings.Contains(filename, "__") {
		filename = strings.ReplaceAll(filename, "__", "_")
	}

	// Trim leading/trailing underscores and dots (prevent hidden files / empty names).
	filename = strings.Trim(filename, "_.")

	// Fallback if the filename is empty after sanitization.
	if filename == "" {
		filename = "submission"
	}

	return filename
}
