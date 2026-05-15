package validation

import "time"

type Status string

const (
	StatusPending Status = "pending"
	StatusRunning Status = "running"
	StatusPassed  Status = "passed"
	StatusFailed  Status = "failed"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Finding struct {
	Code     string   `json:"code"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	FilePath string   `json:"file_path,omitempty"`
}

type CheckResult struct {
	Name     string    `json:"name"`
	Passed   bool      `json:"passed"`
	Errors   []Finding `json:"errors,omitempty"`
	Warnings []Finding `json:"warnings,omitempty"`
}

type LanguageInfo struct {
	Language string `json:"language"`
	Runtime  string `json:"runtime"`
	Standard string `json:"standard,omitempty"`
}

type Report struct {
	SubmissionID  string                 `json:"submission_id"`
	Status        Status                 `json:"status"`
	Language      string                 `json:"language"`
	Runtime       string                 `json:"runtime"`
	ExposedPort   int                    `json:"exposed_port,omitempty"`
	CheckResults  map[string]CheckResult `json:"check_results"`
	Compatibility *CompatibilityReport   `json:"compatibility,omitempty"`
	TotalErrors   int                    `json:"total_errors"`
	TotalWarnings int                    `json:"total_warnings"`
	DurationMs    int64                  `json:"duration_ms"`
}

type Result struct {
	ID           string     `json:"id"`
	SubmissionID string     `json:"submission_id"`
	Status       Status     `json:"status"`
	Language     string     `json:"language"`
	Runtime      string     `json:"runtime"`
	Errors       []Finding  `json:"errors,omitempty"`
	Warnings     []Finding  `json:"warnings,omitempty"`
	Report       *Report    `json:"report,omitempty"`
	ValidatedAt  *time.Time `json:"validated_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SubmissionContract captures the structural expectations for benchmarkable submissions.
type SubmissionContract struct {
	RequiredDirs           []string               `json:"required_dirs"`
	RequiredFiles          []string               `json:"required_files"`
	AllowedExtensions      map[string]bool        `json:"allowed_extensions"`
	ForbiddenExtensions    map[string]bool        `json:"forbidden_extensions"`
	ForbiddenPatterns      []string               `json:"forbidden_patterns"`
	DockerfileRequirements DockerfileRequirements `json:"dockerfile_requirements"`
	CMakeRequirements      CMakeRequirements      `json:"cmake_requirements"`
	RuntimeAPI             RuntimeAPIContract     `json:"runtime_api"`
	MaxExtractedBytes      int64                  `json:"max_extracted_bytes"`
	MaxFileCount           int                    `json:"max_file_count"`
}

type DockerfileRequirements struct {
	RequireFROM   bool `json:"require_from"`
	RequireEXPOSE bool `json:"require_expose"`
}

type CMakeRequirements struct {
	RequireProject       bool `json:"require_project"`
	RequireCXXStandard   bool `json:"require_cxx_standard"`
	RequireAddExecutable bool `json:"require_add_executable"`
}

// RuntimeAPIContract is the required API surface a submitted engine should expose.
type RuntimeAPIContract struct {
	HealthEndpoint   EndpointSpec  `json:"health_endpoint"`
	OrderEndpoint    EndpointSpec  `json:"order_endpoint"`
	CancelEndpoint   EndpointSpec  `json:"cancel_endpoint"`
	MarketDataStream WebSocketSpec `json:"market_data_stream"`
	RequiredPorts    []int         `json:"required_ports,omitempty"`
	EndpointPatterns []string      `json:"endpoint_patterns,omitempty"`
}

type EndpointSpec struct {
	Required    bool              `json:"required"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	ContentType string            `json:"content_type,omitempty"`
	Schema      map[string]string `json:"schema,omitempty"`
}

type WebSocketSpec struct {
	Required     bool     `json:"required"`
	Path         string   `json:"path"`
	MessageTypes []string `json:"message_types,omitempty"`
	RequiresPing bool     `json:"requires_ping"`
}

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
