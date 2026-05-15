package validator

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// ExtensionValidator scans all files and rejects forbidden extensions / patterns.
type ExtensionValidator struct {
	contract *domain.SubmissionContract
}

func NewExtensionValidator(contract *domain.SubmissionContract) *ExtensionValidator {
	return &ExtensionValidator{contract: contract}
}

func (v *ExtensionValidator) Name() string { return "allowed_extensions" }

func (v *ExtensionValidator) Validate(rootDir string) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)

		// Check forbidden patterns (e.g., .git/, __pycache__/).
		for _, pattern := range v.contract.ForbiddenPatterns {
			if strings.Contains(relPath, pattern) {
				result.Warnings = append(result.Warnings, domain.ValidationError{
					Code:     "FORBIDDEN_PATTERN",
					Message:  "File matches forbidden pattern '" + pattern + "'",
					Severity: domain.SeverityWarning,
					FilePath: relPath,
				})
				return nil
			}
		}

		// Check forbidden extensions.
		ext := strings.ToLower(filepath.Ext(path))
		if v.contract.ForbiddenExtensions[ext] {
			result.Passed = false
			result.Errors = append(result.Errors, domain.ValidationError{
				Code:     "FORBIDDEN_EXTENSION",
				Message:  "File has forbidden extension: " + ext,
				Severity: domain.SeverityError,
				FilePath: relPath,
			})
		}

		return nil
	})

	return result
}
