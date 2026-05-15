package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// PortValidator ensures the port exposed in the Dockerfile is sensible.
type PortValidator struct {
	dockerfileInfo DockerfileInfo
	contract       *domain.SubmissionContract
}

func NewPortValidator(dockerfileInfo DockerfileInfo) *PortValidator {
	return &PortValidator{dockerfileInfo: dockerfileInfo}
}

func NewContractPortValidator(contract *domain.SubmissionContract) *PortValidator {
	return &PortValidator{contract: contract}
}

func (v *PortValidator) Name() string { return "port_binding" }

func (v *PortValidator) Validate(rootDir string) domain.CheckResult {
	if v.contract != nil {
		return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
	}
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	if !v.dockerfileInfo.HasEXPOSE {
		// If EXPOSE isn't required by contract and wasn't found, skip.
		// (The Dockerfile validator already fails if it's strictly required).
		return result
	}

	port := v.dockerfileInfo.Port

	if port < 1024 || port > 65535 {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "INVALID_PORT",
			Message:  fmt.Sprintf("Exposed port %d must be an unprivileged port between 1024 and 65535", port),
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
		return result
	}

	// Try to find if this port number is actually present anywhere in the source code as a sanity check.
	foundInCode := false
	portStr := strconv.Itoa(port)

	_ = filepath.WalkDir(filepath.Join(rootDir, "src"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		content, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(content), portStr) {
			foundInCode = true
			return filepath.SkipAll // Stop searching
		}
		return nil
	})

	if !foundInCode {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "PORT_NOT_IN_SOURCE",
			Message:  fmt.Sprintf("Port %d exposed in Dockerfile, but not found hardcoded in src/ code. Ensure it is read via environment variable or passed correctly.", port),
			Severity: domain.SeverityWarning,
			FilePath: "src/",
		})
	}

	return result
}

func (v *PortValidator) ValidateContext(ctx *WorkspaceContext) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	if !ctx.Docker.HasEXPOSE {
		return result
	}

	for _, port := range ctx.Docker.ExposedPorts {
		if port < 1024 || port > 65535 {
			result.Passed = false
			result.Errors = append(result.Errors, domain.ValidationError{
				Code:     "INVALID_PORT",
				Message:  fmt.Sprintf("Exposed port %d must be an unprivileged port between 1024 and 65535", port),
				Severity: domain.SeverityError,
				FilePath: "Dockerfile",
			})
		}
	}

	for _, requiredPort := range ctx.Contract.RuntimeAPI.RequiredPorts {
		if !containsInt(ctx.Docker.ExposedPorts, requiredPort) {
			result.Passed = false
			result.Errors = append(result.Errors, domain.ValidationError{
				Code:     "MISSING_REQUIRED_PORT",
				Message:  fmt.Sprintf("Dockerfile must EXPOSE required benchmark port %d", requiredPort),
				Severity: domain.SeverityError,
				FilePath: "Dockerfile",
			})
			continue
		}
		if !ctx.PortsFoundInSrc[requiredPort] {
			result.Warnings = append(result.Warnings, domain.ValidationError{
				Code:     "PORT_NOT_IN_SOURCE",
				Message:  fmt.Sprintf("Required port %d is exposed but not detected in src/include code. Prefer reading PORT from the environment or binding explicitly.", requiredPort),
				Severity: domain.SeverityWarning,
				FilePath: "src/",
			})
		}
	}

	return result
}

func containsInt(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
