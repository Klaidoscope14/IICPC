package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/iicpc/validation-service-go/internal/domain"
	"github.com/iicpc/validation-service-go/internal/service"
	"github.com/jmoiron/sqlx"
)

type postgresValidationRepository struct {
	db *sqlx.DB
}

func NewPostgresValidationRepository(db *sqlx.DB) service.ValidationRepository {
	return &postgresValidationRepository{db: db}
}

type validationResultRow struct {
	ID           string                  `db:"id"`
	SubmissionID string                  `db:"submission_id"`
	Status       domain.ValidationStatus `db:"status"`
	Language     sql.NullString          `db:"language"`
	Runtime      sql.NullString          `db:"runtime"`
	ErrorsJSON   []byte                  `db:"errors"`
	WarningsJSON []byte                  `db:"warnings"`
	ReportJSON   []byte                  `db:"report"`
	ValidatedAt  sql.NullTime            `db:"validated_at"`
	CreatedAt    time.Time               `db:"created_at"`
	UpdatedAt    time.Time               `db:"updated_at"`
}

func mapRowToResult(row validationResultRow) *domain.ValidationResult {
	result := &domain.ValidationResult{
		ID:           row.ID,
		SubmissionID: row.SubmissionID,
		Status:       row.Status,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}

	if row.Language.Valid {
		result.Language = row.Language.String
	}
	if row.Runtime.Valid {
		result.Runtime = row.Runtime.String
	}
	if row.ValidatedAt.Valid {
		result.ValidatedAt = &row.ValidatedAt.Time
	}

	if len(row.ErrorsJSON) > 0 {
		_ = json.Unmarshal(row.ErrorsJSON, &result.Errors)
	}
	if len(row.WarningsJSON) > 0 {
		_ = json.Unmarshal(row.WarningsJSON, &result.Warnings)
	}
	if len(row.ReportJSON) > 0 {
		_ = json.Unmarshal(row.ReportJSON, &result.Report)
	}

	return result
}

func (r *postgresValidationRepository) SaveResult(ctx context.Context, result *domain.ValidationResult) error {
	if result.ID == "" {
		result.ID = uuid.NewString()
	}

	query := `
		INSERT INTO validation_results (
			id, submission_id, status, language, runtime, errors, warnings, report, validated_at, created_at, updated_at
		) VALUES (
			:id, :submission_id, :status, :language, :runtime, :errors, :warnings, :report, :validated_at, NOW(), NOW()
		)
		ON CONFLICT (submission_id) DO UPDATE SET
			id = EXCLUDED.id,
			status = EXCLUDED.status,
			language = EXCLUDED.language,
			runtime = EXCLUDED.runtime,
			errors = EXCLUDED.errors,
			warnings = EXCLUDED.warnings,
			report = EXCLUDED.report,
			validated_at = EXCLUDED.validated_at,
			updated_at = NOW()
	`

	// Serialize JSON fields
	errorsJSON, err := json.Marshal(result.Errors)
	if err != nil {
		return err
	}
	warningsJSON, err := json.Marshal(result.Warnings)
	if err != nil {
		return err
	}
	reportJSON, err := json.Marshal(result.Report)
	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"id":            result.ID,
		"submission_id": result.SubmissionID,
		"status":        result.Status,
		"language":      result.Language,
		"runtime":       result.Runtime,
		"errors":        errorsJSON,
		"warnings":      warningsJSON,
		"report":        reportJSON,
		"validated_at":  result.ValidatedAt,
	}

	_, err = r.db.NamedExecContext(ctx, query, params)
	return err
}

func (r *postgresValidationRepository) GetResult(ctx context.Context, submissionID string) (*domain.ValidationResult, error) {
	query := `SELECT * FROM validation_results WHERE submission_id = $1 ORDER BY created_at DESC LIMIT 1`

	var row validationResultRow
	err := r.db.GetContext(ctx, &row, query, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return mapRowToResult(row), nil
}

func (r *postgresValidationRepository) ListResults(ctx context.Context, limit int, cursor string) ([]*domain.ValidationResult, string, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var query string
	var args []interface{}

	// We fetch limit + 1 to determine if there is a next page
	fetchLimit := limit + 1

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return nil, "", err
		}
		query = `SELECT * FROM validation_results WHERE created_at < $1 ORDER BY created_at DESC LIMIT $2`
		args = []interface{}{cursorTime, fetchLimit}
	} else {
		query = `SELECT * FROM validation_results ORDER BY created_at DESC LIMIT $1`
		args = []interface{}{fetchLimit}
	}

	var rows []validationResultRow
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(rows) > limit {
		nextCursor = rows[limit].CreatedAt.Format(time.RFC3339Nano)
		rows = rows[:limit]
	}

	results := make([]*domain.ValidationResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, mapRowToResult(row))
	}

	return results, nextCursor, nil
}

func (r *postgresValidationRepository) UpdateStatus(ctx context.Context, submissionID string, status domain.ValidationStatus) error {
	query := `
		INSERT INTO validation_results (id, submission_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (submission_id) DO UPDATE SET
			status = EXCLUDED.status,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, uuid.NewString(), submissionID, status)
	return err
}

func (r *postgresValidationRepository) GetSubmissionStoragePath(ctx context.Context, submissionID string) (string, error) {
	query := `SELECT storage_path FROM submissions WHERE id = $1`
	var storagePath string
	err := r.db.GetContext(ctx, &storagePath, query, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", domain.ErrNotFound
		}
		return "", err
	}
	return storagePath, nil
}

func (r *postgresValidationRepository) UpdateSubmissionStatus(ctx context.Context, submissionID string, status string) error {
	query := `UPDATE submissions SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, status, submissionID)
	return err
}
