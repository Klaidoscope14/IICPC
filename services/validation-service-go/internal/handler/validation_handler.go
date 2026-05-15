package handler

import (
	"context"
	"net/http"

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

// TriggerValidation manually triggers validation (useful for retries or admin overrides).
func (h *ValidationHandler) TriggerValidation(c *gin.Context) {
	submissionID := c.Param("id")

	// Trigger async to avoid blocking HTTP request
	go func() {
		// Create a detached context since the request context will cancel when handler returns
		_ = h.valService.ValidateSubmission(context.Background(), submissionID)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":       "Validation triggered",
		"submission_id": submissionID,
	})
}
