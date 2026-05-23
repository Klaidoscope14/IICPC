package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// OrchestratorRepository defines the persistence contract for deployments and benchmarks.
type OrchestratorRepository interface {
	// Deployments
	CreateDeployment(ctx context.Context, deployment *domain.Deployment) error
	GetDeploymentByID(ctx context.Context, id string) (*domain.Deployment, error)
	GetLatestDeploymentBySubmission(ctx context.Context, submissionID string) (*domain.Deployment, error)
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, serviceURL string, containerID string, errMsg string) error
	GetSubmissionStoragePath(ctx context.Context, submissionID string) (string, error)
	GetSubmissionBuildInputs(ctx context.Context, submissionID string) (storagePath string, checksum string, err error)
	CreateSubmissionLog(ctx context.Context, log *domain.SubmissionLog) error

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
	GetBenchmarkResult(ctx context.Context, benchmarkID string) (*domain.BenchmarkResult, error)
	UpdateCorrectnessScore(ctx context.Context, benchmarkID string, correctnessScore float64, newCompositeScore float64) error
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
		pq.Array(&d.ExposedPorts),
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

func (r *postgresRepository) GetLatestDeploymentBySubmission(ctx context.Context, submissionID string) (*domain.Deployment, error) {
	query := `
		SELECT id, submission_id, container_id, container_image, service_url, exposed_ports, status, resource_limits, error_message, created_at, updated_at
		FROM deployments
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var d domain.Deployment
	var limitsJSON []byte
	var containerID, serviceURL, errMsg sql.NullString

	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(
		&d.ID,
		&d.SubmissionID,
		&containerID,
		&d.ContainerImage,
		&serviceURL,
		pq.Array(&d.ExposedPorts),
		&d.Status,
		&limitsJSON,
		&errMsg,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: no deployments for submission %s", domain.ErrNotFound, submissionID)
		}
		return nil, fmt.Errorf("failed to get latest deployment: %w", err)
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

func (r *postgresRepository) GetSubmissionStoragePath(ctx context.Context, submissionID string) (string, error) {
	query := `SELECT storage_path FROM submissions WHERE id = $1`
	var path string
	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(&path)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("%w: submission %s", domain.ErrNotFound, submissionID)
		}
		return "", fmt.Errorf("failed to get storage path: %w", err)
	}
	return path, nil
}

func (r *postgresRepository) GetSubmissionBuildInputs(ctx context.Context, submissionID string) (string, string, error) {
	query := `SELECT storage_path, checksum FROM submissions WHERE id = $1`
	var storagePath string
	var checksum string
	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(&storagePath, &checksum)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", fmt.Errorf("%w: submission %s", domain.ErrNotFound, submissionID)
		}
		return "", "", fmt.Errorf("failed to get submission build inputs: %w", err)
	}
	return storagePath, checksum, nil
}

func (r *postgresRepository) CreateSubmissionLog(ctx context.Context, log *domain.SubmissionLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal submission log metadata: %w", err)
	}

	query := `
		INSERT INTO submission_logs (id, submission_id, log_type, message, level, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = r.db.ExecContext(ctx, query,
		log.ID,
		log.SubmissionID,
		log.LogType,
		log.Message,
		log.Level,
		metadataJSON,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create submission log: %w", err)
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
		SELECT b.id, b.submission_id, b.deployment_id, b.status, b.config, b.started_at, b.completed_at, b.elapsed_seconds, b.error_message,
			COALESCE(r.tps, m.current_tps), m.avg_latency_ms, m.p50_latency_ms, m.p90_latency_ms, m.p99_latency_ms,
			m.total_orders_sent, m.total_orders_acknowledged, m.total_errors, m.active_connections,
			m.cpu_usage_percent, m.memory_usage_mb
		FROM benchmarks b
		LEFT JOIN LATERAL (
			SELECT current_tps, avg_latency_ms, p50_latency_ms, p90_latency_ms, p99_latency_ms,
				total_orders_sent, total_orders_acknowledged, total_errors, active_connections,
				cpu_usage_percent, memory_usage_mb
			FROM telemetry_snapshots
			WHERE benchmark_id = b.id
			ORDER BY timestamp DESC
			LIMIT 1
		) m ON true
		LEFT JOIN benchmark_results r ON r.benchmark_id = b.id
		WHERE b.id = $1
	`

	var b domain.Benchmark
	var configJSON []byte
	var completedAt sql.NullTime
	var errMsg sql.NullString
	var currentTPS, avgLatencyMs, p50LatencyMs, p90LatencyMs, p99LatencyMs sql.NullFloat64
	var totalSent, totalAcked, totalErrors, activeConnections sql.NullInt64
	var cpuUsage, memoryUsage sql.NullFloat64

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
		&currentTPS,
		&avgLatencyMs,
		&p50LatencyMs,
		&p90LatencyMs,
		&p99LatencyMs,
		&totalSent,
		&totalAcked,
		&totalErrors,
		&activeConnections,
		&cpuUsage,
		&memoryUsage,
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
	if currentTPS.Valid {
		b.Metrics.CurrentTPS = currentTPS.Float64
	}
	if avgLatencyMs.Valid {
		b.Metrics.AvgLatencyMs = avgLatencyMs.Float64
	}
	if p50LatencyMs.Valid {
		b.Metrics.P50LatencyMs = p50LatencyMs.Float64
	}
	if p90LatencyMs.Valid {
		b.Metrics.P90LatencyMs = p90LatencyMs.Float64
	}
	if p99LatencyMs.Valid {
		b.Metrics.P99LatencyMs = p99LatencyMs.Float64
	}
	if totalSent.Valid {
		b.Metrics.TotalOrdersSent = int32(totalSent.Int64)
	}
	if totalAcked.Valid {
		b.Metrics.TotalOrdersAcknowledged = int32(totalAcked.Int64)
	}
	if totalErrors.Valid {
		b.Metrics.TotalErrors = int32(totalErrors.Int64)
	}
	if activeConnections.Valid {
		b.Metrics.ActiveConnections = int32(activeConnections.Int64)
	}
	if cpuUsage.Valid {
		b.Metrics.CPUUsagePercent = cpuUsage.Float64
	}
	if memoryUsage.Valid {
		b.Metrics.MemoryUsageMB = memoryUsage.Float64
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate benchmarks: %w", err)
	}

	return benchmarks, nil
}

// --- Telemetry ---

func (r *postgresRepository) InsertTelemetrySnapshot(ctx context.Context, benchmarkID string, m *domain.TelemetryMetrics) error {
	query := `
		INSERT INTO telemetry_snapshots (benchmark_id, timestamp, current_tps, avg_latency_ms, p50_latency_ms, p90_latency_ms, p99_latency_ms,
			total_orders_sent, total_orders_acknowledged, total_errors, active_connections, cpu_usage_percent, memory_usage_mb)
		VALUES ($1, NOW(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin benchmark result transaction: %w", err)
	}
	defer tx.Rollback()

	historyQuery := `
		INSERT INTO benchmark_history (id, submission_id, benchmark_id, tps, p50_latency_ms, p90_latency_ms, p99_latency_ms,
			correctness_score, total_orders, failed_orders, composite_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (benchmark_id) DO UPDATE SET
			tps = EXCLUDED.tps,
			p50_latency_ms = EXCLUDED.p50_latency_ms,
			p90_latency_ms = EXCLUDED.p90_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			correctness_score = EXCLUDED.correctness_score,
			total_orders = EXCLUDED.total_orders,
			failed_orders = EXCLUDED.failed_orders,
			composite_score = EXCLUDED.composite_score,
			created_at = EXCLUDED.created_at
	`

	_, err = tx.ExecContext(ctx, historyQuery,
		result.ID, result.SubmissionID, result.BenchmarkID,
		result.TPS, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs,
		result.CorrectnessScore, result.TotalOrders, result.FailedOrders,
		result.CompositeScore, result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert benchmark history: %w", err)
	}

	currentQuery := `
		INSERT INTO benchmark_results (id, submission_id, benchmark_id, tps, p50_latency_ms, p90_latency_ms, p99_latency_ms,
			correctness_score, total_orders, failed_orders, composite_score, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (submission_id) DO UPDATE SET
			benchmark_id = EXCLUDED.benchmark_id,
			tps = EXCLUDED.tps,
			p50_latency_ms = EXCLUDED.p50_latency_ms,
			p90_latency_ms = EXCLUDED.p90_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			correctness_score = EXCLUDED.correctness_score,
			total_orders = EXCLUDED.total_orders,
			failed_orders = EXCLUDED.failed_orders,
			composite_score = EXCLUDED.composite_score,
			created_at = EXCLUDED.created_at
	`

	_, err = tx.ExecContext(ctx, currentQuery,
		result.ID, result.SubmissionID, result.BenchmarkID,
		result.TPS, result.P50LatencyMs, result.P90LatencyMs, result.P99LatencyMs,
		result.CorrectnessScore, result.TotalOrders, result.FailedOrders,
		result.CompositeScore, result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert benchmark result: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit benchmark result transaction: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetBenchmarkResult(ctx context.Context, benchmarkID string) (*domain.BenchmarkResult, error) {
	query := `
		SELECT id, submission_id, benchmark_id, tps, p50_latency_ms, p90_latency_ms, p99_latency_ms,
			correctness_score, total_orders, failed_orders, composite_score, created_at
		FROM benchmark_results
		WHERE benchmark_id = $1
	`
	var res domain.BenchmarkResult
	err := r.db.QueryRowContext(ctx, query, benchmarkID).Scan(
		&res.ID, &res.SubmissionID, &res.BenchmarkID,
		&res.TPS, &res.P50LatencyMs, &res.P90LatencyMs, &res.P99LatencyMs,
		&res.CorrectnessScore, &res.TotalOrders, &res.FailedOrders,
		&res.CompositeScore, &res.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: benchmark result %s", domain.ErrNotFound, benchmarkID)
		}
		return nil, fmt.Errorf("failed to get benchmark result: %w", err)
	}
	return &res, nil
}

func (r *postgresRepository) UpdateCorrectnessScore(ctx context.Context, benchmarkID string, correctnessScore float64, newCompositeScore float64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin update score transaction: %w", err)
	}
	defer tx.Rollback()

	queryHistory := `
		UPDATE benchmark_history
		SET correctness_score = $1, composite_score = $2
		WHERE benchmark_id = $3
	`
	if _, err := tx.ExecContext(ctx, queryHistory, correctnessScore, newCompositeScore, benchmarkID); err != nil {
		return fmt.Errorf("failed to update benchmark_history score: %w", err)
	}

	queryCurrent := `
		UPDATE benchmark_results
		SET correctness_score = $1, composite_score = $2
		WHERE benchmark_id = $3
	`
	if _, err := tx.ExecContext(ctx, queryCurrent, correctnessScore, newCompositeScore, benchmarkID); err != nil {
		return fmt.Errorf("failed to update benchmark_results score: %w", err)
	}

	return tx.Commit()
}

func (r *postgresRepository) GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error) {
	query := `
		SELECT
			br.submission_id,
			br.benchmark_id,
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
		ORDER BY br.composite_score DESC, br.tps DESC, br.p99_latency_ms ASC
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
			&e.SubmissionID, &e.BenchmarkID,
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate leaderboard entries: %w", err)
	}

	return entries, nil
}
