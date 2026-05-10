package domain

import "time"

// SubmissionStatus represents the lifecycle state of a submission.
type SubmissionStatus string

const (
	StatusPending      SubmissionStatus = "pending"
	StatusProcessing   SubmissionStatus = "processing"
	StatusDeployed     SubmissionStatus = "deployed"
	StatusFailed       SubmissionStatus = "failed"
	StatusBenchmarking SubmissionStatus = "benchmarking"
)

// IsValid returns true if the status is a recognized submission status.
func (s SubmissionStatus) IsValid() bool {
	switch s {
	case StatusPending, StatusProcessing, StatusDeployed, StatusFailed, StatusBenchmarking:
		return true
	}
	return false
}

// Submission represents a contestant's code upload and its current state.
type Submission struct {
	ID           string            `json:"id" db:"id"`
	ContestantID string            `json:"contestant_id" db:"contestant_id"`
	TeamName     string            `json:"team_name" db:"team_name"`
	Language     string            `json:"language" db:"language"`
	Status       SubmissionStatus  `json:"status" db:"status"`
	CodeArchive  []byte            `json:"-" db:"code_archive"`
	Dockerfile   string            `json:"dockerfile" db:"dockerfile"`
	Metadata     map[string]string `json:"metadata" db:"metadata"`
	CreatedAt    time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
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

// CreateSubmissionParams is a pure value object for creating new submissions.
// This is the service-layer input — not for JSON serialization (handlers have their own DTOs).
type CreateSubmissionParams struct {
	ContestantID string
	TeamName     string
	Language     string
	CodeArchive  []byte
	Dockerfile   string
	Metadata     map[string]string
}
