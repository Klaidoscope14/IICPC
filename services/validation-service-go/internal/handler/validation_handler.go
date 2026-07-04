package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/validation-service-go/internal/domain"
	"github.com/iicpc/validation-service-go/internal/service"
)

type ValidationHandler struct {
	repo       service.ValidationRepository
	valService *service.ValidationService
}

func NewValidationHandler(repo service.ValidationRepository, valService *service.ValidationService) *ValidationHandler {
	return &ValidationHandler{
		repo:       repo,
		valService: valService,
	}
}

// GetContract returns the active submission contract enforced by the validator.
func (h *ValidationHandler) GetContract(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":  "v1",
		"contract": h.valService.Contract(),
	})
}

// ListResults lists validation results with pagination
func (h *ValidationHandler) ListResults(c *gin.Context) {
	limitStr := c.Query("limit")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 10
	}
	cursor := c.Query("cursor")

	results, nextCursor, err := h.repo.ListResults(c.Request.Context(), limit, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        results,
		"next_cursor": nextCursor,
	})
}

// GetResult fetches the validation result for a submission.
func (h *ValidationHandler) GetResult(c *gin.Context) {
	submissionID := c.Param("id")

	result, err := h.repo.GetResult(c.Request.Context(), submissionID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Validation result not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch result"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetReport fetches only the validation report for a submission.
func (h *ValidationHandler) GetReport(c *gin.Context) {
	submissionID := c.Param("id")

	result, err := h.repo.GetResult(c.Request.Context(), submissionID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Validation result not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch result"})
		return
	}

	if result.Report == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Report not available yet"})
		return
	}

	c.JSON(http.StatusOK, result.Report)
}

// TriggerValidation manually triggers validation (useful for retries or admin overrides).
func (h *ValidationHandler) TriggerValidation(c *gin.Context) {
	submissionID := c.Param("id")

	// Trigger async to avoid blocking HTTP request
	go func() {
		// Create a detached context since the request context will cancel when handler returns
		_ = h.valService.ValidateSubmission(context.Background(), submissionID, "")
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Validation triggered",
		"submission_id": submissionID,
	})
}
