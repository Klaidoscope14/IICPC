package container

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/iicpc/pkg/security"
)

// ConvertZipToTar reads a ZIP file from zipPath and returns a buffer containing the equivalent TAR archive.
// This is used to pass build context to the Docker daemon.
func ConvertZipToTar(zipPath string) (*bytes.Buffer, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	limits := security.DefaultArchiveLimits()
	report, err := security.ValidateZipReader(&reader.Reader, limits)
	if err != nil {
		return nil, fmt.Errorf("unsafe build archive: %w", err)
	}

	// Detect if there's a single top-level directory.
	commonPrefix := commonTopLevelPrefix(report.Files)

	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer tarWriter.Close()

	for _, file := range reader.File {
		cleanName, err := security.SafeArchivePath(file.Name, limits)
		if err != nil {
			return nil, fmt.Errorf("unsafe build archive entry %q: %w", file.Name, err)
		}
		if err := security.ValidateArchiveEntry(cleanName, file.Mode()); err != nil {
			return nil, fmt.Errorf("dangerous build archive entry %q: %w", file.Name, err)
		}

		// Remove common prefix if it exists
		tarName := cleanName
		if commonPrefix != "" && strings.HasPrefix(tarName, commonPrefix) {
			tarName = strings.TrimPrefix(tarName, commonPrefix)
			if tarName == "" {
				continue
			}
		}

		// Normalize paths for tar
		tarName, err = security.SafeArchivePath(tarName, limits)
		if err != nil {
			return nil, fmt.Errorf("unsafe normalized tar path %q: %w", tarName, err)
		}

		header := &tar.Header{
			Name:     tarName,
			Size:     int64(file.UncompressedSize64),
			Mode:     int64(file.Mode().Perm()),
			ModTime:  file.Modified,
			Typeflag: tar.TypeReg,
		}

		if file.FileInfo().IsDir() {
			header.Typeflag = tar.TypeDir
			header.Name += "/"
			header.Size = 0
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("failed to write tar header for %s: %w", tarName, err)
		}

		if !file.FileInfo().IsDir() {
			src, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open zip entry %s: %w", tarName, err)
			}
			_, err = io.CopyN(tarWriter, src, int64(file.UncompressedSize64))
			if err == nil {
				var extra [1]byte
				if n, extraErr := src.Read(extra[:]); extraErr == nil && n > 0 {
					err = fmt.Errorf("zip entry %s exceeded declared size", tarName)
				}
			}
			src.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to copy content for %s: %w", tarName, err)
			}
		}
	}

	return &buf, nil
}

func commonTopLevelPrefix(files []string) string {
	if len(files) == 0 {
		return ""
	}

	prefix := ""
	for i, file := range files {
		parts := strings.Split(file, "/")
		if len(parts) < 2 {
			return ""
		}
		if i == 0 {
			prefix = parts[0]
			continue
		}
		if parts[0] != prefix {
			return ""
		}
	}

	return prefix + "/"
}
