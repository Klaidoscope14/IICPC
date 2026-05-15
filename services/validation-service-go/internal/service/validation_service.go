package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	contractvalidation "github.com/iicpc/pkg/contracts/validation"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/validation-service-go/internal/domain"
	"github.com/iicpc/validation-service-go/internal/extractor"
	"github.com/iicpc/validation-service-go/internal/validator"
)

type ValidationService struct {
	repo      ValidationRepository
	storage   StorageClient
	contract  *domain.SubmissionContract
	pipeline  *validator.Pipeline
	extractor *extractor.Extractor
	producer  *events.Producer
}

func NewValidationService(
	repo ValidationRepository,
	storage StorageClient,
	contract *domain.SubmissionContract,
	eventProducer *events.Producer,
) *ValidationService {
	return &ValidationService{
		repo:      repo,
		storage:   storage,
		contract:  contract,
		pipeline:  validator.NewPipeline(contract),
		extractor: extractor.NewExtractor(contract.MaxExtractedBytes, contract.MaxFileCount),
		producer:  eventProducer,
	}
}

// Contract returns the active submission contract used by the validation engine.
func (s *ValidationService) Contract() domain.SubmissionContract {
	if s.contract == nil {
		return domain.DefaultContract
	}
	contract := *s.contract
	return contract
}

// ValidateSubmission handles the full lifecycle of validating a submission.
func (s *ValidationService) ValidateSubmission(ctx context.Context, submissionID string) error {
	slog.Info("Starting validation for submission", "submission_id", submissionID)

	// 1. Mark as running
	err := s.repo.UpdateStatus(ctx, submissionID, domain.ValidationRunning)
	if err != nil && err != domain.ErrNotFound {
		// If it's not found, we might be creating it for the first time later, that's fine.
		slog.Warn("Failed to update status to running", "error", err)
	}

	// 2. Fetch storage path
	storagePath, err := s.repo.GetSubmissionStoragePath(ctx, submissionID)
	if err != nil {
		return s.failValidation(ctx, submissionID, "STORAGE_ERROR", fmt.Sprintf("Failed to get storage path: %v", err))
	}

	// 3. Download to temp file
	tempZipPath := filepath.Join(os.TempDir(), fmt.Sprintf("val-%s.zip", submissionID))
	defer os.Remove(tempZipPath)

	if err := s.storage.DownloadArchive(ctx, storagePath, tempZipPath); err != nil {
		return s.failValidation(ctx, submissionID, "DOWNLOAD_ERROR", fmt.Sprintf("Failed to download archive: %v", err))
	}

	// 4. Extract
	extractResult, err := s.extractor.Extract(tempZipPath)
	if err != nil {
		return s.failValidation(ctx, submissionID, "EXTRACTION_ERROR", err.Error())
	}
	defer extractor.Cleanup(extractResult.RootDir)

	// 5. Run Validation Pipeline
	report := s.pipeline.Run(submissionID, extractResult.RootDir)

	// 6. Save Result
	now := time.Now()
	valResult := &domain.ValidationResult{
		SubmissionID: submissionID,
		Status:       report.Status,
		Language:     report.Language,
		Runtime:      report.Runtime,
		Report:       report,
		ValidatedAt:  &now,
	}

	// Aggregate errors/warnings for top-level access
	for _, check := range report.CheckResults {
		valResult.Errors = append(valResult.Errors, check.Errors...)
		valResult.Warnings = append(valResult.Warnings, check.Warnings...)
	}

	if err := s.repo.SaveResult(ctx, valResult); err != nil {
		slog.Error("Failed to save validation result", "error", err, "submission_id", submissionID)
		return err
	}

	// 7. Publish Event
	event := events.ValidationCompletedEvent{
		SubmissionID: submissionID,
		Status:       string(report.Status),
		Language:     report.Language,
		Runtime:      report.Runtime,
		ErrorCount:   report.TotalErrors,
		WarningCount: report.TotalWarnings,
		Errors:       toContractFindings(valResult.Errors),
		Warnings:     toContractFindings(valResult.Warnings),
		ValidatedAt:  now,
	}

	if err := s.producer.PublishValidationCompleted(ctx, event); err != nil {
		slog.Error("Failed to publish validation event", "error", err, "submission_id", submissionID)
		// We don't fail the transaction here since the DB is updated, but it might need a retry mechanism later.
	}

	slog.Info("Validation completed", "submission_id", submissionID, "status", report.Status, "errors", report.TotalErrors)
	return nil
}

// failValidation is a helper to immediately fail a validation if pre-pipeline steps fail (e.g. extraction size bomb).
func (s *ValidationService) failValidation(ctx context.Context, submissionID, code, message string) error {
	now := time.Now()

	report := &domain.ValidationReport{
		SubmissionID: submissionID,
		Status:       domain.ValidationFailed,
		Language:     "unknown",
		Runtime:      "unknown",
		CheckResults: map[string]domain.CheckResult{
			"pre_flight": {
				Name:   "pre_flight",
				Passed: false,
				Errors: []domain.ValidationError{{
					Code:     code,
					Message:  message,
					Severity: domain.SeverityError,
				}},
			},
		},
		TotalErrors: 1,
	}

	valResult := &domain.ValidationResult{
		SubmissionID: submissionID,
		Status:       domain.ValidationFailed,
		Language:     "unknown",
		Runtime:      "unknown",
		Report:       report,
		Errors:       report.CheckResults["pre_flight"].Errors,
		ValidatedAt:  &now,
	}

	if err := s.repo.SaveResult(ctx, valResult); err != nil {
		slog.Error("Failed to save failed validation result", "error", err)
		return err
	}

	// Publish failure event
	event := events.ValidationCompletedEvent{
		SubmissionID: submissionID,
		Status:       string(domain.ValidationFailed),
		Language:     "unknown",
		Runtime:      "unknown",
		ErrorCount:   1,
		WarningCount: 0,
		Errors:       toContractFindings(report.CheckResults["pre_flight"].Errors),
		ValidatedAt:  now,
	}
	_ = s.producer.PublishValidationCompleted(ctx, event)

	slog.Error("Validation failed during pre-flight", "submission_id", submissionID, "code", code, "message", message)
	return fmt.Errorf("validation failed: %s", message)
}

func toContractFindings(errors []domain.ValidationError) []contractvalidation.Finding {
	findings := make([]contractvalidation.Finding, 0, len(errors))
	for _, err := range errors {
		findings = append(findings, contractvalidation.Finding{
			Code:     err.Code,
			Message:  err.Message,
			Severity: contractvalidation.Severity(err.Severity),
			FilePath: err.FilePath,
		})
	}
	return findings
}
