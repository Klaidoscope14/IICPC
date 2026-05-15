package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/iicpc/submission-service-go/internal/domain"
	"github.com/jmoiron/sqlx"
)

// postgresSubmissionRepository implements service.SubmissionRepository using PostgreSQL.
type postgresSubmissionRepository struct {
	db *sqlx.DB
}

// NewPostgresSubmissionRepository creates a PostgreSQL-backed submission repository.
func NewPostgresSubmissionRepository(db *sqlx.DB) *postgresSubmissionRepository {
	return &postgresSubmissionRepository{db: db}
}

func (r *postgresSubmissionRepository) Create(ctx context.Context, submission *domain.Submission) error {
	if submission.Version == 0 {
		submission.Version = 1
	}
	if err := r.insertSubmission(ctx, r.db, submission); err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}
	return nil
}

// CreateWithNextVersion serializes version assignment per contestant and inserts
// the submission in one transaction. This avoids races when a team retries or
// submits multiple versions concurrently.
func (r *postgresSubmissionRepository) CreateWithNextVersion(ctx context.Context, submission *domain.Submission) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin submission transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, submission.ContestantID); err != nil {
		return fmt.Errorf("failed to lock contestant version stream: %w", err)
	}

	if err := tx.QueryRowxContext(ctx, `
		SELECT COALESCE(MAX(version), 0) + 1
		FROM submissions
		WHERE contestant_id = $1 AND status != 'deleted'
	`, submission.ContestantID).Scan(&submission.Version); err != nil {
		return fmt.Errorf("failed to assign next version: %w", err)
	}

	if err := r.insertSubmission(ctx, tx, submission); err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit submission transaction: %w", err)
	}
	return nil
}

type submissionExecutor interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

func (r *postgresSubmissionRepository) insertSubmission(ctx context.Context, exec submissionExecutor, submission *domain.Submission) error {
	metadataJSON, err := marshalStringMap(submission.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO submissions (id, contestant_id, team_name, language, status, version, code_archive, dockerfile, checksum, original_filename, file_size, storage_path, idempotency_key, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err = r.db.ExecContext(ctx, query,
		submission.ID,
		submission.ContestantID,
		submission.TeamName,
		submission.Language,
		submission.Status,
		submission.Version,
		nil,
		submission.Dockerfile,
		submission.Checksum,
		submission.OriginalFilename,
		submission.FileSize,
		submission.StoragePath,
		nullString(submission.IdempotencyKey),
		metadataJSON,
		submission.CreatedAt,
		submission.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *postgresSubmissionRepository) GetByID(ctx context.Context, id string) (*domain.Submission, error) {
	query := `
		SELECT id, contestant_id, team_name, language, status, version, dockerfile,
		       checksum, original_filename, file_size, storage_path, idempotency_key, metadata, created_at, updated_at
		FROM submissions
		WHERE id = $1 AND status != 'deleted'
	`

	var submission domain.Submission
	var metadataJSON []byte
	var idempotencyKey sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&submission.ID,
		&submission.ContestantID,
		&submission.TeamName,
		&submission.Language,
		&submission.Status,
		&submission.Version,
		&submission.Dockerfile,
		&submission.Checksum,
		&submission.OriginalFilename,
		&submission.FileSize,
		&submission.StoragePath,
		&idempotencyKey,
		&metadataJSON,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: submission %s", domain.ErrNotFound, id)
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}

	if idempotencyKey.Valid {
		submission.IdempotencyKey = idempotencyKey.String
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &submission.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &submission, nil
}

// List returns submissions WITHOUT loading code_archive (BYTEA) to avoid massive I/O overhead.
func (r *postgresSubmissionRepository) List(ctx context.Context, contestantID string, status string, limit, offset int) ([]*domain.Submission, error) {
	query := `
		SELECT id, contestant_id, team_name, language, status, version, dockerfile,
		       checksum, original_filename, file_size, storage_path, metadata, created_at, updated_at
		FROM submissions
		WHERE status != 'deleted'
	`
	args := []interface{}{}
	argPos := 1

	if contestantID != "" {
		query += fmt.Sprintf(" AND contestant_id = $%d", argPos)
		args = append(args, contestantID)
		argPos++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list submissions: %w", err)
	}
	defer rows.Close()

	var submissions []*domain.Submission
	for rows.Next() {
		var submission domain.Submission
		var metadataJSON []byte

		err := rows.Scan(
			&submission.ID,
			&submission.ContestantID,
			&submission.TeamName,
			&submission.Language,
			&submission.Status,
			&submission.Version,
			&submission.Dockerfile,
			&submission.Checksum,
			&submission.OriginalFilename,
			&submission.FileSize,
			&submission.StoragePath,
			&metadataJSON,
			&submission.CreatedAt,
			&submission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &submission.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		submissions = append(submissions, &submission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate submissions: %w", err)
	}

	return submissions, nil
}

func (r *postgresSubmissionRepository) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
	query := `
		UPDATE submissions
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status != 'deleted'
	`

	result, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: submission %s", domain.ErrNotFound, id)
	}

	return nil
}

func (r *postgresSubmissionRepository) UpdateBenchmarkResult(ctx context.Context, result *domain.BenchmarkResult) error {
	query := `
		INSERT INTO benchmark_results (id, submission_id, tps, p50_latency_ms, p90_latency_ms, p99_latency_ms, correctness_score, total_orders, failed_orders, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (submission_id) DO UPDATE SET
			tps = $3,
			p50_latency_ms = $4,
			p90_latency_ms = $5,
			p99_latency_ms = $6,
			correctness_score = $7,
			total_orders = $8,
			failed_orders = $9,
			created_at = $10
	`

	_, err := r.db.ExecContext(ctx, query,
		result.ID,
		result.SubmissionID,
		result.TPS,
		result.P50LatencyMs,
		result.P90LatencyMs,
		result.P99LatencyMs,
		result.CorrectnessScore,
		result.TotalOrders,
		result.FailedOrders,
		result.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update benchmark result: %w", err)
	}

	return nil
}

// SoftDelete sets a submission's status to 'deleted' instead of removing the row.
func (r *postgresSubmissionRepository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE submissions
		SET status = 'deleted', updated_at = $1
		WHERE id = $2 AND status != 'deleted'
	`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to soft delete submission: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: submission %s", domain.ErrNotFound, id)
	}

	return nil
}

// GetLatestVersion returns the highest version number for a contestant's submissions.
// Returns 0 if no submissions exist.
func (r *postgresSubmissionRepository) GetLatestVersion(ctx context.Context, contestantID string) (int, error) {
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM submissions
		WHERE contestant_id = $1 AND status != 'deleted'
	`

	var version int
	if err := r.db.QueryRowContext(ctx, query, contestantID).Scan(&version); err != nil {
		return 0, fmt.Errorf("failed to get latest version: %w", err)
	}

	return version, nil
}

// GetByIdempotencyKey returns a submission matching the given idempotency key, or nil if none.
func (r *postgresSubmissionRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.Submission, error) {
	if key == "" {
		return nil, nil
	}

	query := `
		SELECT id, contestant_id, team_name, language, status, version, dockerfile,
		       checksum, original_filename, file_size, storage_path, idempotency_key, metadata, created_at, updated_at
		FROM submissions
		WHERE idempotency_key = $1 AND status != 'deleted'
		LIMIT 1
	`

	var submission domain.Submission
	var metadataJSON []byte
	var idempotencyKey sql.NullString

	err := r.db.QueryRowContext(ctx, query, key).Scan(
		&submission.ID,
		&submission.ContestantID,
		&submission.TeamName,
		&submission.Language,
		&submission.Status,
		&submission.Version,
		&submission.Dockerfile,
		&submission.Checksum,
		&submission.OriginalFilename,
		&submission.FileSize,
		&submission.StoragePath,
		&idempotencyKey,
		&metadataJSON,
		&submission.CreatedAt,
		&submission.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No duplicate — this is not an error
		}
		return nil, fmt.Errorf("failed to get by idempotency key: %w", err)
	}

	if idempotencyKey.Valid {
		submission.IdempotencyKey = idempotencyKey.String
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &submission.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &submission, nil
}

// CreateSubmissionLog persists a log entry associated with a submission.
func (r *postgresSubmissionRepository) CreateSubmissionLog(ctx context.Context, log *domain.SubmissionLog) error {
	metadataJSON, err := marshalStringMap(log.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal log metadata: %w", err)
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

// ListSubmissionLogs returns paginated logs for one submission, newest first.
func (r *postgresSubmissionRepository) ListSubmissionLogs(ctx context.Context, submissionID string, limit, offset int) ([]*domain.SubmissionLog, error) {
	query := `
		SELECT id, submission_id, log_type, message, level, metadata, created_at
		FROM submission_logs
		WHERE submission_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, submissionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list submission logs: %w", err)
	}
	defer rows.Close()

	logs := make([]*domain.SubmissionLog, 0, limit)
	for rows.Next() {
		var log domain.SubmissionLog
		var metadataJSON []byte
		if err := rows.Scan(
			&log.ID,
			&log.SubmissionID,
			&log.LogType,
			&log.Message,
			&log.Level,
			&metadataJSON,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan submission log: %w", err)
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal log metadata: %w", err)
			}
		}
		if log.Metadata == nil {
			log.Metadata = map[string]string{}
		}
		logs = append(logs, &log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate submission logs: %w", err)
	}

	return logs, nil
}

// nullString converts a Go string to a sql.NullString for nullable VARCHAR columns.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func marshalStringMap(values map[string]string) ([]byte, error) {
	if values == nil {
		values = map[string]string{}
	}
	return json.Marshal(values)
}
