package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/submission-service-go/internal/domain"
	"github.com/iicpc/submission-service-go/internal/service"
)

// SubmissionHandler handles HTTP requests for submission operations.
type SubmissionHandler struct {
	service service.SubmissionService
}

// NewSubmissionHandler creates a handler wired to the given service.
func NewSubmissionHandler(service service.SubmissionService) *SubmissionHandler {
	return &SubmissionHandler{service: service}
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
		}
	}
}

// --- Request / Response DTOs ---

// UploadSubmissionRequest is the handler-level DTO with binding validation tags.
type UploadSubmissionRequest struct {
	ContestantID string            `json:"contestant_id" binding:"required"`
	TeamName     string            `json:"team_name" binding:"required"`
	Language     string            `json:"language" binding:"required"`
	CodeArchive  []byte            `json:"code_archive" binding:"required"`
	Dockerfile   string            `json:"dockerfile"`
	Metadata     map[string]string `json:"metadata"`
}

// UploadSubmissionResponse is returned on successful submission creation.
type UploadSubmissionResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
}

// UpdateStatusRequest is the DTO for status update requests.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// --- Handlers ---

func (h *SubmissionHandler) UploadSubmission(c *gin.Context) {
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
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

	// Handle file upload
	file, err := c.FormFile("code_archive")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code_archive file is required"})
		return
	}

	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
		return
	}
	defer fileContent.Close()

	// Read file into memory (for both DB storage and later processing)
	// In a real system, we'd stream this directly to Object Storage.
	codeBytes, err := io.ReadAll(fileContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file contents"})
		return
	}

	params := &domain.CreateSubmissionParams{
		ContestantID: contestantID,
		TeamName:     teamName,
		Language:     language,
		CodeArchive:  codeBytes,
		Dockerfile:   dockerfile,
		Metadata:     make(map[string]string),
	}

	submission, err := h.service.UploadSubmission(c.Request.Context(), params)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := UploadSubmissionResponse{
		ID:        submission.ID,
		Status:    string(submission.Status),
		CreatedAt: submission.CreatedAt.Unix(),
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *SubmissionHandler) GetSubmission(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	submission, err := h.service.GetSubmission(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, submission)
}

func (h *SubmissionHandler) ListSubmissions(c *gin.Context) {
	contestantID := c.Query("contestant_id")
	status := c.Query("status")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	submissions, err := h.service.ListSubmissions(c.Request.Context(), contestantID, status, page, pageSize)
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

	if err := h.service.UpdateSubmissionStatus(c.Request.Context(), id, domain.SubmissionStatus(req.Status)); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
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
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
