package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/submission-service-go/internal/domain"
	"github.com/iicpc/submission-service-go/internal/publisher"
	"github.com/iicpc/submission-service-go/internal/storage"
	"github.com/iicpc/submission-service-go/internal/validation"
)

// SubmissionService defines the business logic contract for submission operations.
type SubmissionService interface {
	UploadSubmission(ctx context.Context, req *domain.CreateSubmissionParams) (*domain.Submission, error)
	GetSubmission(ctx context.Context, id string) (*domain.Submission, error)
	ListSubmissions(ctx context.Context, contestantID string, status string, page, pageSize int) ([]*domain.Submission, error)
	UpdateSubmissionStatus(ctx context.Context, id string, status domain.SubmissionStatus) error
	DeleteSubmission(ctx context.Context, id string) error
}

type submissionService struct {
	repo           SubmissionRepository
	producer       *events.Producer
	redisPublisher *publisher.RedisPublisher
	storage        storage.Storage
	validator      *validation.UploadValidator
	logger         *slog.Logger
}

// NewSubmissionService creates a SubmissionService with the given dependencies.
func NewSubmissionService(
	repo SubmissionRepository,
	producer *events.Producer,
	redisPublisher *publisher.RedisPublisher,
	storage storage.Storage,
	validator *validation.UploadValidator,
	logger *slog.Logger,
) SubmissionService {
	return &submissionService{
		repo:           repo,
		producer:       producer,
		redisPublisher: redisPublisher,
		storage:        storage,
		validator:      validator,
		logger:         logger,
	}
}

func (s *submissionService) UploadSubmission(ctx context.Context, req *domain.CreateSubmissionParams) (*domain.Submission, error) {
	// --- Input validation ---
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

	// --- Idempotency check ---
	if req.IdempotencyKey != "" {
		existing, err := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("%w: idempotency lookup failed: %v", domain.ErrInternal, err)
		}
		if existing != nil {
			s.logger.Info("idempotent duplicate detected, returning cached submission",
				slog.String("submission_id", existing.ID),
				slog.String("idempotency_key", req.IdempotencyKey),
			)
			return existing, nil
		}
	}

	// --- File size validation ---
	if err := s.validator.ValidateFileSize(req.FileSize); err != nil {
		return nil, err
	}

	// --- Sanitize filename ---
	sanitizedFilename := validation.SanitizeFilename(req.OriginalFilename)

	// --- Auto-versioning ---
	latestVersion, err := s.repo.GetLatestVersion(ctx, req.ContestantID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to determine version: %v", domain.ErrInternal, err)
	}
	nextVersion := latestVersion + 1

	submissionID := uuid.New().String()
	now := time.Now()

	// --- Streaming validation + checksum + storage write ---
	objectKey := fmt.Sprintf("submissions/%s/v%d/%s.zip", req.ContestantID, nextVersion, submissionID)

	// We use a pipe to stream validated content directly into storage without buffering.
	// The validator reads from the archive, validates ZIP magic bytes + MIME,
	// computes SHA-256, and writes to the pipe. Storage reads from the pipe.
	var storageBuf bytes.Buffer
	checksum, bytesWritten, err := s.validator.ValidateAndHash(
		bytes.NewReader(req.CodeArchive),
		&storageBuf,
	)
	if err != nil {
		return nil, err // Already wrapped with domain errors by validator
	}

	// Write validated content to storage.
	storagePath, err := s.storage.Save(ctx, objectKey, &storageBuf)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to save to storage: %v", domain.ErrInternal, err)
	}

	submission := &domain.Submission{
		ID:               submissionID,
		ContestantID:     req.ContestantID,
		TeamName:         req.TeamName,
		Language:         req.Language,
		Status:           domain.StatusPending,
		Version:          nextVersion,
		CodeArchive:      req.CodeArchive,
		Dockerfile:       req.Dockerfile,
		Checksum:         checksum,
		OriginalFilename: sanitizedFilename,
		FileSize:         bytesWritten,
		StoragePath:      storagePath,
		IdempotencyKey:   req.IdempotencyKey,
		Metadata:         req.Metadata,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// --- Persist to database ---
	if err := s.repo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	// --- Log the upload event ---
	uploadLog := &domain.SubmissionLog{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		LogType:      "upload",
		Message:      fmt.Sprintf("Uploaded %s (v%d, %d bytes, checksum: %s)", sanitizedFilename, nextVersion, bytesWritten, checksum),
		Level:        "info",
		CreatedAt:    now,
	}
	if err := s.repo.CreateSubmissionLog(ctx, uploadLog); err != nil {
		// Log but don't fail the upload for a log persistence error.
		s.logger.Warn("failed to persist upload log", slog.String("error", err.Error()))
	}

	// --- Publish events asynchronously (fire-and-forget, don't block response) ---
	go s.publishEvents(submission)

	s.logger.Info("submission uploaded successfully",
		slog.String("submission_id", submissionID),
		slog.String("contestant_id", req.ContestantID),
		slog.Int("version", nextVersion),
		slog.String("checksum", checksum),
		slog.Int64("file_size", bytesWritten),
	)

	return submission, nil
}

// publishEvents fires events to both Redpanda (durable pipeline) and Redis (real-time notifications).
func (s *submissionService) publishEvents(submission *domain.Submission) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Redpanda: durable event for downstream services (orchestrator).
	if s.producer != nil {
		event := events.SubmissionCreatedEvent{
			SubmissionID:   submission.ID,
			TeamName:       submission.TeamName,
			Language:       submission.Language,
			ContainerImage: fmt.Sprintf("iicpc/submission-%s:latest", submission.ID[:8]),
			CreatedAt:      submission.CreatedAt,
		}
		if err := s.producer.PublishSubmissionCreated(ctx, event); err != nil {
			s.logger.Error("failed to publish redpanda event",
				slog.String("submission_id", submission.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	// Redis: lightweight notification for frontend dashboard.
	if s.redisPublisher != nil {
		redisEvent := publisher.SubmissionEvent{
			SubmissionID: submission.ID,
			ContestantID: submission.ContestantID,
			TeamName:     submission.TeamName,
			Status:       string(submission.Status),
			Version:      submission.Version,
		}
		if err := s.redisPublisher.PublishSubmissionCreated(ctx, redisEvent); err != nil {
			s.logger.Error("failed to publish redis event",
				slog.String("submission_id", submission.ID),
				slog.String("error", err.Error()),
			)
		}
	}
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

	// Publish status change notification via Redis.
	if s.redisPublisher != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			redisEvent := publisher.SubmissionEvent{
				SubmissionID: id,
				Status:       string(status),
			}
			if err := s.redisPublisher.PublishSubmissionStatus(ctx, redisEvent); err != nil {
				s.logger.Warn("failed to publish status change to redis", slog.String("error", err.Error()))
			}
		}()
	}

	return nil
}

// DeleteSubmission performs a soft-delete on a submission and publishes a deletion event.
func (s *submissionService) DeleteSubmission(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", domain.ErrInvalidInput)
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		return err
	}

	s.logger.Info("submission soft-deleted", slog.String("submission_id", id))

	// Publish deletion notification via Redis.
	if s.redisPublisher != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			redisEvent := publisher.SubmissionEvent{
				SubmissionID: id,
				Status:       string(domain.StatusDeleted),
			}
			if err := s.redisPublisher.PublishSubmissionDeleted(ctx, redisEvent); err != nil {
				s.logger.Warn("failed to publish deletion event to redis", slog.String("error", err.Error()))
			}
		}()
	}

	return nil
}

