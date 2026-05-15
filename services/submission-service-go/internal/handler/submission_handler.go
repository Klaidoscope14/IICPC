package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/submission-service-go/internal/domain"
	"github.com/iicpc/submission-service-go/internal/service"
)

// SubmissionHandler handles HTTP requests for submission operations.
type SubmissionHandler struct {
	service        service.SubmissionService
	maxUploadBytes int64
}

// NewSubmissionHandler creates a handler wired to the given service.
func NewSubmissionHandler(service service.SubmissionService, maxUploadBytes int64) *SubmissionHandler {
	return &SubmissionHandler{
		service:        service,
		maxUploadBytes: maxUploadBytes,
	}
}

// RegisterRoutes binds all submission endpoints to the given router.
func (h *SubmissionHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		submissions := api.Group("/submissions")
		{
			submissions.POST("", h.UploadSubmission)
			submissions.GET("/:id", h.GetSubmission)
			submissions.GET("", h.ListSubmissions)
			submissions.PATCH("/:id/status", h.UpdateStatus)
			submissions.DELETE("/:id", h.DeleteSubmission)
		}
	}
}

// --- Request / Response DTOs ---

// UploadSubmissionResponse is returned on successful submission creation.
type UploadSubmissionResponse struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Version          int    `json:"version"`
	Checksum         string `json:"checksum"`
	FileSize         int64  `json:"file_size"`
	OriginalFilename string `json:"original_filename"`
	CreatedAt        int64  `json:"created_at"`
}

// UpdateStatusRequest is the DTO for status update requests.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// --- Handlers ---

func (h *SubmissionHandler) UploadSubmission(c *gin.Context) {
	// Enforce hard size limit at the HTTP layer BEFORE reading anything.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxUploadBytes)

	// Per-request timeout for uploads (30 seconds).
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Parse multipart form with max upload size.
	if err := c.Request.ParseMultipartForm(h.maxUploadBytes); err != nil {
		if err.Error() == "http: request body too large" {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds maximum upload size"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	contestantID := c.PostForm("contestant_id")
	teamName := c.PostForm("team_name")
	language := c.PostForm("language")
	dockerfile := c.PostForm("dockerfile")

	if contestantID == "" || teamName == "" || language == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contestant_id, team_name, and language are required"})
		return
	}

	// Handle file upload.
	file, fileHeader, err := c.Request.FormFile("code_archive")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_archive file is required"})
		return
	}
	defer file.Close()

	// Read file into memory for validation + storage pipeline.
	// The service layer streams this through validate → hash → write.
	codeBytes := make([]byte, fileHeader.Size)
	n, err := file.Read(codeBytes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
		return
	}
	codeBytes = codeBytes[:n]

	// Extract Idempotency-Key header for retry support.
	idempotencyKey := c.GetHeader("Idempotency-Key")

	params := &domain.CreateSubmissionParams{
		ContestantID:     contestantID,
		TeamName:         teamName,
		Language:         language,
		CodeArchive:      codeBytes,
		Dockerfile:       dockerfile,
		OriginalFilename: fileHeader.Filename,
		FileSize:         fileHeader.Size,
		IdempotencyKey:   idempotencyKey,
		Metadata:         make(map[string]string),
	}

	submission, err := h.service.UploadSubmission(ctx, params)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := UploadSubmissionResponse{
		ID:               submission.ID,
		Status:           string(submission.Status),
		Version:          submission.Version,
		Checksum:         submission.Checksum,
		FileSize:         submission.FileSize,
		OriginalFilename: submission.OriginalFilename,
		CreatedAt:        submission.CreatedAt.Unix(),
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *SubmissionHandler) GetSubmission(c *gin.Context) {
	// Per-request timeout for reads (5 seconds).
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	submission, err := h.service.GetSubmission(ctx, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, submission)
}

func (h *SubmissionHandler) ListSubmissions(c *gin.Context) {
	// Per-request timeout for reads (5 seconds).
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	contestantID := c.Query("contestant_id")
	status := c.Query("status")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	submissions, err := h.service.ListSubmissions(ctx, contestantID, status, page, pageSize)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": submissions,
		"page":        page,
		"page_size":   pageSize,
		"count":       len(submissions),
	})
}

func (h *SubmissionHandler) UpdateStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateSubmissionStatus(ctx, id, domain.SubmissionStatus(req.Status)); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *SubmissionHandler) DeleteSubmission(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if err := h.service.DeleteSubmission(ctx, id); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "submission deleted"})
}

// handleError maps domain errors to appropriate HTTP status codes.
func (h *SubmissionHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrFileTooLarge):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrUnsupportedMIME):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidArchive):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrDuplicateSubmission):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

