package events

import (
	"encoding/json"
	"time"

	"github.com/iicpc/pkg/contracts/benchmark"
	"github.com/iicpc/pkg/contracts/validation"
)

const SchemaVersion = "v1"

const (
	// Submission uploads use the existing submission.created topic name so current consumers stay compatible.
	TopicSubmissionUploaded   = "submission.created"
	TopicSubmissionDeleted    = "submission.deleted"
	TopicValidationCompleted  = "validation.completed"
	TopicEngineReady          = "deployment.ready"
	TopicBenchmarkStarted     = "benchmark.started"
	TopicBenchmarkFinished    = "benchmark.completed"
	TopicTelemetrySnapshot    = "telemetry.snapshot"
	TopicLeaderboardUpdated   = "leaderboard.updated"
	TopicTraceAvailable       = "benchmark.trace_available"
	TopicCorrectnessEvaluated = "correctness.evaluated"
)

type Envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       string          `json:"version"`
	Source        string          `json:"source"`
	Key           string          `json:"key"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

type SubmissionUploadedEvent struct {
	SubmissionID   string    `json:"submission_id"`
	ContestantID   string    `json:"contestant_id,omitempty"`
	TeamName       string    `json:"team_name"`
	Language       string    `json:"language"`
	Version        int       `json:"version,omitempty"`
	StoragePath    string    `json:"storage_path,omitempty"`
	Checksum       string    `json:"checksum,omitempty"`
	ContainerImage string    `json:"container_image,omitempty"`
	Preset         string    `json:"preset,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type SubmissionDeletedEvent struct {
	SubmissionID string    `json:"submission_id"`
	DeletedAt    time.Time `json:"deleted_at"`
}

type ValidationCompletedEvent struct {
	SubmissionID string               `json:"submission_id"`
	Status       string               `json:"status"`
	Language     string               `json:"language"`
	Runtime      string               `json:"runtime"`
	Preset       string               `json:"preset,omitempty"`
	ErrorCount   int                  `json:"error_count"`
	WarningCount int                  `json:"warning_count"`
	Errors       []validation.Finding `json:"errors,omitempty"`
	Warnings     []validation.Finding `json:"warnings,omitempty"`
	ValidatedAt  time.Time            `json:"validated_at"`
}

type EngineReadyEvent struct {
	DeploymentID string    `json:"deployment_id"`
	SubmissionID string    `json:"submission_id"`
	ServiceURL   string    `json:"service_url"`
	ContainerID  string    `json:"container_id"`
	Preset       string    `json:"preset,omitempty"`
	ReadyAt      time.Time `json:"ready_at"`
}

type BenchmarkStartedEvent struct {
	BenchmarkID  string           `json:"benchmark_id"`
	SubmissionID string           `json:"submission_id"`
	DeploymentID string           `json:"deployment_id"`
	ServiceURL   string           `json:"service_url"`
	Config       benchmark.Config `json:"config,omitempty"`
	StartedAt    time.Time        `json:"started_at"`
}

type BenchmarkFinishedEvent struct {
	BenchmarkID      string    `json:"benchmark_id"`
	SubmissionID     string    `json:"submission_id"`
	CompositeScore   float64   `json:"composite_score"`
	TPS              float64   `json:"tps"`
	P50LatencyMs     float64   `json:"p50_latency_ms,omitempty"`
	P90LatencyMs     float64   `json:"p90_latency_ms,omitempty"`
	P99LatencyMs     float64   `json:"p99_latency_ms"`
	CorrectnessScore float64   `json:"correctness_score"`
	TotalOrders      int32     `json:"total_orders,omitempty"`
	FailedOrders     int32     `json:"failed_orders,omitempty"`
	ElapsedSeconds   int64     `json:"elapsed_seconds"`
	FinishedAt       time.Time `json:"finished_at,omitempty"`
}

type TelemetrySnapshotEvent struct {
	BenchmarkID string                     `json:"benchmark_id"`
	Timestamp   time.Time                  `json:"timestamp"`
	Metrics     benchmark.TelemetryMetrics `json:"metrics"`
}

type LeaderboardUpdatedEvent struct {
	BenchmarkID string                       `json:"benchmark_id,omitempty"`
	UpdatedAt   time.Time                    `json:"updated_at"`
	Entries     []benchmark.LeaderboardEntry `json:"entries"`
}

type TraceAvailableEvent struct {
	BenchmarkID string    `json:"benchmark_id"`
	FilePath    string    `json:"file_path"`
	CreatedAt   time.Time `json:"created_at"`
}

type CorrectnessEvaluatedEvent struct {
	BenchmarkID      string    `json:"benchmark_id"`
	CorrectnessScore float64   `json:"correctness_score"`
	TotalViolations  int32     `json:"total_violations"`
	EvaluatedAt      time.Time `json:"evaluated_at"`
}
