package validator

import (
	"fmt"
	"sync"
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
	workspace := AnalyzeWorkspace(rootDir, p.contract)

	report := &domain.ValidationReport{
		SubmissionID: submissionID,
		Status:       domain.ValidationPassed,
		CheckResults: make(map[string]domain.CheckResult),
	}

	// 1. Core structural validators
	folderVal := NewFolderValidator(p.contract)
	extVal := NewExtensionValidator(p.contract)

	p.runValidators(workspace, report, []validatorJob{
		{name: folderVal.Name(), run: folderVal.ValidateContext},
		{name: extVal.Name(), run: extVal.ValidateContext},
	})

	if !report.CheckResults[folderVal.Name()].Passed {
		p.finalizeReport(report, workspace, start)
		return report
	}

	// 2. Concurrent Stage: Dockerfile, Config, Schema, Language
	var wg sync.WaitGroup
	var mu sync.Mutex

	dockerVal := NewDockerfileValidator(p.contract)
	configVal := NewConfigValidator(p.contract)
	schemaVal := NewSchemaValidator(p.contract)
	portVal := NewContractPortValidator(p.contract)
	langDetector := NewLanguageDetector()

	dockerDone := make(chan bool)

	// Dockerfile
	wg.Add(1)
	go func() {
		defer wg.Done()
		res := dockerVal.ValidateContext(workspace)
		mu.Lock()
		report.CheckResults[dockerVal.Name()] = res
		if res.Passed && workspace.Docker.HasEXPOSE {
			report.ExposedPort = workspace.Docker.Port
		}
		mu.Unlock()
		dockerDone <- res.Passed
	}()

	// Port (depends on Dockerfile)
	wg.Add(1)
	go func() {
		defer wg.Done()
		passed := <-dockerDone
		if !passed {
			return // skip port validation if dockerfile failed
		}
		res := portVal.ValidateContext(workspace)
		mu.Lock()
		report.CheckResults[portVal.Name()] = res
		mu.Unlock()
	}()

	// Config
	wg.Add(1)
	go func() {
		defer wg.Done()
		res := configVal.ValidateContext(workspace)
		mu.Lock()
		report.CheckResults[configVal.Name()] = res
		mu.Unlock()
	}()

	// Schema
	wg.Add(1)
	go func() {
		defer wg.Done()
		res := schemaVal.ValidateContext(workspace)
		mu.Lock()
		report.CheckResults[schemaVal.Name()] = res
		mu.Unlock()
	}()

	// Language
	wg.Add(1)
	go func() {
		defer wg.Done()
		langInfo := langDetector.DetectContext(workspace)
		mu.Lock()
		report.Language = langInfo.Language
		report.Runtime = langInfo.Runtime
		mu.Unlock()
	}()

	wg.Wait()

	p.finalizeReport(report, workspace, start)
	return report
}

type validatorJob struct {
	name string
	run  func(*WorkspaceContext) domain.CheckResult
}

func (p *Pipeline) runValidators(workspace *WorkspaceContext, report *domain.ValidationReport, jobs []validatorJob) {
	var wg sync.WaitGroup
	results := make(chan domain.CheckResult, len(jobs))

	for _, job := range jobs {
		wg.Add(1)
		go func(job validatorJob) {
			defer wg.Done()
			results <- job.run(workspace)
		}(job)
	}

	wg.Wait()
	close(results)

	for result := range results {
		report.CheckResults[result.Name] = result
	}
}

func (p *Pipeline) finalizeReport(report *domain.ValidationReport, workspace *WorkspaceContext, start time.Time) {
	report.DurationMs = time.Since(start).Milliseconds()

	for _, result := range report.CheckResults {
		report.TotalErrors += len(result.Errors)
		report.TotalWarnings += len(result.Warnings)

		if !result.Passed {
			report.Status = domain.ValidationFailed
		}
	}
	report.Compatibility = p.buildCompatibilityReport(report, workspace)
}

func (p *Pipeline) buildCompatibilityReport(report *domain.ValidationReport, workspace *WorkspaceContext) *domain.CompatibilityReport {
	compat := &domain.CompatibilityReport{
		Compatible:         report.Status == domain.ValidationPassed,
		RequiredPorts:      append([]int(nil), p.contract.RuntimeAPI.RequiredPorts...),
		ExposedPorts:       append([]int(nil), workspace.Docker.ExposedPorts...),
		BlockingIssueCount: report.TotalErrors,
		WarningCount:       report.TotalWarnings,
		RequiredAPI: []domain.EndpointCompatibility{
			endpointCompatibility("health", p.contract.RuntimeAPI.HealthEndpoint, workspace),
			endpointCompatibility("orders", p.contract.RuntimeAPI.OrderEndpoint, workspace),
			endpointCompatibility("cancel", p.contract.RuntimeAPI.CancelEndpoint, workspace),
		},
		RequiredWebSockets: []domain.WebSocketCompatibility{
			webSocketCompatibility("market_data", p.contract.RuntimeAPI.MarketDataStream, workspace),
		},
	}

	if compat.Compatible {
		compat.Summary = "Submission satisfies the benchmark contract and can be deployed."
	} else {
		compat.Summary = fmt.Sprintf("Submission has %d blocking contract issue(s).", report.TotalErrors)
	}

	if !workspace.Docker.HasHEALTHCHECK {
		compat.PerformanceHints = append(compat.PerformanceHints, "Add a Docker HEALTHCHECK so deployment readiness can be detected faster.")
	}
	if !workspace.Docker.HasUSER {
		compat.PerformanceHints = append(compat.PerformanceHints, "Run the container as a non-root USER to reduce sandbox risk.")
	}
	if !workspace.EndpointSignals.PingHandler {
		compat.PerformanceHints = append(compat.PerformanceHints, "Implement WebSocket ping/pong or heartbeat handling for stable long benchmark runs.")
	}

	return compat
}

func endpointCompatibility(name string, spec domain.EndpointSpec, workspace *WorkspaceContext) domain.EndpointCompatibility {
	return domain.EndpointCompatibility{
		Name:    name,
		Method:  spec.Method,
		Path:    spec.Path,
		Present: spec.Path == "" || (workspace.EndpointSignals.PathHits[spec.Path] && methodLikelyPresent(workspace, spec.Method)),
	}
}

func webSocketCompatibility(name string, spec domain.WebSocketSpec, workspace *WorkspaceContext) domain.WebSocketCompatibility {
	return domain.WebSocketCompatibility{
		Name:    name,
		Path:    spec.Path,
		Present: spec.Path == "" || workspace.EndpointSignals.WebSocketPathHits[spec.Path] || workspace.EndpointSignals.WebSocketUpgrade,
	}
}
