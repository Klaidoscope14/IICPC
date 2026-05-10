package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/iicpc/benchmark-orchestrator-go/internal/container"
	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
	"github.com/iicpc/benchmark-orchestrator-go/internal/repository"
)

// OrchestratorService defines the contract for deployment and benchmark operations.
type OrchestratorService interface {
	DeploySubmission(ctx context.Context, submissionID, containerImage string, ports []string, limits domain.ResourceLimits) (*domain.Deployment, error)
	StartBenchmark(ctx context.Context, submissionID, deploymentID string, config domain.BenchmarkConfig) (*domain.Benchmark, error)
	GetBenchmarkStatus(ctx context.Context, benchmarkID string) (*domain.Benchmark, error)
	StopBenchmark(ctx context.Context, benchmarkID string) error
	GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error)
}

type orchestratorService struct {
	repo           repository.OrchestratorRepository
	scoring        *ScoringService
	logger         *slog.Logger
	containerMgr   container.Manager

	// Track active benchmark goroutines for cancellation.
	mu         sync.Mutex
	cancelFns  map[string]context.CancelFunc
}

// NewOrchestratorService creates a new OrchestratorService with persistent storage and scoring.
func NewOrchestratorService(repo repository.OrchestratorRepository, scoring *ScoringService, containerMgr container.Manager, logger *slog.Logger) OrchestratorService {
	return &orchestratorService{
		repo:         repo,
		scoring:      scoring,
		logger:       logger,
		containerMgr: containerMgr,
		cancelFns:    make(map[string]context.CancelFunc),
	}
}

func (s *orchestratorService) DeploySubmission(ctx context.Context, submissionID, containerImage string, ports []string, limits domain.ResourceLimits) (*domain.Deployment, error) {
	if submissionID == "" {
		return nil, fmt.Errorf("%w: submission_id is required", domain.ErrInvalidInput)
	}
	if containerImage == "" {
		return nil, fmt.Errorf("%w: container_image is required", domain.ErrInvalidInput)
	}

	now := time.Now()
	deployment := &domain.Deployment{
		ID:             uuid.New().String(),
		SubmissionID:   submissionID,
		ContainerImage: containerImage,
		ExposedPorts:   ports,
		ServiceURL:     fmt.Sprintf("http://submission-%s:8080", submissionID[:8]),
		Status:         domain.DeploymentStatusPending,
		ResourceLimits: limits,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	s.logger.Info("deployment created",
		slog.String("deployment_id", deployment.ID),
		slog.String("submission_id", submissionID),
	)

	// Use real container manager instead of simulation.
	go s.executeDeployment(deployment.ID, containerImage, ports, limits)

	return deployment, nil
}

func (s *orchestratorService) executeDeployment(deploymentID, containerImage string, ports []string, limits domain.ResourceLimits) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	s.logger.Info("starting deployment", slog.String("deployment_id", deploymentID))

	// For the simulation, since we haven't implemented the Docker Image Builder worker yet,
	// we will force the use of a lightweight dummy image (alpine) that just sleeps.
	// This allows the orchestrator to actually start a container, manage its lifecycle,
	// and run the benchmark simulation against a real running container process.
	containerImage = "alpine:latest"
	
	opts := container.CreateOptions{
		ImageName:      containerImage,
		ContainerName:  fmt.Sprintf("submission-%s", deploymentID[:8]),
		ExposedPorts:   ports,
		CPUMilli:       limits.CPUMilli,
		MemoryMB:       limits.MemoryMB,
		TimeoutSeconds: 60,
		NetworkMode:    "host", // For simple networking in hackathon
		Cmd:            []string{"sleep", "120"}, // Dummy process
	}

	// Pull the dummy image first
	if err := s.containerMgr.PullImage(ctx, containerImage); err != nil {
		s.logger.Warn("failed to pull image, continuing anyway", slog.String("error", err.Error()))
	}

	containerID, serviceURL, err := s.containerMgr.CreateAndStart(ctx, opts)
	if err != nil {
		s.logger.Error("failed to create container", slog.String("error", err.Error()))
		s.repo.UpdateDeploymentStatus(ctx, deploymentID, domain.DeploymentStatusFailed, "", "", err.Error())
		return
	}

	// Update deployment with success
	err = s.repo.UpdateDeploymentStatus(ctx, deploymentID, domain.DeploymentStatusDeployed, containerID, serviceURL, "")
	if err != nil {
		s.logger.Error("failed to update deployment status",
			slog.String("deployment_id", deploymentID),
			slog.String("error", err.Error()),
		)
	}
}

func (s *orchestratorService) StartBenchmark(ctx context.Context, submissionID, deploymentID string, config domain.BenchmarkConfig) (*domain.Benchmark, error) {
	if submissionID == "" {
		return nil, fmt.Errorf("%w: submission_id is required", domain.ErrInvalidInput)
	}
	if deploymentID == "" {
		return nil, fmt.Errorf("%w: deployment_id is required", domain.ErrInvalidInput)
	}

	now := time.Now()
	benchmark := &domain.Benchmark{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		DeploymentID: deploymentID,
		Status:       domain.BenchmarkStatusPending,
		Config:       config,
		StartedAt:    now,
		Metrics:      domain.TelemetryMetrics{},
	}

	if err := s.repo.CreateBenchmark(ctx, benchmark); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInternal, err)
	}

	// Create a cancellable context for the benchmark goroutine.
	benchCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancelFns[benchmark.ID] = cancel
	s.mu.Unlock()

	s.logger.Info("benchmark started",
		slog.String("benchmark_id", benchmark.ID),
		slog.String("submission_id", submissionID),
	)

	go s.simulateBenchmark(benchCtx, benchmark.ID, config)

	return benchmark, nil
}

func (s *orchestratorService) simulateBenchmark(ctx context.Context, benchmarkID string, config domain.BenchmarkConfig) {
	defer func() {
		s.mu.Lock()
		delete(s.cancelFns, benchmarkID)
		s.mu.Unlock()
	}()

	time.Sleep(1 * time.Second)

	bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.repo.UpdateBenchmarkStatus(bgCtx, benchmarkID, domain.BenchmarkStatusRunning, 0, "")
	if err != nil {
		s.logger.Error("failed to set benchmark running", slog.String("error", err.Error()))
		return
	}

	// Simulate metrics updates at 1-second intervals.
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	elapsed := int64(0)
	duration := int64(config.DurationSeconds)

	var lastMetrics domain.TelemetryMetrics

	for elapsed < duration {
		select {
		case <-ctx.Done():
			// Benchmark was stopped externally.
			s.finishBenchmark(benchmarkID, elapsed, lastMetrics)
			return
		case <-ticker.C:
			elapsed++

			metrics := domain.TelemetryMetrics{
				CurrentTPS:              float64(config.OrdersPerSecond) * (0.9 + (float64(elapsed%10) / 100)),
				AvgLatencyMs:            0.5 + (float64(elapsed%5) / 10),
				TotalOrdersSent:         int32(elapsed) * config.OrdersPerSecond,
				TotalOrdersAcknowledged: int32(elapsed)*config.OrdersPerSecond - int32(elapsed/10),
				TotalErrors:             int32(elapsed / 20),
				P50LatencyMs:            0.45,
				P90LatencyMs:            0.52,
				P99LatencyMs:            0.78,
			}

			lastMetrics = metrics

			// Persist telemetry snapshot.
			snapCtx, snapCancel := context.WithTimeout(context.Background(), 2*time.Second)
			s.repo.InsertTelemetrySnapshot(snapCtx, benchmarkID, &metrics)
			snapCancel()

			// Update elapsed time.
			updCtx, updCancel := context.WithTimeout(context.Background(), 2*time.Second)
			s.repo.UpdateBenchmarkStatus(updCtx, benchmarkID, domain.BenchmarkStatusRunning, elapsed, "")
			updCancel()
		}
	}

	s.finishBenchmark(benchmarkID, duration, lastMetrics)
}

func (s *orchestratorService) finishBenchmark(benchmarkID string, elapsed int64, metrics domain.TelemetryMetrics) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.repo.CompleteBenchmark(ctx, benchmarkID, elapsed); err != nil {
		s.logger.Error("failed to complete benchmark",
			slog.String("benchmark_id", benchmarkID),
			slog.String("error", err.Error()),
		)
		return
	}

	// Compute and store composite score.
	score := s.scoring.ComputeScore(metrics)

	// Run C++ Validation Engine
	correctnessScore := s.runValidationEngine(benchmarkID)

	// Get the benchmark to find the submission_id.
	benchmark, err := s.repo.GetBenchmarkByID(ctx, benchmarkID)
	if err != nil {
		s.logger.Error("failed to get benchmark for scoring", slog.String("error", err.Error()))
		return
	}

	result := &domain.BenchmarkResult{
		ID:               uuid.New().String(),
		SubmissionID:     benchmark.SubmissionID,
		BenchmarkID:      benchmarkID,
		TPS:              metrics.CurrentTPS,
		P50LatencyMs:     metrics.P50LatencyMs,
		P90LatencyMs:     metrics.P90LatencyMs,
		P99LatencyMs:     metrics.P99LatencyMs,
		CorrectnessScore: correctnessScore,
		TotalOrders:      metrics.TotalOrdersSent,
		FailedOrders:     metrics.TotalErrors,
		CompositeScore:   score * (correctnessScore / 100), // Adjust composite score by correctness
		CreatedAt:        time.Now(),
	}

	if err := s.repo.UpsertBenchmarkResult(ctx, result); err != nil {
		s.logger.Error("failed to upsert benchmark result", slog.String("error", err.Error()))
	}

	s.logger.Info("benchmark completed",
		slog.String("benchmark_id", benchmarkID),
		slog.Float64("composite_score", score),
	)
}

func (s *orchestratorService) runValidationEngine(benchmarkID string) float64 {
	// Mock generating CSV logs for the validation engine
	workDir := filepath.Join("/tmp", "iicpc-benchmarks", benchmarkID)
	os.MkdirAll(workDir, 0755)

	tradesPath := filepath.Join(workDir, "trades.csv")
	ordersPath := filepath.Join(workDir, "orders.csv")

	// Dummy CSV files so the engine doesn't crash on file not found
	os.WriteFile(tradesPath, []byte("trade_id,order_id,price,quantity,timestamp\n1,1,100.5,10,1234567890\n"), 0644)
	os.WriteFile(ordersPath, []byte("order_id,symbol,side,price,quantity,timestamp\n1,AAPL,BUY,100.5,10,1234567800\n"), 0644)

	// Note: In production, the bot engine would produce the actual CSVs or Redpanda topics directly to this dir.
	
	enginePath, _ := filepath.Abs("../../high-performance/validation-engine-cpp/build/validation_engine")
	cmd := exec.Command(enginePath, "--trades", tradesPath, "--orders", ordersPath)
	cmd.Dir = workDir
	
	err := cmd.Run()
	if err != nil {
		s.logger.Warn("Validation engine detected correctness issues", slog.String("error", err.Error()))
		// Penalize correctness score
		return 85.0
	}

	s.logger.Info("Validation engine passed successfully")
	return 100.0
}

func (s *orchestratorService) GetBenchmarkStatus(ctx context.Context, benchmarkID string) (*domain.Benchmark, error) {
	if benchmarkID == "" {
		return nil, fmt.Errorf("%w: benchmark_id is required", domain.ErrInvalidInput)
	}

	benchmark, err := s.repo.GetBenchmarkByID(ctx, benchmarkID)
	if err != nil {
		return nil, err
	}

	return benchmark, nil
}

func (s *orchestratorService) StopBenchmark(ctx context.Context, benchmarkID string) error {
	if benchmarkID == "" {
		return fmt.Errorf("%w: benchmark_id is required", domain.ErrInvalidInput)
	}

	// Cancel the benchmark goroutine.
	s.mu.Lock()
	cancelFn, ok := s.cancelFns[benchmarkID]
	s.mu.Unlock()

	if ok {
		cancelFn()
		s.logger.Info("benchmark stop signal sent", slog.String("benchmark_id", benchmarkID))
	}

	return nil
}

func (s *orchestratorService) GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.GetLeaderboard(ctx, limit)
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
