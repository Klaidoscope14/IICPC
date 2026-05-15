package service

import (
	"context"

	"github.com/iicpc/validation-service-go/internal/domain"
)

// ValidationRepository handles persistence of validation results.
type ValidationRepository interface {
	SaveResult(ctx context.Context, result *domain.ValidationResult) error
	GetResult(ctx context.Context, submissionID string) (*domain.ValidationResult, error)
	UpdateStatus(ctx context.Context, submissionID string, status domain.ValidationStatus) error
	GetSubmissionStoragePath(ctx context.Context, submissionID string) (string, error)
}

// StorageClient defines the interface for downloading submission archives.
type StorageClient interface {
	DownloadArchive(ctx context.Context, storagePath string, destinationPath string) error
}
