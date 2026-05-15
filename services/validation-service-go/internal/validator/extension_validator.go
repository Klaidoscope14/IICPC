package validator

import "github.com/iicpc/validation-service-go/internal/domain"

// ExtensionValidator scans all files and rejects forbidden extensions / patterns.
type ExtensionValidator struct {
	contract *domain.SubmissionContract
}

func NewExtensionValidator(contract *domain.SubmissionContract) *ExtensionValidator {
	return &ExtensionValidator{contract: contract}
}

func (v *ExtensionValidator) Name() string { return "allowed_extensions" }

func (v *ExtensionValidator) Validate(rootDir string) domain.CheckResult {
	return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
}

func (v *ExtensionValidator) ValidateContext(ctx *WorkspaceContext) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}
	result.Errors = append(result.Errors, ctx.ForbiddenErrors...)
	result.Warnings = append(result.Warnings, ctx.ForbiddenWarnings...)
	result.Passed = len(result.Errors) == 0

	return result
}
