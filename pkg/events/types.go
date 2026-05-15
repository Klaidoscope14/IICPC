package events

import "time"

// Topic names for the event bus.
const (
	TopicSubmissionCreated   = "submission.created"
	TopicValidationCompleted = "validation.completed"
	TopicDeploymentReady     = "deployment.ready"
	TopicBenchmarkStarted    = "benchmark.started"
	TopicBenchmarkCompleted  = "benchmark.completed"
	TopicTelemetrySnapshot   = "telemetry.snapshot"
)

// SubmissionCreatedEvent is published when a new submission is accepted.
type SubmissionCreatedEvent struct {
	SubmissionID   string    `json:"submission_id"`
	TeamName       string    `json:"team_name"`
	Language       string    `json:"language"`
	ContainerImage string    `json:"container_image,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// ValidationCompletedEvent is published when a submission has been validated.
type ValidationCompletedEvent struct {
	SubmissionID string    `json:"submission_id"`
	Status       string    `json:"status"` // "passed" or "failed"
	Language     string    `json:"language"`
	Runtime      string    `json:"runtime"`
	ErrorCount   int       `json:"error_count"`
	WarningCount int       `json:"warning_count"`
	ValidatedAt  time.Time `json:"validated_at"`
}

// DeploymentReadyEvent is published when a submission's container is deployed.
type DeploymentReadyEvent struct {
	DeploymentID string    `json:"deployment_id"`
	SubmissionID string    `json:"submission_id"`
	ServiceURL   string    `json:"service_url"`
	ContainerID  string    `json:"container_id"`
	ReadyAt      time.Time `json:"ready_at"`
}

// BenchmarkStartedEvent is published when a benchmark run begins.
type BenchmarkStartedEvent struct {
	BenchmarkID  string    `json:"benchmark_id"`
	SubmissionID string    `json:"submission_id"`
	DeploymentID string    `json:"deployment_id"`
	StartedAt    time.Time `json:"started_at"`
}

// BenchmarkCompletedEvent is published when a benchmark run finishes.
type BenchmarkCompletedEvent struct {
	BenchmarkID    string  `json:"benchmark_id"`
	SubmissionID   string  `json:"submission_id"`
	CompositeScore float64 `json:"composite_score"`
	TPS            float64 `json:"tps"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	CorrectnessScore float64 `json:"correctness_score"`
	ElapsedSeconds int64   `json:"elapsed_seconds"`
}
