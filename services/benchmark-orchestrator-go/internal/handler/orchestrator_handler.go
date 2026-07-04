package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
	"github.com/iicpc/benchmark-orchestrator-go/internal/service"
)

// OrchestratorHandler handles HTTP requests for deployment and benchmark operations.
type OrchestratorHandler struct {
	service service.OrchestratorService
}

// NewOrchestratorHandler creates a handler wired to the given service.
func NewOrchestratorHandler(service service.OrchestratorService) *OrchestratorHandler {
	return &OrchestratorHandler{service: service}
}

// RegisterRoutes binds all orchestrator endpoints to the given router.
func (h *OrchestratorHandler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api/v1")
	{
		deployments := api.Group("/deployments")
		{
			deployments.POST("", h.DeploySubmission)
		}

		benchmarks := api.Group("/benchmarks")
		{
			benchmarks.POST("", h.StartBenchmark)
			benchmarks.GET("/:id", h.GetBenchmarkStatus)
			benchmarks.POST("/:id/stop", h.StopBenchmark)
		}

		api.GET("/leaderboard", h.GetLeaderboard)
	}
}

// --- Request / Response DTOs ---

type DeploySubmissionRequest struct {
	SubmissionID   string                `json:"submission_id" binding:"required"`
	ContainerImage string                `json:"container_image" binding:"required"`
	ExposedPorts   []string              `json:"exposed_ports"`
	ResourceLimits domain.ResourceLimits `json:"resource_limits"`
}

type DeploySubmissionResponse struct {
	DeploymentID string `json:"deployment_id"`
	ServiceURL   string `json:"service_url"`
	Status       string `json:"status"`
}

type StartBenchmarkRequest struct {
	SubmissionID string                 `json:"submission_id" binding:"required"`
	DeploymentID string                 `json:"deployment_id" binding:"required"`
	Config       domain.BenchmarkConfig `json:"config"`
}

type StartBenchmarkResponse struct {
	BenchmarkID string `json:"benchmark_id"`
	Status      string `json:"status"`
	StartedAt   int64  `json:"started_at"`
}

// --- Handlers ---

func (h *OrchestratorHandler) DeploySubmission(c *gin.Context) {
	var req DeploySubmissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply default resource limits.
	if req.ResourceLimits.CPUMilli == 0 {
		req.ResourceLimits.CPUMilli = 1000
	}
	if req.ResourceLimits.MemoryMB == 0 {
		req.ResourceLimits.MemoryMB = 512
	}
	if req.ResourceLimits.TimeoutSeconds == 0 {
		req.ResourceLimits.TimeoutSeconds = 300
	}
	if len(req.ExposedPorts) == 0 {
		req.ExposedPorts = []string{"8080"}
	}

	deployment, err := h.service.DeploySubmission(c.Request.Context(), req.SubmissionID, req.ContainerImage, req.ExposedPorts, req.ResourceLimits)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := DeploySubmissionResponse{
		DeploymentID: deployment.ID,
		ServiceURL:   deployment.ServiceURL,
		Status:       string(deployment.Status),
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *OrchestratorHandler) StartBenchmark(c *gin.Context) {
	var req StartBenchmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Apply default benchmark config.
	if req.Config.BotCount == 0 {
		req.Config.BotCount = 100
	}
	if req.Config.DurationSeconds == 0 {
		req.Config.DurationSeconds = 60
	}
	if req.Config.OrdersPerSecond == 0 {
		req.Config.OrdersPerSecond = 1000
	}
	if len(req.Config.Protocols) == 0 {
		req.Config.Protocols = []string{"rest"}
	}

	benchmark, err := h.service.StartBenchmark(c.Request.Context(), req.SubmissionID, req.DeploymentID, req.Config)
	if err != nil {
		h.handleError(c, err)
		return
	}

	resp := StartBenchmarkResponse{
		BenchmarkID: benchmark.ID,
		Status:      string(benchmark.Status),
		StartedAt:   benchmark.StartedAt.Unix(),
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *OrchestratorHandler) GetBenchmarkStatus(c *gin.Context) {
	benchmarkID := c.Param("id")
	if benchmarkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "benchmark_id is required"})
		return
	}

	benchmark, err := h.service.GetBenchmarkStatus(c.Request.Context(), benchmarkID)
	if err != nil {
		// Fallback: Try looking it up as a submission_id instead
		benchmarks, listErr := h.service.ListBenchmarksBySubmission(c.Request.Context(), benchmarkID)
		if listErr == nil && len(benchmarks) > 0 {
			benchmark = benchmarks[0]
		} else {
			h.handleError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, benchmark)
}

func (h *OrchestratorHandler) StopBenchmark(c *gin.Context) {
	benchmarkID := c.Param("id")
	if benchmarkID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "benchmark_id is required"})
		return
	}

	if err := h.service.StopBenchmark(c.Request.Context(), benchmarkID); err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *OrchestratorHandler) GetLeaderboard(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	entries, err := h.service.GetLeaderboard(c.Request.Context(), limit)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": entries,
		"count":       len(entries),
	})
}

// handleError maps domain errors to appropriate HTTP status codes.
func (h *OrchestratorHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
