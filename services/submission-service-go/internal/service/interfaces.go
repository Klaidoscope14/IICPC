package service

import (
	"context"

	"github.com/iicpc/submission-service-go/internal/domain"
)

// SubmissionRepository defines the persistence contract for submissions.
// Defined in the service package (consumer) per Dependency Inversion Principle —
// the repository package provides implementations.
type SubmissionRepository interface {
	Create(ctx context.Context, submission *domain.Submission) error
	GetByID(ctx context.Context, id string) (*domain.Submission, error)
	List(ctx context.Context, contestantID string, status string, limit, offset int) ([]*domain.Submission, error)
	UpdateStatus(ctx context.Context, id string, status domain.SubmissionStatus) error
	UpdateBenchmarkResult(ctx context.Context, result *domain.BenchmarkResult) error
}
