package domain

import "time"

// ValidationStatus represents the lifecycle state of a validation run.
type ValidationStatus string

const (
	ValidationPending ValidationStatus = "pending"
	ValidationRunning ValidationStatus = "running"
	ValidationPassed  ValidationStatus = "passed"
	ValidationFailed  ValidationStatus = "failed"
)

// Severity classifies validation findings.
type Severity string

const (
	SeverityError   Severity = "error"   // Blocks the submission from proceeding
	SeverityWarning Severity = "warning" // Non-blocking, informational
)

// ValidationError represents a single validation finding (error or warning).
type ValidationError struct {
	Code     string   `json:"code"`                // Machine-readable code, e.g. "MISSING_DOCKERFILE"
	Message  string   `json:"message"`             // Human-readable description
	Severity Severity `json:"severity"`            // "error" or "warning"
	FilePath string   `json:"file_path,omitempty"` // Path within the archive that caused the issue
}

// ValidationResult represents the persisted outcome of validating a submission.
type ValidationResult struct {
	ID           string            `json:"id" db:"id"`
	SubmissionID string            `json:"submission_id" db:"submission_id"`
	Status       ValidationStatus  `json:"status" db:"status"`
	Language     string            `json:"language" db:"language"`
	Runtime      string            `json:"runtime" db:"runtime"`
	Errors       []ValidationError `json:"errors"`
	Warnings     []ValidationError `json:"warnings"`
	Report       *ValidationReport `json:"report,omitempty"`
	ValidatedAt  *time.Time        `json:"validated_at,omitempty" db:"validated_at"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

// ValidationReport is the complete structured report with per-check sections.
type ValidationReport struct {
	SubmissionID  string                 `json:"submission_id"`
	Status        ValidationStatus       `json:"status"`
	Language      string                 `json:"language"`
	Runtime       string                 `json:"runtime"`
	ExposedPort   int                    `json:"exposed_port,omitempty"`
	CheckResults  map[string]CheckResult `json:"check_results"`
	Compatibility *CompatibilityReport   `json:"compatibility,omitempty"`
	TotalErrors   int                    `json:"total_errors"`
	TotalWarnings int                    `json:"total_warnings"`
	DurationMs    int64                  `json:"duration_ms"`
}

// CheckResult holds the outcome of a single validation check.
type CheckResult struct {
	Name     string            `json:"name"`
	Passed   bool              `json:"passed"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// LanguageInfo holds detected language and runtime information.
type LanguageInfo struct {
	Language string `json:"language"` // e.g. "cpp", "rust", "go"
	Runtime  string `json:"runtime"`  // e.g. "C++17", "C++20"
	Standard string `json:"standard"` // e.g. "17", "20"
}

// CompatibilityReport summarizes whether the submission can be deployed and benchmarked.
type CompatibilityReport struct {
	Compatible         bool                     `json:"compatible"`
	Summary            string                   `json:"summary"`
	RequiredPorts      []int                    `json:"required_ports,omitempty"`
	ExposedPorts       []int                    `json:"exposed_ports,omitempty"`
	RequiredAPI        []EndpointCompatibility  `json:"required_api,omitempty"`
	RequiredWebSockets []WebSocketCompatibility `json:"required_websockets,omitempty"`
	BlockingIssueCount int                      `json:"blocking_issue_count"`
	WarningCount       int                      `json:"warning_count"`
	PerformanceHints   []string                 `json:"performance_hints,omitempty"`
}

type EndpointCompatibility struct {
	Name    string `json:"name"`
	Method  string `json:"method"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}

type WebSocketCompatibility struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Present bool   `json:"present"`
}
