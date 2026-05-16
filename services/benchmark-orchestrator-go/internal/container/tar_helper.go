package container

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ConvertZipToTar reads a ZIP file from zipPath and returns a buffer containing the equivalent TAR archive.
// This is used to pass build context to the Docker daemon.
func ConvertZipToTar(zipPath string) (*bytes.Buffer, error) {
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file: %w", err)
	}
	defer reader.Close()

	// Detect if there's a single top-level directory.
	var commonPrefix string
	var hasCommonPrefix = true
	var firstLoop = true

	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		parts := strings.Split(cleanName, string(os.PathSeparator))
		if len(parts) == 0 {
			continue
		}
		prefix := parts[0]
		if firstLoop {
			commonPrefix = prefix
			firstLoop = false
		} else if prefix != commonPrefix {
			hasCommonPrefix = false
		}
	}

	if !hasCommonPrefix {
		commonPrefix = ""
	} else {
		// Ensure the common prefix is actually a directory (or the only thing)
		commonPrefix += "/"
	}

	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	defer tarWriter.Close()

	for _, file := range reader.File {
		// Path traversal check
		cleanName := filepath.Clean(file.Name)
		if strings.Contains(cleanName, "..") || filepath.IsAbs(cleanName) {
			continue
		}

		// Remove common prefix if it exists
		tarName := file.Name
		if commonPrefix != "" && strings.HasPrefix(tarName, commonPrefix) {
			tarName = strings.TrimPrefix(tarName, commonPrefix)
			if tarName == "" {
				continue
			}
		}

		// Normalize paths for tar
		tarName = filepath.ToSlash(filepath.Clean(tarName))

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
			_, err = io.Copy(tarWriter, src)
			src.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to copy content for %s: %w", tarName, err)
			}
		}
	}

	return &buf, nil
}
