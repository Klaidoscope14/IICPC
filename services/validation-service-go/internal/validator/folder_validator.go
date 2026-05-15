package validator

import (
	"os"
	"path/filepath"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// FolderValidator checks that required directories and files exist.
type FolderValidator struct {
	contract *domain.SubmissionContract
}

func NewFolderValidator(contract *domain.SubmissionContract) *FolderValidator {
	return &FolderValidator{contract: contract}
}

func (v *FolderValidator) Name() string { return "folder_structure" }

func (v *FolderValidator) Validate(rootDir string) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	// Check required directories.
	for _, dir := range v.contract.RequiredDirs {
		fullPath := filepath.Join(rootDir, dir)
		info, err := os.Stat(fullPath)
		if err != nil || !info.IsDir() {
			result.Passed = false
			result.Errors = append(result.Errors, domain.ValidationError{
				Code:     "MISSING_DIRECTORY",
				Message:  "Required directory not found: " + dir,
				Severity: domain.SeverityError,
				FilePath: dir,
			})
		}
	}

	// Check required files.
	for _, file := range v.contract.RequiredFiles {
		fullPath := filepath.Join(rootDir, file)
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			result.Passed = false
			result.Errors = append(result.Errors, domain.ValidationError{
				Code:     "MISSING_FILE",
				Message:  "Required file not found: " + file,
				Severity: domain.SeverityError,
				FilePath: file,
			})
		}
	}

	return result
}
