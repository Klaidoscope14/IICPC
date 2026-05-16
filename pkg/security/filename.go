package security

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var safeFilenameRegex = regexp.MustCompile(`[^a-zA-Z0-9_.\-]`)

// SanitizeFilename strips path components, control characters, and unsafe
// bytes from a user-supplied filename.
func SanitizeFilename(filename string) string {
	filename = filepath.Base(filename)
	filename = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, filename)
	filename = safeFilenameRegex.ReplaceAllString(filename, "_")
	for strings.Contains(filename, "__") {
		filename = strings.ReplaceAll(filename, "__", "_")
	}
	filename = strings.Trim(filename, "_.")
	if filename == "" {
		return "submission"
	}
	return filename
}
