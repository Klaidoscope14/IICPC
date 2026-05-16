package domain

import "time"

// --- Deployment ---

type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"
	DeploymentStatusBuilding   DeploymentStatus = "building"
	DeploymentStatusDeployed   DeploymentStatus = "deployed"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusTerminated DeploymentStatus = "terminated"
)

type Deployment struct {
	ID             string           `json:"id"`
	SubmissionID   string           `json:"submission_id"`
	ContainerID    string           `json:"container_id,omitempty"`
	ContainerImage string           `json:"container_image"`
	ExposedPorts   []string         `json:"exposed_ports"`
	ServiceURL     string           `json:"service_url"`
	Status         DeploymentStatus `json:"status"`
	ResourceLimits ResourceLimits   `json:"resource_limits"`
	ErrorMessage   string           `json:"error_message,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type ResourceLimits struct {
	CPUMilli       int64 `json:"cpu_milli"`
	MemoryMB       int64 `json:"memory_mb"`
	TimeoutSeconds int64 `json:"timeout_seconds"`
}

// --- Benchmark ---

type BenchmarkStatus string

const (
	BenchmarkStatusPending   BenchmarkStatus = "pending"
	BenchmarkStatusRunning   BenchmarkStatus = "running"
	BenchmarkStatusCompleted BenchmarkStatus = "completed"
	BenchmarkStatusFailed    BenchmarkStatus = "failed"
	BenchmarkStatusStopped   BenchmarkStatus = "stopped"
)

type Benchmark struct {
	ID           string           `json:"id"`
	SubmissionID string           `json:"submission_id"`
	DeploymentID string           `json:"deployment_id"`
	Status       BenchmarkStatus  `json:"status"`
	Config       BenchmarkConfig  `json:"config"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
	ElapsedTime  int64            `json:"elapsed_time"`
	Metrics      TelemetryMetrics `json:"metrics"`
	ErrorMessage string           `json:"error_message,omitempty"`
}

type BenchmarkConfig struct {
	BotCount        int32    `json:"bot_count"`
	DurationSeconds int32    `json:"duration_seconds"`
	OrdersPerSecond int32    `json:"orders_per_second"`
	Protocols       []string `json:"protocols"`
}

// --- Submission logs ---

type SubmissionLog struct {
	ID           string            `json:"id"`
	SubmissionID string            `json:"submission_id"`
	LogType      string            `json:"log_type"`
	Message      string            `json:"message"`
	Level        string            `json:"level"`
	Metadata     map[string]string `json:"metadata"`
	CreatedAt    time.Time         `json:"created_at"`
}

// --- Telemetry ---

type TelemetryMetrics struct {
	CurrentTPS              float64 `json:"current_tps"`
	AvgLatencyMs            float64 `json:"avg_latency_ms"`
	TotalOrdersSent         int32   `json:"total_orders_sent"`
	TotalOrdersAcknowledged int32   `json:"total_orders_acknowledged"`
	TotalErrors             int32   `json:"total_errors"`
	P50LatencyMs            float64 `json:"p50_latency_ms"`
	P90LatencyMs            float64 `json:"p90_latency_ms"`
	P99LatencyMs            float64 `json:"p99_latency_ms"`
	ActiveConnections       int32   `json:"active_connections"`
	CPUUsagePercent         float64 `json:"cpu_usage_percent"`
	MemoryUsageMB           float64 `json:"memory_usage_mb"`
}

// --- Results ---

type BenchmarkResult struct {
	ID               string    `json:"id"`
	SubmissionID     string    `json:"submission_id"`
	BenchmarkID      string    `json:"benchmark_id"`
	TPS              float64   `json:"tps"`
	P50LatencyMs     float64   `json:"p50_latency_ms"`
	P90LatencyMs     float64   `json:"p90_latency_ms"`
	P99LatencyMs     float64   `json:"p99_latency_ms"`
	CorrectnessScore float64   `json:"correctness_score"`
	TotalOrders      int32     `json:"total_orders"`
	FailedOrders     int32     `json:"failed_orders"`
	CompositeScore   float64   `json:"composite_score"`
	CreatedAt        time.Time `json:"created_at"`
}

// --- Leaderboard ---

type LeaderboardEntry struct {
	Rank             int     `json:"rank"`
	TeamName         string  `json:"team_name"`
	TPS              float64 `json:"tps"`
	P50LatencyMs     float64 `json:"p50_latency_ms"`
	P90LatencyMs     float64 `json:"p90_latency_ms"`
	P99LatencyMs     float64 `json:"p99_latency_ms"`
	CorrectnessScore float64 `json:"correctness_score"`
	TotalOrders      int32   `json:"total_orders"`
	FailedOrders     int32   `json:"failed_orders"`
	CompositeScore   float64 `json:"composite_score"`
}

// --- Scoring ---

// DefaultScoringWeights returns the default weights for the composite score formula.
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		TPSWeight:         0.35,
		LatencyWeight:     0.30,
		CorrectnessWeight: 0.35,
	}
}

type ScoringWeights struct {
	TPSWeight         float64 `json:"tps_weight"`
	LatencyWeight     float64 `json:"latency_weight"`
	CorrectnessWeight float64 `json:"correctness_weight"`
}
