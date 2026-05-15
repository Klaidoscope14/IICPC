package validator

import "github.com/iicpc/validation-service-go/internal/domain"

// FolderValidator checks that required directories and files exist.
type FolderValidator struct {
	contract *domain.SubmissionContract
}

func NewFolderValidator(contract *domain.SubmissionContract) *FolderValidator {
	return &FolderValidator{contract: contract}
}

func (v *FolderValidator) Name() string { return "folder_structure" }

func (v *FolderValidator) Validate(rootDir string) domain.CheckResult {
	return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
}

func (v *FolderValidator) ValidateContext(ctx *WorkspaceContext) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	for _, dir := range ctx.MissingDirs {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_DIRECTORY",
			Message:  "Required directory not found: " + dir,
			Severity: domain.SeverityError,
			FilePath: dir,
		})
	}

	for _, file := range ctx.MissingFiles {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_FILE",
			Message:  "Required file not found: " + file,
			Severity: domain.SeverityError,
			FilePath: file,
		})
	}

	return result
}
