package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/submission-service-go/internal/domain"
	"github.com/iicpc/submission-service-go/internal/storage"
)

// SubmissionService defines the business logic contract for submission operations.
type SubmissionService interface {
	UploadSubmission(ctx context.Context, req *domain.CreateSubmissionParams) (*domain.Submission, error)
	GetSubmission(ctx context.Context, id string) (*domain.Submission, error)
	ListSubmissions(ctx context.Context, contestantID string, status string, page, pageSize int) ([]*domain.Submission, error)
	UpdateSubmissionStatus(ctx context.Context, id string, status domain.SubmissionStatus) error
}

type submissionService struct {
	repo     SubmissionRepository
	producer *events.Producer
	storage  storage.Storage
}

// NewSubmissionService creates a SubmissionService with the given repository, event producer, and storage.
func NewSubmissionService(repo SubmissionRepository, producer *events.Producer, storage storage.Storage) SubmissionService {
	return &submissionService{
		repo:     repo,
		producer: producer,
		storage:  storage,
	}
}

func (s *submissionService) UploadSubmission(ctx context.Context, req *domain.CreateSubmissionParams) (*domain.Submission, error) {
	if req.ContestantID == "" {
		return nil, fmt.Errorf("%w: contestant_id is required", domain.ErrInvalidInput)
	}
	if req.TeamName == "" {
		return nil, fmt.Errorf("%w: team_name is required", domain.ErrInvalidInput)
	}
	if req.Language == "" {
		return nil, fmt.Errorf("%w: language is required", domain.ErrInvalidInput)
	}
	if len(req.CodeArchive) == 0 {
		return nil, fmt.Errorf("%w: code_archive is required", domain.ErrInvalidInput)
	}

	submissionID := uuid.New().String()
	now := time.Now()

	// 1. Save to Object Storage
	objectKey := fmt.Sprintf("submissions/%s/%s.zip", req.ContestantID, submissionID)
	if _, err := s.storage.Save(ctx, objectKey, bytes.NewReader(req.CodeArchive)); err != nil {
		return nil, fmt.Errorf("%w: failed to save to storage: %v", domain.ErrInternal, err)
	}

	submission := &domain.Submission{
		ID:           submissionID,
		ContestantID: req.ContestantID,
		TeamName:     req.TeamName,
		Language:     req.Language,
		Status:       domain.StatusPending,
		CodeArchive:  req.CodeArchive, // Still stored in DB for simplicity right now
		Dockerfile:   req.Dockerfile,
		Metadata:     req.Metadata,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 2. Save to DB
	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	// 3. Publish Event
	if s.producer != nil {
		event := events.SubmissionCreatedEvent{
			SubmissionID:   submission.ID,
			TeamName:       submission.TeamName,
			Language:       submission.Language,
			ContainerImage: fmt.Sprintf("iicpc/submission-%s:latest", submission.ID[:8]), // Target image name
			CreatedAt:      now,
		}
		fmt.Printf("Attempting to publish SubmissionCreatedEvent for %s\n", submission.ID)
		if err := s.producer.PublishSubmissionCreated(ctx, event); err != nil {
			// Log error but don't fail the upload
			fmt.Printf("Failed to publish event: %v\n", err)
		} else {
			fmt.Printf("Successfully published event for %s\n", submission.ID)
		}
	} else {
		fmt.Printf("WARNING: s.producer is nil! Bypassing Redpanda publish for %s\n", submission.ID)
	}

	return submission, nil
}

func (s *submissionService) GetSubmission(ctx context.Context, id string) (*domain.Submission, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: id is required", domain.ErrInvalidInput)
	}

	submission, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return submission, nil
}

func (s *submissionService) ListSubmissions(ctx context.Context, contestantID string, status string, page, pageSize int) ([]*domain.Submission, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	submissions, err := s.repo.List(ctx, contestantID, status, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	return submissions, nil
}

func (s *submissionService) UpdateSubmissionStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", domain.ErrInvalidInput)
	}
	if !status.IsValid() {
		return fmt.Errorf("%w: invalid status '%s'", domain.ErrInvalidInput, status)
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	return nil
}
