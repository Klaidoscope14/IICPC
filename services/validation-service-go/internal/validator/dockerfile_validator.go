package validator

import (
	"github.com/iicpc/validation-service-go/internal/domain"
)

// DockerfileValidator checks the Dockerfile for required instructions.
// All parsing is done once by AnalyzeWorkspace; this validator only inspects
// the pre-computed WorkspaceContext.
type DockerfileValidator struct {
	contract *domain.SubmissionContract
}

func NewDockerfileValidator(contract *domain.SubmissionContract) *DockerfileValidator {
	return &DockerfileValidator{contract: contract}
}

func (v *DockerfileValidator) Name() string { return "dockerfile" }

func (v *DockerfileValidator) Validate(rootDir string) domain.CheckResult {
	return v.ValidateContext(AnalyzeWorkspace(rootDir, v.contract))
}

func (v *DockerfileValidator) ValidateContext(ctx *WorkspaceContext) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	if !ctx.Docker.Exists {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "DOCKERFILE_UNREADABLE",
			Message:  "Cannot read Dockerfile",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
		return result
	}

	if v.contract.DockerfileRequirements.RequireFROM && !ctx.Docker.HasFROM {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_FROM",
			Message:  "Dockerfile must contain a FROM instruction to specify the base image",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
	}

	if v.contract.DockerfileRequirements.RequireEXPOSE && !ctx.Docker.HasEXPOSE {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_EXPOSE",
			Message:  "Dockerfile must contain an EXPOSE instruction for the bot fleet to connect",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
	}

	if v.contract.DockerfileRequirements.WarnIfNoUSER && !ctx.Docker.HasUSER {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "DOCKER_NO_USER",
			Message:  "Dockerfile does not switch to a non-root USER; the platform will still apply sandbox limits, but non-root images are safer",
			Severity: domain.SeverityWarning,
			FilePath: "Dockerfile",
		})
	}

	if v.contract.DockerfileRequirements.WarnIfNoHEALTHCHECK && !ctx.Docker.HasHEALTHCHECK {
		result.Warnings = append(result.Warnings, domain.ValidationError{
			Code:     "DOCKER_NO_HEALTHCHECK",
			Message:  "Dockerfile does not define HEALTHCHECK; the platform will probe the required /health endpoint externally",
			Severity: domain.SeverityWarning,
			FilePath: "Dockerfile",
		})
	}

	return result
}
