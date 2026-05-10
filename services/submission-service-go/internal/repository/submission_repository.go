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
	metadataJSON, err := json.Marshal(submission.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO submissions (id, contestant_id, team_name, language, status, code_archive, dockerfile, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = r.db.ExecContext(ctx, query,
		submission.ID,
		submission.ContestantID,
		submission.TeamName,
		submission.Language,
		submission.Status,
		submission.CodeArchive,
		submission.Dockerfile,
		metadataJSON,
		submission.CreatedAt,
		submission.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create submission: %w", err)
	}

	return nil
}

func (r *postgresSubmissionRepository) GetByID(ctx context.Context, id string) (*domain.Submission, error) {
	query := `
		SELECT id, contestant_id, team_name, language, status, code_archive, dockerfile, metadata, created_at, updated_at
		FROM submissions
		WHERE id = $1
	`

	var submission domain.Submission
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&submission.ID,
		&submission.ContestantID,
		&submission.TeamName,
		&submission.Language,
		&submission.Status,
		&submission.CodeArchive,
		&submission.Dockerfile,
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

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &submission.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &submission, nil
}

func (r *postgresSubmissionRepository) List(ctx context.Context, contestantID string, status string, limit, offset int) ([]*domain.Submission, error) {
	query := `
		SELECT id, contestant_id, team_name, language, status, code_archive, dockerfile, metadata, created_at, updated_at
		FROM submissions
		WHERE 1=1
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
			&submission.CodeArchive,
			&submission.Dockerfile,
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

	return submissions, nil
}

func (r *postgresSubmissionRepository) UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
	query := `
		UPDATE submissions
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update submission status: %w", err)
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
