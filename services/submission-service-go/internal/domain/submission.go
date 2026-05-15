package domain

import (
	"io"
	"time"

	contractsubmission "github.com/iicpc/pkg/contracts/submission"
)

// SubmissionStatus represents the lifecycle state of a submission.
type SubmissionStatus string

const (
	StatusPending          SubmissionStatus = SubmissionStatus(contractsubmission.StatusPending)
	StatusUploaded         SubmissionStatus = SubmissionStatus(contractsubmission.StatusUploaded)
	StatusValidationQueued SubmissionStatus = SubmissionStatus(contractsubmission.StatusValidationQueued)
	StatusValidating       SubmissionStatus = SubmissionStatus(contractsubmission.StatusValidating)
	StatusValidated        SubmissionStatus = SubmissionStatus(contractsubmission.StatusValidated)
	StatusValidationFailed SubmissionStatus = SubmissionStatus(contractsubmission.StatusValidationFailed)
	StatusProcessing       SubmissionStatus = "processing"
	StatusDeploying        SubmissionStatus = SubmissionStatus(contractsubmission.StatusDeploying)
	StatusDeployed         SubmissionStatus = SubmissionStatus(contractsubmission.StatusDeployed)
	StatusBenchmarking     SubmissionStatus = SubmissionStatus(contractsubmission.StatusBenchmarking)
	StatusCompleted        SubmissionStatus = SubmissionStatus(contractsubmission.StatusCompleted)
	StatusFailed           SubmissionStatus = SubmissionStatus(contractsubmission.StatusFailed)
	StatusDeleted          SubmissionStatus = SubmissionStatus(contractsubmission.StatusDeleted)
)

// IsValid returns true if the status is a recognized submission status.
func (s SubmissionStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusUploaded, StatusValidationQueued, StatusValidating, StatusValidated,
		StatusValidationFailed, StatusProcessing, StatusDeploying, StatusDeployed, StatusBenchmarking,
		StatusCompleted, StatusFailed, StatusDeleted:
		return true
	}
	return false
}

// Submission represents a contestant's code upload and its current state.
type Submission struct {
	ID               string            `json:"id" db:"id"`
	ContestantID     string            `json:"contestant_id" db:"contestant_id"`
	TeamName         string            `json:"team_name" db:"team_name"`
	Language         string            `json:"language" db:"language"`
	Status           SubmissionStatus  `json:"status" db:"status"`
	Version          int               `json:"version" db:"version"`
	CodeArchive      []byte            `json:"-" db:"code_archive"`
	Dockerfile       string            `json:"dockerfile" db:"dockerfile"`
	Checksum         string            `json:"checksum" db:"checksum"`
	OriginalFilename string            `json:"original_filename" db:"original_filename"`
	FileSize         int64             `json:"file_size" db:"file_size"`
	StoragePath      string            `json:"storage_path" db:"storage_path"`
	IdempotencyKey   string            `json:"idempotency_key,omitempty" db:"idempotency_key"`
	Metadata         map[string]string `json:"metadata" db:"metadata"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
}

// BenchmarkResult holds the performance metrics from a completed benchmark run.
type BenchmarkResult struct {
	ID               string    `json:"id" db:"id"`
	SubmissionID     string    `json:"submission_id" db:"submission_id"`
	TPS              float64   `json:"tps" db:"tps"`
	P50LatencyMs     float64   `json:"p50_latency_ms" db:"p50_latency_ms"`
	P90LatencyMs     float64   `json:"p90_latency_ms" db:"p90_latency_ms"`
	P99LatencyMs     float64   `json:"p99_latency_ms" db:"p99_latency_ms"`
	CorrectnessScore float64   `json:"correctness_score" db:"correctness_score"`
	TotalOrders      int32     `json:"total_orders" db:"total_orders"`
	FailedOrders     int32     `json:"failed_orders" db:"failed_orders"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// SubmissionLog represents a log entry (build, runtime, validation) attached to a submission.
type SubmissionLog struct {
	ID           string            `json:"id" db:"id"`
	SubmissionID string            `json:"submission_id" db:"submission_id"`
	LogType      string            `json:"log_type" db:"log_type"` // "upload", "build", "runtime", "validation"
	Message      string            `json:"message" db:"message"`
	Level        string            `json:"level" db:"level"` // "info", "warn", "error"
	Metadata     map[string]string `json:"metadata" db:"metadata"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
}

// CreateSubmissionParams is a pure value object for creating new submissions.
// This is the service-layer input — not for JSON serialization (handlers have their own DTOs).
type CreateSubmissionParams struct {
	ContestantID     string
	TeamName         string
	Language         string
	ArchiveReader    io.Reader
	CodeArchive      []byte
	Dockerfile       string
	OriginalFilename string
	FileSize         int64
	IdempotencyKey   string
	Metadata         map[string]string
}
