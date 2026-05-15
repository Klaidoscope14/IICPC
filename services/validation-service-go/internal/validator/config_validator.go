package validator

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

var (
	cmakeProjectRegex    = regexp.MustCompile(`(?i)^\s*project\s*\(`)
	cmakeStandardRegex   = regexp.MustCompile(`(?i)^\s*set\s*\(\s*CMAKE_CXX_STANDARD\s+(\d+)`)
	cmakeExecutableRegex = regexp.MustCompile(`(?i)^\s*add_executable\s*\(`)
)

// ConfigValidator checks CMakeLists.txt for required project setup.
type ConfigValidator struct {
	contract *domain.SubmissionContract
}

func NewConfigValidator(contract *domain.SubmissionContract) *ConfigValidator {
	return &ConfigValidator{contract: contract}
}

func (v *ConfigValidator) Name() string { return "build_config" }

func (v *ConfigValidator) Validate(rootDir string) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	cmakePath := filepath.Join(rootDir, "CMakeLists.txt")
	file, err := os.Open(cmakePath)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "CMAKE_UNREADABLE",
			Message:  "Cannot read CMakeLists.txt: " + err.Error(),
			Severity: domain.SeverityError,
			FilePath: "CMakeLists.txt",
		})
		return result
	}
	defer file.Close()

	var hasProject, hasStandard, hasExecutable bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if cmakeProjectRegex.MatchString(line) {
			hasProject = true
		}
		if cmakeStandardRegex.MatchString(line) {
			hasStandard = true
		}
		if cmakeExecutableRegex.MatchString(line) {
			hasExecutable = true
		}
	}

	if v.contract.CMakeRequirements.RequireProject && !hasProject {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "CMAKE_NO_PROJECT",
			Message:  "CMakeLists.txt must contain a project() declaration",
			Severity: domain.SeverityError,
			FilePath: "CMakeLists.txt",
		})
	}

	if v.contract.CMakeRequirements.RequireCXXStandard && !hasStandard {
		// Just a warning, they might be relying on compiler defaults or target_compile_features
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "CMAKE_NO_STANDARD",
			Message:  "CMAKE_CXX_STANDARD is not explicitly set in CMakeLists.txt",
			Severity: domain.SeverityWarning,
			FilePath: "CMakeLists.txt",
		})
	}

	if v.contract.CMakeRequirements.RequireAddExecutable && !hasExecutable {
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
