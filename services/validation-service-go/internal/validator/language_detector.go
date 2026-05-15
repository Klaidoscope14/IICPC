package validator

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// LanguageDetector infers the primary language and runtime of the submission.
type LanguageDetector struct{}

func NewLanguageDetector() *LanguageDetector {
	return &LanguageDetector{}
}

func (v *LanguageDetector) Name() string { return "language_detection" }

// Detect analyzes the project to figure out language and standard.
func (v *LanguageDetector) Detect(rootDir string) domain.LanguageInfo {
	info := domain.LanguageInfo{
		Language: "unknown",
		Runtime:  "unknown",
		Standard: "unknown",
	}

	// For trade-engine, it's primarily C++. We look at CMakeLists.txt.
	cmakePath := filepath.Join(rootDir, "CMakeLists.txt")
	file, err := os.Open(cmakePath)
	if err == nil {
		defer file.Close()
		info.Language = "cpp" // Default if CMake exists

		standardRegex := regexp.MustCompile(`(?i)^\s*set\s*\(\s*CMAKE_CXX_STANDARD\s+(\d+)`)
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if matches := standardRegex.FindStringSubmatch(line); len(matches) >= 2 {
				info.Standard = matches[1]
				info.Runtime = "C++" + matches[1]
				break
			}
		}

		if info.Runtime == "unknown" {
			info.Runtime = "C++" // Default if standard not found
		}
	} else {
		// Fallback: scan extensions if CMake isn't available
		counts := make(map[string]int)
		_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != "" {
				counts[ext]++
			}
			return nil
		})

		if counts[".cpp"] > 0 || counts[".cc"] > 0 || counts[".cxx"] > 0 {
			info.Language = "cpp"
			info.Runtime = "C++"
		} else if counts[".c"] > 0 {
			info.Language = "c"
			info.Runtime = "C"
		} else if counts[".rs"] > 0 {
			info.Language = "rust"
			info.Runtime = "Rust"
		} else if counts[".go"] > 0 {
			info.Language = "go"
			info.Runtime = "Go"
		}
	}

	return info
}
