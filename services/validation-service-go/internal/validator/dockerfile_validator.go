package validator

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/iicpc/validation-service-go/internal/domain"
)

var (
	dockerfileFromRegex   = regexp.MustCompile(`(?i)^\s*FROM\s+`)
	dockerfileExposeRegex = regexp.MustCompile(`(?i)^\s*EXPOSE\s+(\d+)`)
)

// DockerfileValidator parses the Dockerfile for required instructions.
type DockerfileValidator struct {
	contract *domain.SubmissionContract
}

func NewDockerfileValidator(contract *domain.SubmissionContract) *DockerfileValidator {
	return &DockerfileValidator{contract: contract}
}

func (v *DockerfileValidator) Name() string { return "dockerfile" }

// DockerfileInfo holds extracted information from the Dockerfile.
type DockerfileInfo struct {
	HasFROM    bool
	HasEXPOSE bool
	Port       int
	BaseImage  string
}

func (v *DockerfileValidator) Validate(rootDir string) domain.CheckResult {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	dockerfilePath := filepath.Join(rootDir, "Dockerfile")
	file, err := os.Open(dockerfilePath)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "DOCKERFILE_UNREADABLE",
			Message:  "Cannot read Dockerfile: " + err.Error(),
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
		return result
	}
	defer file.Close()

	info := v.parseDockerfile(file)

	if v.contract.DockerfileRequirements.RequireFROM && !info.HasFROM {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_FROM",
			Message:  "Dockerfile must contain a FROM instruction to specify the base image",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
	}

	if v.contract.DockerfileRequirements.RequireEXPOSE && !info.HasEXPOSE {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_EXPOSE",
			Message:  "Dockerfile must contain an EXPOSE instruction for the bot fleet to connect",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
	}

	return result
}

// ValidateAndExtractInfo validates and also returns extracted Dockerfile info.
func (v *DockerfileValidator) ValidateAndExtractInfo(rootDir string) (domain.CheckResult, DockerfileInfo) {
	result := domain.CheckResult{Name: v.Name(), Passed: true}

	dockerfilePath := filepath.Join(rootDir, "Dockerfile")
	file, err := os.Open(dockerfilePath)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "DOCKERFILE_UNREADABLE",
			Message:  "Cannot read Dockerfile: " + err.Error(),
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
		return result, DockerfileInfo{}
	}
	defer file.Close()

	info := v.parseDockerfile(file)

	if v.contract.DockerfileRequirements.RequireFROM && !info.HasFROM {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_FROM",
			Message:  "Dockerfile must contain a FROM instruction",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
	}

	if v.contract.DockerfileRequirements.RequireEXPOSE && !info.HasEXPOSE {
		result.Passed = false
		result.Errors = append(result.Errors, domain.ValidationError{
			Code:     "MISSING_EXPOSE",
			Message:  "Dockerfile must contain an EXPOSE instruction",
			Severity: domain.SeverityError,
			FilePath: "Dockerfile",
		})
	}

	return result, info
}

func (v *DockerfileValidator) parseDockerfile(file *os.File) DockerfileInfo {
	info := DockerfileInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines.
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if dockerfileFromRegex.MatchString(line) {
			info.HasFROM = true
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				info.BaseImage = parts[1]
			}
		}

		if matches := dockerfileExposeRegex.FindStringSubmatch(line); len(matches) >= 2 {
			info.HasEXPOSE = true
			if port, err := strconv.Atoi(matches[1]); err == nil {
				info.Port = port
			}
		}
	}

	return info
}
