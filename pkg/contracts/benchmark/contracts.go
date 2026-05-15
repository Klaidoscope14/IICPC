package benchmark

import "time"

type DeploymentStatus string

const (
	DeploymentPending    DeploymentStatus = "pending"
	DeploymentBuilding   DeploymentStatus = "building"
	DeploymentDeployed   DeploymentStatus = "deployed"
	DeploymentFailed     DeploymentStatus = "failed"
	DeploymentTerminated DeploymentStatus = "terminated"
)

type BenchmarkStatus string

const (
	BenchmarkPending   BenchmarkStatus = "pending"
	BenchmarkRunning   BenchmarkStatus = "running"
	BenchmarkCompleted BenchmarkStatus = "completed"
	BenchmarkFailed    BenchmarkStatus = "failed"
	BenchmarkStopped   BenchmarkStatus = "stopped"
)

type ResourceLimits struct {
	CPUMilli       int64 `json:"cpu_milli"`
	MemoryMB       int64 `json:"memory_mb"`
	TimeoutSeconds int64 `json:"timeout_seconds"`
}

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

type Config struct {
	BotCount        int32    `json:"bot_count"`
	DurationSeconds int32    `json:"duration_seconds"`
	OrdersPerSecond int32    `json:"orders_per_second"`
	Protocols       []string `json:"protocols"`
	Preset          string   `json:"preset,omitempty"`
}

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

type Run struct {
	ID           string           `json:"id"`
	SubmissionID string           `json:"submission_id"`
	DeploymentID string           `json:"deployment_id"`
	Status       BenchmarkStatus  `json:"status"`
	Config       Config           `json:"config"`
	StartedAt    time.Time        `json:"started_at"`
	CompletedAt  *time.Time       `json:"completed_at,omitempty"`
	ElapsedTime  int64            `json:"elapsed_time"`
	Metrics      TelemetryMetrics `json:"metrics"`
	ErrorMessage string           `json:"error_message,omitempty"`
}

type Result struct {
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

type LeaderboardEntry struct {
	Rank             int     `json:"rank"`
	SubmissionID     string  `json:"submission_id,omitempty"`
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
