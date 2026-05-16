package security

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	slashpath "path"
	"path/filepath"
	"strings"
	"unicode"
)

var (
	ErrUnsafeArchiveEntry  = errors.New("unsafe archive entry")
	ErrDangerousFile       = errors.New("dangerous file blocked")
	ErrArchiveTooLarge     = errors.New("archive exceeds size limit")
	ErrTooManyArchiveFiles = errors.New("archive contains too many files")
)

type ArchiveLimits struct {
	MaxFiles             int
	MaxUncompressedBytes int64
	MaxEntryBytes        int64
	MaxCompressionRatio  float64
	MaxPathDepth         int
	MaxPathBytes         int
}

type ArchiveReport struct {
	FileCount              int
	TotalUncompressedBytes int64
	Files                  []string
}

func DefaultArchiveLimits() ArchiveLimits {
	return ArchiveLimits{
		MaxFiles:             1000,
		MaxUncompressedBytes: 500 * 1024 * 1024,
		MaxEntryBytes:        100 * 1024 * 1024,
		MaxCompressionRatio:  200,
		MaxPathDepth:         16,
		MaxPathBytes:         512,
	}
}

func (l ArchiveLimits) withDefaults() ArchiveLimits {
	defaults := DefaultArchiveLimits()
	if l.MaxFiles <= 0 {
		l.MaxFiles = defaults.MaxFiles
	}
	if l.MaxUncompressedBytes <= 0 {
		l.MaxUncompressedBytes = defaults.MaxUncompressedBytes
	}
	if l.MaxEntryBytes <= 0 {
		l.MaxEntryBytes = defaults.MaxEntryBytes
	}
	if l.MaxCompressionRatio <= 0 {
		l.MaxCompressionRatio = defaults.MaxCompressionRatio
	}
	if l.MaxPathDepth <= 0 {
		l.MaxPathDepth = defaults.MaxPathDepth
	}
	if l.MaxPathBytes <= 0 {
		l.MaxPathBytes = defaults.MaxPathBytes
	}
	return l
}

func ValidateZipReader(reader *zip.Reader, limits ArchiveLimits) (*ArchiveReport, error) {
	limits = limits.withDefaults()
	report := &ArchiveReport{Files: make([]string, 0, minInt(len(reader.File), limits.MaxFiles))}

	if len(reader.File) > limits.MaxFiles {
		return nil, fmt.Errorf("%w: %d entries exceeds %d", ErrTooManyArchiveFiles, len(reader.File), limits.MaxFiles)
	}

	for _, file := range reader.File {
		cleanName, err := SafeArchivePath(file.Name, limits)
		if err != nil {
			return nil, err
		}
		if err := ValidateArchiveEntry(cleanName, file.Mode()); err != nil {
			return nil, err
		}
		if file.FileInfo().IsDir() {
			continue
		}

		uncompressed := int64(file.UncompressedSize64)
		compressed := int64(file.CompressedSize64)
		if uncompressed > limits.MaxEntryBytes {
			return nil, fmt.Errorf("%w: %s is %d bytes", ErrArchiveTooLarge, cleanName, uncompressed)
		}
		if report.TotalUncompressedBytes+uncompressed > limits.MaxUncompressedBytes {
			return nil, fmt.Errorf("%w: total uncompressed bytes exceed %d", ErrArchiveTooLarge, limits.MaxUncompressedBytes)
		}
		if compressed == 0 && uncompressed > 0 {
			return nil, fmt.Errorf("%w: %s has suspicious compression metadata", ErrArchiveTooLarge, cleanName)
		}
		if compressed > 0 && float64(uncompressed)/float64(compressed) > limits.MaxCompressionRatio {
			return nil, fmt.Errorf("%w: %s compression ratio exceeds %.0f", ErrArchiveTooLarge, cleanName, limits.MaxCompressionRatio)
		}

		report.FileCount++
		report.TotalUncompressedBytes += uncompressed
		report.Files = append(report.Files, cleanName)
	}

	return report, nil
}

func SafeArchivePath(name string, limits ArchiveLimits) (string, error) {
	limits = limits.withDefaults()
	if name == "" || strings.ContainsRune(name, '\x00') {
		return "", fmt.Errorf("%w: empty or null path", ErrUnsafeArchiveEntry)
	}
	if len(name) > limits.MaxPathBytes {
		return "", fmt.Errorf("%w: path exceeds %d bytes", ErrUnsafeArchiveEntry, limits.MaxPathBytes)
	}

	normalized := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(normalized, "/") || hasWindowsDrivePrefix(normalized) {
		return "", fmt.Errorf("%w: absolute path %q", ErrUnsafeArchiveEntry, name)
	}
	for _, part := range strings.Split(normalized, "/") {
		if part == ".." {
			return "", fmt.Errorf("%w: traversal path %q", ErrUnsafeArchiveEntry, name)
		}
	}

	cleanName := slashpath.Clean(normalized)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("%w: traversal path %q", ErrUnsafeArchiveEntry, name)
	}

	parts := strings.Split(cleanName, "/")
	if len(parts) > limits.MaxPathDepth {
		return "", fmt.Errorf("%w: path depth exceeds %d", ErrUnsafeArchiveEntry, limits.MaxPathDepth)
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || hasControlRune(part) {
			return "", fmt.Errorf("%w: invalid path segment %q", ErrUnsafeArchiveEntry, part)
		}
		if len(part) > 255 {
			return "", fmt.Errorf("%w: path segment too long", ErrUnsafeArchiveEntry)
		}
	}

	return cleanName, nil
}

func ValidateArchiveEntry(cleanName string, mode os.FileMode) error {
	if mode&os.ModeSymlink != 0 || mode&os.ModeDevice != 0 || mode&os.ModeNamedPipe != 0 || mode&os.ModeSocket != 0 {
		return fmt.Errorf("%w: special file %s", ErrDangerousFile, cleanName)
	}

	parts := strings.Split(strings.ToLower(cleanName), "/")
	for _, part := range parts {
		switch part {
		case ".git", ".hg", ".svn", ".ssh", "__macosx":
			return fmt.Errorf("%w: blocked directory %s", ErrDangerousFile, part)
		}
	}

	base := slashpath.Base(parts[len(parts)-1])
	switch base {
	case ".ds_store", ".env", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "authorized_keys", "known_hosts":
		return fmt.Errorf("%w: blocked file %s", ErrDangerousFile, base)
	}
	if strings.HasPrefix(base, ".env.") {
		return fmt.Errorf("%w: blocked env file %s", ErrDangerousFile, base)
	}
	switch slashpath.Ext(base) {
	case ".key", ".pem", ".p12", ".pfx", ".jks":
		return fmt.Errorf("%w: blocked secret file %s", ErrDangerousFile, base)
	}

	return nil
}

func SafeJoin(baseDir string, cleanArchivePath string) (string, error) {
	base, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	target := filepath.Join(base, filepath.FromSlash(cleanArchivePath))
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s escapes %s", ErrUnsafeArchiveEntry, cleanArchivePath, baseDir)
	}
	return target, nil
}

func hasControlRune(value string) bool {
	for _, r := range value {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func hasWindowsDrivePrefix(value string) bool {
	return len(value) >= 2 && value[1] == ':' && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z'))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
