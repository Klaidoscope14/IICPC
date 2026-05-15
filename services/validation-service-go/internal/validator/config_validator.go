package validator

import "github.com/iicpc/validation-service-go/internal/domain"

// ConfigValidator checks CMakeLists.txt for required project setup.
type ConfigValidator struct {
	contract *domain.SubmissionContract
}

func NewConfigValidator(contract *domain.SubmissionContract) *ConfigValidator {
	return &ConfigValidator{contract: contract}
}

func (v *ConfigValidator) Name() string { return "build_config" }

func (v *ConfigValidator) Validate(rootDir string) domain.CheckResult {
	return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
}

func (v *ConfigValidator) ValidateContext(ctx *WorkspaceContext) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	if !ctx.CMake.Exists {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "CMAKE_UNREADABLE",
			Message:  "Cannot read CMakeLists.txt",
			Severity: domain.SeverityError,
			FilePath: "CMakeLists.txt",
		})
		return result
	}

	if v.contract.CMakeRequirements.RequireProject && !ctx.CMake.HasProject {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "CMAKE_NO_PROJECT",
			Message:  "CMakeLists.txt must contain a project() declaration",
			Severity: domain.SeverityError,
			FilePath: "CMakeLists.txt",
		})
	}

	if v.contract.CMakeRequirements.RequireCXXStandard && !ctx.CMake.HasStandard {
		// Just a warning, they might be relying on compiler defaults or target_compile_features
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "CMAKE_NO_STANDARD",
			Message:  "CMAKE_CXX_STANDARD is not explicitly set in CMakeLists.txt",
			Severity: domain.SeverityWarning,
			FilePath: "CMakeLists.txt",
		})
	}

	if v.contract.CMakeRequirements.RequireAddExecutable && !ctx.CMake.HasExecutable {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "CMAKE_NO_EXECUTABLE",
			Message:  "CMakeLists.txt must define an executable using add_executable()",
			Severity: domain.SeverityError,
			FilePath: "CMakeLists.txt",
		})
	}

	return result
}
