package repository

import (
	"context"
	"database/sql"
	"encoding/json"

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
		ON CONFLICT (id) DO UPDATE SET
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

	type ValidationResultRow struct {
		ID           string                  `db:"id"`
		SubmissionID string                  `db:"submission_id"`
		Status       domain.ValidationStatus `db:"status"`
		Language     sql.NullString          `db:"language"`
		Runtime      sql.NullString          `db:"runtime"`
		ErrorsJSON   []byte                  `db:"errors"`
		WarningsJSON []byte                  `db:"warnings"`
		ReportJSON   []byte                  `db:"report"`
		ValidatedAt  sql.NullTime            `db:"validated_at"`
	}

	var row ValidationResultRow
	err := r.db.GetContext(ctx, &row, query, submissionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	result := &domain.ValidationResult{
		ID:           row.ID,
		SubmissionID: row.SubmissionID,
		Status:       row.Status,
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

	return result, nil
}

func (r *postgresValidationRepository) UpdateStatus(ctx context.Context, submissionID string, status domain.ValidationStatus) error {
	query := `
		UPDATE validation_results 
		SET status = $1, updated_at = NOW()
		WHERE submission_id = $2
	`
	_, err := r.db.ExecContext(ctx, query, status, submissionID)
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

