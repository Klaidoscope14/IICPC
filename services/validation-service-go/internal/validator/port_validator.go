package validator

import (
	"fmt"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// PortValidator ensures the ports exposed in the Dockerfile are sensible and
// match the contract's required ports. All filesystem analysis is done once by
// AnalyzeWorkspace; this validator only inspects the pre-computed WorkspaceContext.
type PortValidator struct {
	contract *domain.SubmissionContract
}

func NewContractPortValidator(contract *domain.SubmissionContract) *PortValidator {
	return &PortValidator{contract: contract}
}

func (v *PortValidator) Name() string { return "port_binding" }

func (v *PortValidator) Validate(rootDir string) domain.CheckResult {
	return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
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
