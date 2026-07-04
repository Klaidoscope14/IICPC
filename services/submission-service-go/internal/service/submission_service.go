package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
	GetSubmissionArchive(ctx context.Context, id string) (*domain.Submission, io.ReadCloser, error)
	ListSubmissions(ctx context.Context, contestantID string, status string, page, pageSize int) ([]*domain.Submission, error)
	ListSubmissionLogs(ctx context.Context, submissionID string, page, pageSize int) ([]*domain.SubmissionLog, error)
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
	if req.ArchiveReader == nil && len(req.CodeArchive) == 0 {
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

	submissionID := uuid.New().String()
	now := time.Now()

	// --- Streaming validation + checksum + storage write ---
	contestantStorageKey := validation.SanitizeFilename(req.ContestantID)
	objectKey := fmt.Sprintf("submissions/%s/%s.zip", contestantStorageKey, submissionID)

	archiveReader := req.ArchiveReader
	if archiveReader == nil {
		archiveReader = bytes.NewReader(req.CodeArchive)
	}
	if seekable, ok := archiveReader.(interface {
		io.ReaderAt
		io.Seeker
	}); ok {
		if err := s.validator.ValidateArchiveStructure(seekable, req.FileSize); err != nil {
			return nil, err
		}
		if _, err := seekable.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("%w: failed to rewind archive", domain.ErrInvalidArchive)
		}
	}

	checksum, bytesWritten, storagePath, err := s.validateAndStore(ctx, objectKey, archiveReader)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidArchive) || errors.Is(err, domain.ErrUnsupportedMIME) || errors.Is(err, domain.ErrFileTooLarge) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: failed to save to storage: %v", domain.ErrInternal, err)
	}

	submission := &domain.Submission{
		ID:               submissionID,
		ContestantID:     req.ContestantID,
		TeamName:         req.TeamName,
		Language:         req.Language,
		Status:           domain.StatusUploaded,
		Version:          1,
		CodeArchive:      nil,
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
	if err := s.repo.CreateWithNextVersion(ctx, submission); err != nil {
		if deleteErr := s.storage.Delete(context.Background(), objectKey); deleteErr != nil {
			s.logger.Warn("failed to clean up stored archive after database error",
				slog.String("submission_id", submissionID),
				slog.String("storage_key", objectKey),
				slog.String("error", deleteErr.Error()),
			)
		}
		if req.IdempotencyKey != "" {
			existing, lookupErr := s.repo.GetByIdempotencyKey(ctx, req.IdempotencyKey)
			if lookupErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	// --- Log the upload event ---
	uploadLog := &domain.SubmissionLog{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		LogType:      "upload",
		Message:      fmt.Sprintf("Uploaded %s (v%d, %d bytes, checksum: %s)", sanitizedFilename, submission.Version, bytesWritten, checksum),
		Level:        "info",
		Metadata: map[string]string{
			"checksum":          checksum,
			"original_filename": sanitizedFilename,
			"storage_path":      storagePath,
		},
		CreatedAt: now,
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
		slog.Int("version", submission.Version),
		slog.String("checksum", checksum),
		slog.Int64("file_size", bytesWritten),
	)

	return submission, nil
}

func (s *submissionService) validateAndStore(ctx context.Context, objectKey string, archiveReader io.Reader) (string, int64, string, error) {
	type validationResult struct {
		checksum     string
		bytesWritten int64
		err          error
	}

	pipeReader, pipeWriter := io.Pipe()
	resultCh := make(chan validationResult, 1)

	go func() {
		checksum, bytesWritten, err := s.validator.ValidateAndHash(archiveReader, pipeWriter)
		if err != nil {
			_ = pipeWriter.CloseWithError(err)
		} else {
			_ = pipeWriter.Close()
		}
		resultCh <- validationResult{checksum: checksum, bytesWritten: bytesWritten, err: err}
	}()

	storagePath, storageErr := s.storage.Save(ctx, objectKey, pipeReader)
	if storageErr != nil {
		_ = pipeReader.CloseWithError(storageErr)
	}

	result := <-resultCh
	if result.err != nil {
		return "", 0, "", result.err
	}
	if storageErr != nil {
		return "", 0, "", storageErr
	}

	return result.checksum, result.bytesWritten, storagePath, nil
}

// publishEvents fires events to both Redpanda (durable pipeline) and Redis (real-time notifications).
func (s *submissionService) publishEvents(submission *domain.Submission) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Redpanda: durable event for downstream services (orchestrator).
	if s.producer != nil {
		event := events.SubmissionCreatedEvent{
			SubmissionID:   submission.ID,
			ContestantID:   submission.ContestantID,
			TeamName:       submission.TeamName,
			Language:       submission.Language,
			Version:        submission.Version,
			StoragePath:    submission.StoragePath,
			Checksum:       submission.Checksum,
			ContainerImage: fmt.Sprintf("iicpc/submission-%s:latest", submission.ID[:8]),
			Preset:         submission.Metadata["benchmark_preset"],
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

func (s *submissionService) GetSubmissionArchive(ctx context.Context, id string) (*domain.Submission, io.ReadCloser, error) {
	submission, err := s.GetSubmission(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if submission.StoragePath == "" {
		return nil, nil, fmt.Errorf("%w: submission archive is unavailable", domain.ErrNotFound)
	}

	reader, err := s.storage.Get(ctx, submission.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: failed to open submission archive: %v", domain.ErrInternal, err)
	}
	return submission, reader, nil
}

func (s *submissionService) ListSubmissions(ctx context.Context, contestantID string, status string, page, pageSize int) ([]*domain.Submission, error) {
	if status != "" && !domain.SubmissionStatus(status).IsValid() {
		return nil, fmt.Errorf("%w: invalid status '%s'", domain.ErrInvalidInput, status)
	}
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

func (s *submissionService) ListSubmissionLogs(ctx context.Context, submissionID string, page, pageSize int) ([]*domain.SubmissionLog, error) {
	if submissionID == "" {
		return nil, fmt.Errorf("%w: submission_id is required", domain.ErrInvalidInput)
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize
	logs, err := s.repo.ListSubmissionLogs(ctx, submissionID, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}
	return logs, nil
}

func (s *submissionService) UpdateSubmissionStatus(ctx context.Context, id string, status domain.SubmissionStatus) error {
	if id == "" {
		return fmt.Errorf("%w: id is required", domain.ErrInvalidInput)
	}
	if !status.IsValid() {
		return fmt.Errorf("%w: invalid status '%s'", domain.ErrInvalidInput, status)
	}

	if err := s.repo.UpdateStatus(ctx, id, status); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return err
		}
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

	if s.producer != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			event := events.SubmissionDeletedEvent{
				SubmissionID: id,
				DeletedAt:    time.Now().UTC(),
			}
			if err := s.producer.PublishSubmissionDeleted(ctx, event); err != nil {
				s.logger.Warn("failed to publish deletion event to redpanda", slog.String("error", err.Error()))
			}
		}()
	}

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
