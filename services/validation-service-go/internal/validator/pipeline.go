package validator

import (
	"time"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// Pipeline orchestrates the execution of all validation rules.
type Pipeline struct {
	contract *domain.SubmissionContract
}

func NewPipeline(contract *domain.SubmissionContract) *Pipeline {
	return &Pipeline{contract: contract}
}

// Run executes the full validation suite against an extracted directory.
func (p *Pipeline) Run(submissionID string, rootDir string) *domain.ValidationReport {
	start := time.Now()
	
	report := &domain.ValidationReport{
		SubmissionID: submissionID,
		Status:       domain.ValidationPassed,
		CheckResults: make(map[string]domain.CheckResult),
	}

	// 1. Core structural validators
	folderVal := NewFolderValidator(p.contract)
	extVal := NewExtensionValidator(p.contract)

	report.CheckResults[folderVal.Name()] = folderVal.Validate(rootDir)
	report.CheckResults[extVal.Name()] = extVal.Validate(rootDir)

	// If core structure fails, short-circuit to avoid cascading failures
	if !report.CheckResults[folderVal.Name()].Passed {
		p.finalizeReport(report, start)
		return report
	}

	// 2. Dockerfile parsing
	dockerVal := NewDockerfileValidator(p.contract)
	dockerResult, dockerInfo := dockerVal.ValidateAndExtractInfo(rootDir)
	report.CheckResults[dockerVal.Name()] = dockerResult

	if dockerResult.Passed && dockerInfo.HasEXPOSE {
		report.ExposedPort = dockerInfo.Port
	}

	// If Dockerfile fails, short-circuit since port validation depends on it
	if !dockerResult.Passed {
		p.finalizeReport(report, start)
		return report
	}

	// 3. Deeper static analysis
	configVal := NewConfigValidator(p.contract)
	schemaVal := NewSchemaValidator(p.contract)
	portVal := NewPortValidator(dockerInfo)

	report.CheckResults[configVal.Name()] = configVal.Validate(rootDir)
	report.CheckResults[schemaVal.Name()] = schemaVal.Validate(rootDir)
	report.CheckResults[portVal.Name()] = portVal.Validate(rootDir)

	// 4. Language Detection
	langDetector := NewLanguageDetector()
	langInfo := langDetector.Detect(rootDir)
	report.Language = langInfo.Language
	report.Runtime = langInfo.Runtime

	p.finalizeReport(report, start)
	return report
}

func (p *Pipeline) finalizeReport(report *domain.ValidationReport, start time.Time) {
	report.DurationMs = time.Since(start).Milliseconds()

	for _, result := range report.CheckResults {
		report.TotalErrors += len(result.Errors)
		report.TotalWarnings += len(result.Warnings)

		if !result.Passed {
			report.Status = domain.ValidationFailed
		}
	}
}
