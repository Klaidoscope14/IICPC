package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
	"github.com/jmoiron/sqlx"
)

// OrchestratorRepository defines the persistence contract for deployments and benchmarks.
type OrchestratorRepository interface {
	// Deployments
	CreateDeployment(ctx context.Context, deployment *domain.Deployment) error
	GetDeploymentByID(ctx context.Context, id string) (*domain.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, serviceURL string, containerID string, errMsg string) error

	// Benchmarks
	CreateBenchmark(ctx context.Context, benchmark *domain.Benchmark) error
	GetBenchmarkByID(ctx context.Context, id string) (*domain.Benchmark, error)
	UpdateBenchmarkStatus(ctx context.Context, id string, status domain.BenchmarkStatus, elapsed int64, errMsg string) error
	CompleteBenchmark(ctx context.Context, id string, elapsed int64) error
	ListBenchmarksBySubmission(ctx context.Context, submissionID string) ([]*domain.Benchmark, error)

	// Telemetry
	InsertTelemetrySnapshot(ctx context.Context, benchmarkID string, metrics *domain.TelemetryMetrics) error

	// Results
	UpsertBenchmarkResult(ctx context.Context, result *domain.BenchmarkResult) error
	GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error)
}

// postgresRepository implements OrchestratorRepository using PostgreSQL.
type postgresRepository struct {
	db *sqlx.DB
}

// NewPostgresRepository creates a PostgreSQL-backed orchestrator repository.
func NewPostgresRepository(db *sqlx.DB) OrchestratorRepository {
	return &postgresRepository{db: db}
}

// --- Deployments ---

func (r *postgresRepository) CreateDeployment(ctx context.Context, d *domain.Deployment) error {
	limitsJSON, err := json.Marshal(d.ResourceLimits)
	if err != nil {
		return fmt.Errorf("failed to marshal resource limits: %w", err)
	}

	query := `
		INSERT INTO deployments (id, submission_id, container_id, container_image, service_url, exposed_ports, status, resource_limits, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.ExecContext(ctx, query,
		d.ID,
		d.SubmissionID,
		d.ContainerID,
		d.ContainerImage,
		d.ServiceURL,
		d.ExposedPorts,
		d.Status,
		limitsJSON,
		d.CreatedAt,
		d.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetDeploymentByID(ctx context.Context, id string) (*domain.Deployment, error) {
	query := `
		SELECT id, submission_id, container_id, container_image, service_url, exposed_ports, status, resource_limits, error_message, created_at, updated_at
		FROM deployments
		WHERE id = $1
	`

	var d domain.Deployment
	var limitsJSON []byte
	var containerID, serviceURL, errMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&d.ID,
		&d.SubmissionID,
		&containerID,
		&d.ContainerImage,
		&serviceURL,
		&d.ExposedPorts,
		&d.Status,
		&limitsJSON,
		&errMsg,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: deployment %s", domain.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}

	if containerID.Valid {
		d.ContainerID = containerID.String
	}
	if serviceURL.Valid {
		d.ServiceURL = serviceURL.String
	}
	if errMsg.Valid {
		d.ErrorMessage = errMsg.String
	}

	if len(limitsJSON) > 0 {
		if err := json.Unmarshal(limitsJSON, &d.ResourceLimits); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource limits: %w", err)
		}
	}

	return &d, nil
}

func (r *postgresRepository) UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, serviceURL string, containerID string, errMsg string) error {
	query := `
		UPDATE deployments
		SET status = $1, service_url = $2, container_id = $3, error_message = $4, updated_at = $5
		WHERE id = $6
	`

	_, err := r.db.ExecContext(ctx, query, status, serviceURL, containerID, errMsg, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update deployment status: %w", err)
	}
	return nil
}

// --- Benchmarks ---

func (r *postgresRepository) CreateBenchmark(ctx context.Context, b *domain.Benchmark) error {
	configJSON, err := json.Marshal(b.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark config: %w", err)
	}

	query := `
		INSERT INTO benchmarks (id, submission_id, deployment_id, status, config, started_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query,
		b.ID,
		b.SubmissionID,
		b.DeploymentID,
		b.Status,
		configJSON,
		b.StartedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create benchmark: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetBenchmarkByID(ctx context.Context, id string) (*domain.Benchmark, error) {
	query := `
		SELECT id, submission_id, deployment_id, status, config, started_at, completed_at, elapsed_seconds, error_message
		FROM benchmarks
		WHERE id = $1
	`

	var b domain.Benchmark
	var configJSON []byte
	var completedAt sql.NullTime
	var errMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID,
		&b.SubmissionID,
		&b.DeploymentID,
		&b.Status,
		&configJSON,
		&b.StartedAt,
		&completedAt,
		&b.ElapsedTime,
		&errMsg,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: benchmark %s", domain.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to get benchmark: %w", err)
	}

	if completedAt.Valid {
		b.CompletedAt = &completedAt.Time
	}
	if errMsg.Valid {
		b.ErrorMessage = errMsg.String
	}

	if len(configJSON) > 0 {
		if err := json.Unmarshal(configJSON, &b.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal benchmark config: %w", err)
		}
	}

	return &b, nil
}

func (r *postgresRepository) UpdateBenchmarkStatus(ctx context.Context, id string, status domain.BenchmarkStatus, elapsed int64, errMsg string) error {
	query := `
		UPDATE benchmarks
		SET status = $1, elapsed_seconds = $2, error_message = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, status, elapsed, errMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update benchmark status: %w", err)
	}
	return nil
}

func (r *postgresRepository) CompleteBenchmark(ctx context.Context, id string, elapsed int64) error {
	query := `
		UPDATE benchmarks
		SET status = $1, elapsed_seconds = $2, completed_at = $3
		WHERE id = $4
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, domain.BenchmarkStatusCompleted, elapsed, now, id)
	if err != nil {
		return fmt.Errorf("failed to complete benchmark: %w", err)
	}
	return nil
}

func (r *postgresRepository) ListBenchmarksBySubmission(ctx context.Context, submissionID string) ([]*domain.Benchmark, error) {
	query := `
		SELECT id, submission_id, deployment_id, status, config, started_at, completed_at, elapsed_seconds
		FROM benchmarks
		WHERE submission_id = $1
		ORDER BY started_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list benchmarks: %w", err)
	}
	defer rows.Close()

	var benchmarks []*domain.Benchmark
	for rows.Next() {
		var b domain.Benchmark
		var configJSON []byte
		var completedAt sql.NullTime

		err := rows.Scan(
			&b.ID, &b.SubmissionID, &b.DeploymentID, &b.Status,
			&configJSON, &b.StartedAt, &completedAt, &b.ElapsedTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan benchmark: %w", err)
		}

		if completedAt.Valid {
			b.CompletedAt = &completedAt.Time
		}
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &b.Config)
		}
		benchmarks = append(benchmarks, &b)
	}

	return benchmarks, nil
}

// --- Telemetry ---

func (r *postgresRepository) InsertTelemetrySnapshot(ctx context.Context, benchmarkID string, m *domain.TelemetryMetrics) error {
	query := `
		INSERT INTO telemetry_snapshots (benchmark_id, current_tps, avg_latency_ms, p50_latency_ms, p90_latency_ms, p99_latency_ms,
			total_orders_sent, total_orders_acknowledged, total_errors, active_connections, cpu_usage_percent, memory_usage_mb)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		benchmarkID,
		m.CurrentTPS, m.AvgLatencyMs, m.P50LatencyMs, m.P90LatencyMs, m.P99LatencyMs,
		m.TotalOrdersSent, m.TotalOrdersAcknowledged, m.TotalErrors,
		m.ActiveConnections, m.CPUUsagePercent, m.MemoryUsageMB,
	)
	if err != nil {
		return fmt.Errorf("failed to insert telemetry snapshot: %w", err)
	}
	return nil
}

// --- Results ---

func (r *postgresRepository) UpsertBenchmarkResult(ctx context.Context, result *domain.BenchmarkResult) error {
	query := `
		INSERT INTO benchmark_results (id, submission_id, benchmark_id, tps, p50_latency_ms, p90_latency_ms, p99_latency_ms,
			correctness_score, total_orders, failed_orders, composite_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (submission_id) DO UPDATE SET
			benchmark_id = $3,
			tps = $4,
			p50_latency_ms = $5,
			p90_latency_ms = $6,
			p99_latency_ms = $7,
			correctness_score = $8,
			total_orders = $9,
			failed_orders = $10,
			composite_score = $11,
			created_at = $12
	`

	_, err := r.db.ExecContext(ctx, query,
		result.ID, result.SubmissionID, result.BenchmarkID,
		result.TPS, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs,
		result.CorrectnessScore, result.TotalOrders, result.FailedOrders,
		result.CompositeScore, result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert benchmark result: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error) {
	query := `
		SELECT
			s.team_name,
			br.tps,
			br.p50_latency_ms,
			br.p90_latency_ms,
			br.p99_latency_ms,
			br.correctness_score,
			br.total_orders,
			br.failed_orders,
			br.composite_score
		FROM benchmark_results br
		JOIN submissions s ON s.id = br.submission_id
		ORDER BY br.composite_score DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []*domain.LeaderboardEntry
	rank := 1
	for rows.Next() {
		var e domain.LeaderboardEntry
		err := rows.Scan(
			&e.TeamName, &e.TPS,
			&e.P50LatencyMs, &e.P90LatencyMs, &e.P99LatencyMs,
			&e.CorrectnessScore, &e.TotalOrders, &e.FailedOrders,
			&e.CompositeScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		e.Rank = rank
		rank++
		entries = append(entries, &e)
	}

	return entries, nil
}
