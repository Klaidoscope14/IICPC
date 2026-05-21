package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/iicpc/benchmark-orchestrator-go/internal/container"
	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
	"github.com/iicpc/benchmark-orchestrator-go/internal/repository"
	contractbenchmark "github.com/iicpc/pkg/contracts/benchmark"
	"github.com/iicpc/pkg/events"
)

// OrchestratorService defines the contract for deployment and benchmark operations.
type OrchestratorService interface {
	BuildAndDeploy(ctx context.Context, submissionID string) (*domain.Deployment, error)
	DeploySubmission(ctx context.Context, submissionID, containerImage string, ports []string, limits domain.ResourceLimits) (*domain.Deployment, error)
	StartBenchmark(ctx context.Context, submissionID, deploymentID string, config domain.BenchmarkConfig) (*domain.Benchmark, error)
	GetBenchmarkStatus(ctx context.Context, benchmarkID string) (*domain.Benchmark, error)
	StopBenchmark(ctx context.Context, benchmarkID string) error
	ProcessBenchmarkFinished(ctx context.Context, evt events.BenchmarkFinishedEvent) error
	ProcessCorrectnessEvaluated(ctx context.Context, evt events.CorrectnessEvaluatedEvent) error
	GetLeaderboard(ctx context.Context, limit int) ([]*domain.LeaderboardEntry, error)
}

// StorageClient defines how the orchestrator accesses archived submissions.
type StorageClient interface {
	DownloadArchive(ctx context.Context, storagePath string, destinationPath string) error
}

// Options controls orchestrator timeouts and container lifecycle policy.
type Options struct {
	BuildTimeout       time.Duration
	DeployTimeout      time.Duration
	HealthProbeTimeout time.Duration
	IdleContainerTTL   time.Duration
	RestartAttempts    int
	SandboxNetworkMode string
	SandboxBindHost    string
	SandboxServiceHost string
}

// DefaultOptions returns conservative defaults for local development.
func DefaultOptions() Options {
	return Options{
		BuildTimeout:       5 * time.Minute,
		DeployTimeout:      3 * time.Minute,
		HealthProbeTimeout: 30 * time.Second,
		IdleContainerTTL:   30 * time.Minute,
		RestartAttempts:    1,
		SandboxNetworkMode: "bridge",
		SandboxBindHost:    "127.0.0.1",
		SandboxServiceHost: "localhost",
	}
}

type orchestratorService struct {
	repo          repository.OrchestratorRepository
	scoring       *ScoringService
	logger        *slog.Logger
	containerMgr  container.Manager
	eventProducer *events.Producer
	storage       StorageClient
	opts          Options

	mu                     sync.Mutex
	cancelFns              map[string]context.CancelFunc
	deployments            map[string]*deploymentState
	deploymentBySubmission map[string]string
}

// NewOrchestratorService creates a new OrchestratorService with persistent storage and scoring.
func NewOrchestratorService(repo repository.OrchestratorRepository, scoring *ScoringService, containerMgr container.Manager, eventProducer *events.Producer, storage StorageClient, logger *slog.Logger, optionOverrides ...Options) OrchestratorService {
	opts := DefaultOptions()
	if len(optionOverrides) > 0 {
		opts = mergeOptions(opts, optionOverrides[0])
	}

	svc := &orchestratorService{
		repo:                   repo,
		scoring:                scoring,
		logger:                 logger,
		containerMgr:           containerMgr,
		eventProducer:          eventProducer,
		storage:                storage,
		opts:                   opts,
		cancelFns:              make(map[string]context.CancelFunc),
		deployments:            make(map[string]*deploymentState),
		deploymentBySubmission: make(map[string]string),
	}
	svc.startIdleCleanupLoop()
	return svc
}

type deploymentState struct {
	DeploymentID string
	SubmissionID string
	ContainerID  string
	ServiceURL   string
	LastUsed     time.Time
}

func (s *orchestratorService) BuildAndDeploy(ctx context.Context, submissionID string) (*domain.Deployment, error) {
	if submissionID == "" {
		return nil, fmt.Errorf("%w: submission_id is required", domain.ErrInvalidInput)
	}

	storagePath, err := s.repo.GetSubmissionStoragePath(ctx, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage path: %w", err)
	}

	tempFile, err := os.CreateTemp("", fmt.Sprintf("deploy-%s-*.zip", shortID(submissionID)))
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary archive path: %w", err)
	}
	tempZipPath := tempFile.Name()
	tempFile.Close()
	defer os.Remove(tempZipPath)

	if err := s.storage.DownloadArchive(ctx, storagePath, tempZipPath); err != nil {
		return nil, fmt.Errorf("failed to download archive: %w", err)
	}

	tarBuf, err := container.ConvertZipToTar(tempZipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert zip to tar: %w", err)
	}

	imageName := fmt.Sprintf("submission-%s:%d", shortID(submissionID), time.Now().UnixNano())

	// For now we assume a standard set of ports and limits as defined by the platform.
	// In the real system, these would be read from the contract or validation event.
	ports := []string{"8080"}
	limits := domain.ResourceLimits{CPUMilli: 1000, MemoryMB: 512, TimeoutSeconds: int64(s.opts.DeployTimeout.Seconds())}

	if s.containerMgr == nil {
		s.logger.Warn("docker manager unavailable, skipping image build and creating simulated deployment",
			slog.String("submission_id", submissionID),
		)
		s.recordSubmissionLog(context.Background(), submissionID, "build", "warn", "Docker manager unavailable; deployment will be simulated.", map[string]string{
			"image": imageName,
		})
		return s.DeploySubmission(ctx, submissionID, imageName, ports, limits)
	}

	s.logger.Info("building docker image", slog.String("submission_id", submissionID), slog.String("image", imageName))
	buildCtx, buildCancel := context.WithTimeout(ctx, s.opts.BuildTimeout)
	defer buildCancel()

	buildResult, err := s.containerMgr.BuildImage(buildCtx, tarBuf, imageName)
	if err != nil {
		s.recordBuildLog(submissionID, "error", buildResult, err)
		return nil, fmt.Errorf("failed to build image: %w", err)
	}
	s.recordBuildLog(submissionID, "info", buildResult, nil)

	return s.DeploySubmission(ctx, submissionID, imageName, ports, limits)
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
		ServiceURL:     fmt.Sprintf("pending://submission-%s", shortID(submissionID)),
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
	go s.executeDeployment(deployment.ID, submissionID, containerImage, ports, limits)

	return deployment, nil
}

func (s *orchestratorService) executeDeployment(deploymentID, submissionID, containerImage string, ports []string, limits domain.ResourceLimits) {
	ctx, cancel := context.WithTimeout(context.Background(), s.opts.DeployTimeout)
	defer cancel()

	s.logger.Info("starting deployment", slog.String("deployment_id", deploymentID))

	if s.containerMgr == nil {
		serviceURL := fmt.Sprintf("http://submission-%s:8080", shortID(submissionID))
		containerID := fmt.Sprintf("simulated-%s", shortID(deploymentID))
		s.logger.Warn("docker manager unavailable, marking deployment as simulated",
			slog.String("deployment_id", deploymentID),
		)
		s.markDeploymentReady(ctx, deploymentID, submissionID, serviceURL, containerID)
		return
	}

	attempts := s.opts.RestartAttempts + 1
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		opts := container.CreateOptions{
			ImageName:      containerImage,
			ContainerName:  fmt.Sprintf("submission-%s-%d", shortID(deploymentID), attempt),
			ExposedPorts:   ports,
			CPUMilli:       limits.CPUMilli,
			MemoryMB:       limits.MemoryMB,
			PidsLimit:      100,
			TimeoutSeconds: int64(s.opts.DeployTimeout.Seconds()),
			NetworkMode:    s.opts.SandboxNetworkMode,
			HostBindIP:     s.opts.SandboxBindHost,
			ServiceHost:    s.opts.SandboxServiceHost,
			Cmd:            []string{}, // Allow the image's ENTRYPOINT to run.
			RunAsUser:      "65532:65532",
		}

		containerID, serviceURL, err := s.containerMgr.CreateAndStart(ctx, opts)
		if err != nil {
			lastErr = err
			s.logger.Warn("failed to create container",
				slog.String("deployment_id", deploymentID),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()),
			)
			continue
		}

		healthURL := joinURL(serviceURL, "health")
		if err := s.containerMgr.WaitForHealthy(ctx, containerID, healthURL, s.opts.HealthProbeTimeout); err != nil {
			lastErr = err
			s.logger.Warn("container failed readiness check",
				slog.String("deployment_id", deploymentID),
				slog.String("container_id", containerID),
				slog.Int("attempt", attempt),
				slog.String("error", err.Error()),
			)
			s.cleanupContainer(containerID, "failed readiness check")
			continue
		}

		s.markDeploymentReady(ctx, deploymentID, submissionID, serviceURL, containerID)
		
		// Capture container logs asynchronously.
		if err := s.containerMgr.CaptureLogs(context.Background(), containerID, s.logger); err != nil {
			s.logger.Warn("failed to capture container logs", slog.String("error", err.Error()))
		}
		
		s.recordSubmissionLog(context.Background(), submissionID, "runtime", "info", "Container started and passed readiness check.", map[string]string{
			"deployment_id": deploymentID,
			"container_id":  containerID,
			"service_url":   serviceURL,
		})
		return
	}

	errMsg := "deployment failed"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	s.logger.Error("deployment failed after restart attempts",
		slog.String("deployment_id", deploymentID),
		slog.Int("attempts", attempts),
		slog.String("error", errMsg),
	)
	if err := s.repo.UpdateDeploymentStatus(ctx, deploymentID, domain.DeploymentStatusFailed, "", "", errMsg); err != nil {
		s.logger.Error("failed to mark deployment failed", slog.String("deployment_id", deploymentID), slog.String("error", err.Error()))
	}
	s.recordSubmissionLog(context.Background(), submissionID, "runtime", "error", "Container failed to start or pass readiness checks.", map[string]string{
		"deployment_id": deploymentID,
		"error":         errMsg,
	})
}

func (s *orchestratorService) markDeploymentReady(ctx context.Context, deploymentID, submissionID, serviceURL, containerID string) {
	err := s.repo.UpdateDeploymentStatus(ctx, deploymentID, domain.DeploymentStatusDeployed, serviceURL, containerID, "")
	if err != nil {
		s.logger.Error("failed to update deployment status",
			slog.String("deployment_id", deploymentID),
			slog.String("error", err.Error()),
		)
		return
	}
	s.trackDeployment(deploymentID, submissionID, serviceURL, containerID)

	if s.eventProducer != nil {
		eventCtx, eventCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer eventCancel()

		err := s.eventProducer.PublishEngineReady(eventCtx, events.EngineReadyEvent{
			DeploymentID: deploymentID,
			SubmissionID: submissionID,
			ServiceURL:   serviceURL,
			ContainerID:  containerID,
			ReadyAt:      time.Now().UTC(),
		})
		if err != nil {
			s.logger.Warn("failed to publish engine ready event",
				slog.String("deployment_id", deploymentID),
				slog.String("error", err.Error()),
			)
		}
	}
}

func (s *orchestratorService) StartBenchmark(ctx context.Context, submissionID, deploymentID string, config domain.BenchmarkConfig) (*domain.Benchmark, error) {
	if submissionID == "" {
		return nil, fmt.Errorf("%w: submission_id is required", domain.ErrInvalidInput)
	}
	if deploymentID == "" {
		return nil, fmt.Errorf("%w: deployment_id is required", domain.ErrInvalidInput)
	}
	if err := s.ensureDeploymentReady(ctx, submissionID, deploymentID); err != nil {
		return nil, err
	}
	s.touchDeployment(deploymentID)

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

	if s.eventProducer != nil {
		eventCtx, eventCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer eventCancel()

		s.mu.Lock()
		serviceURL := ""
		if state, ok := s.deployments[deploymentID]; ok {
			serviceURL = state.ServiceURL
		}
		s.mu.Unlock()

		if serviceURL == "" {
			deployment, _ := s.repo.GetDeploymentByID(ctx, deploymentID)
			if deployment != nil {
				serviceURL = deployment.ServiceURL
			}
		}

		if err := s.eventProducer.PublishBenchmarkStarted(eventCtx, events.BenchmarkStartedEvent{
			BenchmarkID:  benchmark.ID,
			SubmissionID: submissionID,
			DeploymentID: deploymentID,
			ServiceURL:   serviceURL,
			Config:       toContractBenchmarkConfig(config),
			StartedAt:    benchmark.StartedAt,
		}); err != nil {
			s.logger.Warn("failed to publish benchmark started event",
				slog.String("benchmark_id", benchmark.ID),
				slog.String("error", err.Error()),
			)
		}
	}

	go s.simulateBenchmark(benchCtx, benchmark.ID, config)

	return benchmark, nil
}

func (s *orchestratorService) simulateBenchmark(ctx context.Context, benchmarkID string, config domain.BenchmarkConfig) {
	defer func() {
		s.mu.Lock()
		delete(s.cancelFns, benchmarkID)
		s.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		s.stopBenchmarkRun(benchmarkID, 0)
		return
	case <-time.After(1 * time.Second):
	}

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
			s.stopBenchmarkRun(benchmarkID, elapsed)
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

			s.publishTelemetrySnapshot(benchmarkID, metrics)

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

	// Compute base score (before correctness multiplier).
	baseScore := s.scoring.ComputeScore(metrics)

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
		CorrectnessScore: 0, // Awaits real evaluation from Correctness Engine via Redpanda
		TotalOrders:      metrics.TotalOrdersSent,
		FailedOrders:     metrics.TotalErrors,
		CompositeScore:   0, // Initial composite is 0 until correctness score is published
		CreatedAt:        time.Now(),
	}

	if err := s.repo.UpsertBenchmarkResult(ctx, result); err != nil {
		s.logger.Error("failed to upsert benchmark result", slog.String("error", err.Error()))
	}

	s.publishBenchmarkFinished(ctx, benchmark, result, elapsed)
	s.publishLeaderboardUpdated(ctx, benchmarkID)

	s.logger.Info("benchmark completed (awaiting correctness evaluation)",
		slog.String("benchmark_id", benchmarkID),
		slog.Float64("base_score", baseScore),
	)
	s.touchDeployment(benchmark.DeploymentID)
}

func (s *orchestratorService) ProcessBenchmarkFinished(ctx context.Context, evt events.BenchmarkFinishedEvent) error {
	// Look up the active deployment ID for this submission.
	s.mu.Lock()
	deploymentID := s.deploymentBySubmission[evt.SubmissionID]
	s.mu.Unlock()

	if deploymentID == "" {
		s.logger.Warn("could not find active deployment in cache for submission, falling back to db query", slog.String("submission_id", evt.SubmissionID))
		deployment, err := s.repo.GetLatestDeploymentBySubmission(ctx, evt.SubmissionID)
		if err != nil {
			s.logger.Error("failed to find deployment in DB, using submission_id as unsafe fallback", slog.String("submission_id", evt.SubmissionID), slog.String("error", err.Error()))
			deploymentID = evt.SubmissionID
		} else {
			deploymentID = deployment.ID
		}
	}

	// Because bot-fleet bypasses /benchmarks/start and auto-generates its own benchmark_id,
	// we must create the parent `benchmarks` table row first to satisfy Foreign Key constraints!
	b := &domain.Benchmark{
		ID:           evt.BenchmarkID,
		SubmissionID: evt.SubmissionID,
		DeploymentID: deploymentID,
		Status:       domain.BenchmarkStatusCompleted,
		Config:       domain.BenchmarkConfig{},
		StartedAt:    time.Now().Add(-time.Duration(evt.ElapsedSeconds) * time.Second),
		CompletedAt:  &evt.FinishedAt,
		ElapsedTime:  evt.ElapsedSeconds,
	}
	if err := s.repo.CreateBenchmark(ctx, b); err != nil {
		s.logger.Warn("failed to create parent benchmark row (might already exist)", slog.String("error", err.Error()))
	}

	result := &domain.BenchmarkResult{
		ID:               uuid.New().String(),
		SubmissionID:     evt.SubmissionID,
		BenchmarkID:      evt.BenchmarkID,
		TPS:              evt.TPS,
		P50LatencyMs:     evt.P99LatencyMs * 0.5, // approximate since bot-fleet doesn't send p50 in this event yet
		P90LatencyMs:     evt.P99LatencyMs * 0.8,
		P99LatencyMs:     evt.P99LatencyMs,
		CorrectnessScore: 0,
		TotalOrders:      int32(evt.TPS * float64(evt.ElapsedSeconds)),
		FailedOrders:     0,
		CompositeScore:   0,
		CreatedAt:        time.Now(),
	}
	
	if err := s.repo.UpsertBenchmarkResult(ctx, result); err != nil {
		s.logger.Error("failed to upsert benchmark result", slog.String("error", err.Error()))
		return err
	}
	
	s.logger.Info("created benchmark result from external bot-fleet event", slog.String("benchmark_id", evt.BenchmarkID))
	s.publishLeaderboardUpdated(ctx, evt.BenchmarkID)
	return nil
}

func (s *orchestratorService) ProcessCorrectnessEvaluated(ctx context.Context, evt events.CorrectnessEvaluatedEvent) error {
	s.logger.Info("processing correctness evaluated event",
		slog.String("benchmark_id", evt.BenchmarkID),
		slog.Float64("correctness_score", evt.CorrectnessScore),
		slog.Int("violations", int(evt.TotalViolations)),
	)

	// Fetch existing result (with retry to handle async race condition where correctness finishes before orchestrator's 15s loop)
	var result *domain.BenchmarkResult
	var err error
	for i := 0; i < 5; i++ {
		result, err = s.repo.GetBenchmarkResult(ctx, evt.BenchmarkID)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to get benchmark result after retries: %w", err)
	}

	// Re-calculate the base score based on the original result metrics
	metrics := domain.TelemetryMetrics{
		CurrentTPS:              result.TPS,
		P50LatencyMs:            result.P50LatencyMs,
		P90LatencyMs:            result.P90LatencyMs,
		P99LatencyMs:            result.P99LatencyMs,
		TotalOrdersSent:         result.TotalOrders,
		TotalErrors:             result.FailedOrders,
		TotalOrdersAcknowledged: result.TotalOrders - result.FailedOrders, // Approximation for score re-calc
	}
	baseScore := s.scoring.ComputeScore(metrics)

	// New composite score incorporates real correctness score
	newCompositeScore := baseScore * (evt.CorrectnessScore / 100.0)

	// Update the database
	err = s.repo.UpdateCorrectnessScore(ctx, evt.BenchmarkID, evt.CorrectnessScore, newCompositeScore)
	if err != nil {
		return fmt.Errorf("failed to update correctness score: %w", err)
	}

	s.logger.Info("updated benchmark result with correctness score",
		slog.String("benchmark_id", evt.BenchmarkID),
		slog.Float64("new_composite_score", newCompositeScore),
	)

	// Republish leaderboard so UI updates
	s.publishLeaderboardUpdated(ctx, evt.BenchmarkID)

	return nil
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

func (s *orchestratorService) ensureDeploymentReady(ctx context.Context, submissionID, deploymentID string) error {
	deployment, err := s.repo.GetDeploymentByID(ctx, deploymentID)
	if err != nil {
		return err
	}
	if deployment.SubmissionID != submissionID {
		return fmt.Errorf("%w: deployment does not belong to submission", domain.ErrInvalidInput)
	}
	if deployment.Status != domain.DeploymentStatusDeployed {
		return fmt.Errorf("%w: deployment is %s", domain.ErrInvalidInput, deployment.Status)
	}
	if s.containerMgr == nil || deployment.ContainerID == "" || strings.HasPrefix(deployment.ContainerID, "simulated") {
		return nil
	}

	running, err := s.containerMgr.IsRunning(ctx, deployment.ContainerID)
	if err != nil {
		return fmt.Errorf("%w: failed to inspect deployment container: %v", domain.ErrInternal, err)
	}
	if !running {
		return fmt.Errorf("%w: deployment container is not running", domain.ErrInvalidInput)
	}

	probeTimeout := minDuration(s.opts.HealthProbeTimeout, 3*time.Second)
	if probeTimeout <= 0 {
		probeTimeout = 3 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	if err := s.containerMgr.WaitForHealthy(probeCtx, deployment.ContainerID, joinURL(deployment.ServiceURL, "health"), probeTimeout); err != nil {
		return fmt.Errorf("%w: deployment readiness check failed: %v", domain.ErrInvalidInput, err)
	}
	return nil
}

func (s *orchestratorService) recordBuildLog(submissionID, level string, result container.BuildResult, buildErr error) {
	message := strings.TrimSpace(result.Logs)
	if message == "" {
		if buildErr != nil {
			message = buildErr.Error()
		} else {
			message = "Docker image build completed."
		}
	}

	metadata := map[string]string{
		"truncated": fmt.Sprintf("%t", result.Truncated),
	}
	if buildErr != nil {
		metadata["error"] = buildErr.Error()
	}
	s.recordSubmissionLog(context.Background(), submissionID, "build", level, message, metadata)
}

func (s *orchestratorService) recordSubmissionLog(parent context.Context, submissionID, logType, level, message string, metadata map[string]string) {
	if submissionID == "" || s.repo == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	if metadata == nil {
		metadata = map[string]string{}
	}

	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()

	log := &domain.SubmissionLog{
		ID:           uuid.New().String(),
		SubmissionID: submissionID,
		LogType:      logType,
		Message:      truncateMessage(message, 256*1024),
		Level:        level,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.repo.CreateSubmissionLog(ctx, log); err != nil {
		s.logger.Warn("failed to persist submission log",
			slog.String("submission_id", submissionID),
			slog.String("log_type", logType),
			slog.String("error", err.Error()),
		)
	}
}

func (s *orchestratorService) trackDeployment(deploymentID, submissionID, serviceURL, containerID string) {
	if containerID == "" || strings.HasPrefix(containerID, "simulated") {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.deployments[deploymentID] = &deploymentState{
		DeploymentID: deploymentID,
		SubmissionID: submissionID,
		ContainerID:  containerID,
		ServiceURL:   serviceURL,
		LastUsed:     time.Now(),
	}
	s.deploymentBySubmission[submissionID] = deploymentID
}

func (s *orchestratorService) touchDeployment(deploymentID string) {
	if deploymentID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if state, ok := s.deployments[deploymentID]; ok {
		state.LastUsed = time.Now()
	}
}

func (s *orchestratorService) startIdleCleanupLoop() {
	if s.containerMgr == nil || s.opts.IdleContainerTTL <= 0 {
		return
	}

	interval := s.opts.IdleContainerTTL / 2
	if interval < 10*time.Second {
		interval = 10 * time.Second
	}
	if interval > time.Minute {
		interval = time.Minute
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.cleanupIdleDeployments(time.Now())
		}
	}()
}

func (s *orchestratorService) cleanupIdleDeployments(now time.Time) {
	var expired []deploymentState

	s.mu.Lock()
	for deploymentID, state := range s.deployments {
		if now.Sub(state.LastUsed) < s.opts.IdleContainerTTL {
			continue
		}
		expired = append(expired, *state)
		delete(s.deployments, deploymentID)
		if s.deploymentBySubmission[state.SubmissionID] == deploymentID {
			delete(s.deploymentBySubmission, state.SubmissionID)
		}
	}
	s.mu.Unlock()

	for _, state := range expired {
		s.logger.Info("cleaning up idle deployment",
			slog.String("deployment_id", state.DeploymentID),
			slog.String("container_id", state.ContainerID),
		)
		s.cleanupContainer(state.ContainerID, "idle timeout")

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		err := s.repo.UpdateDeploymentStatus(ctx, state.DeploymentID, domain.DeploymentStatusTerminated, state.ServiceURL, state.ContainerID, "idle timeout")
		cancel()
		if err != nil {
			s.logger.Warn("failed to mark idle deployment terminated",
				slog.String("deployment_id", state.DeploymentID),
				slog.String("error", err.Error()),
			)
		}
		s.recordSubmissionLog(context.Background(), state.SubmissionID, "runtime", "info", "Idle container cleaned up.", map[string]string{
			"deployment_id": state.DeploymentID,
			"container_id":  state.ContainerID,
		})
	}
}

func (s *orchestratorService) cleanupContainer(containerID, reason string) {
	if s.containerMgr == nil || containerID == "" || strings.HasPrefix(containerID, "simulated") {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.containerMgr.Stop(ctx, containerID); err != nil {
		s.logger.Debug("container stop during cleanup failed",
			slog.String("container_id", containerID),
			slog.String("reason", reason),
			slog.String("error", err.Error()),
		)
	}
	if err := s.containerMgr.Remove(ctx, containerID); err != nil {
		s.logger.Warn("container removal during cleanup failed",
			slog.String("container_id", containerID),
			slog.String("reason", reason),
			slog.String("error", err.Error()),
		)
	}
}

func (s *orchestratorService) stopBenchmarkRun(benchmarkID string, elapsed int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.repo.UpdateBenchmarkStatus(ctx, benchmarkID, domain.BenchmarkStatusStopped, elapsed, "stopped by request"); err != nil {
		s.logger.Error("failed to mark benchmark stopped",
			slog.String("benchmark_id", benchmarkID),
			slog.String("error", err.Error()),
		)
		return
	}
	if benchmark, err := s.repo.GetBenchmarkByID(ctx, benchmarkID); err == nil {
		s.touchDeployment(benchmark.DeploymentID)
	}
}

func mergeOptions(base, override Options) Options {
	if override.BuildTimeout > 0 {
		base.BuildTimeout = override.BuildTimeout
	}
	if override.DeployTimeout > 0 {
		base.DeployTimeout = override.DeployTimeout
	}
	if override.HealthProbeTimeout > 0 {
		base.HealthProbeTimeout = override.HealthProbeTimeout
	}
	if override.IdleContainerTTL >= 0 {
		base.IdleContainerTTL = override.IdleContainerTTL
	}
	if override.RestartAttempts >= 0 {
		base.RestartAttempts = override.RestartAttempts
	}
	if override.SandboxNetworkMode != "" {
		base.SandboxNetworkMode = override.SandboxNetworkMode
	}
	if override.SandboxBindHost != "" {
		base.SandboxBindHost = override.SandboxBindHost
	}
	if override.SandboxServiceHost != "" {
		base.SandboxServiceHost = override.SandboxServiceHost
	}
	return base
}

func joinURL(base, path string) string {
	if base == "" {
		return ""
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

func truncateMessage(message string, maxBytes int) string {
	if maxBytes <= 0 || len(message) <= maxBytes {
		return message
	}
	return message[:maxBytes] + "\n[truncated]"
}

func shortID(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func minDuration(a, b time.Duration) time.Duration {
	if a <= 0 {
		return b
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (s *orchestratorService) publishTelemetrySnapshot(benchmarkID string, metrics domain.TelemetryMetrics) {
	if s.eventProducer == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := s.eventProducer.PublishAsync(ctx, events.TopicTelemetrySnapshot, benchmarkID, events.TelemetrySnapshotEvent{
		BenchmarkID: benchmarkID,
		Timestamp:   time.Now().UTC(),
		Metrics:     toContractTelemetry(metrics),
	}, nil)
	if err != nil {
		s.logger.Warn("failed to queue telemetry event",
			slog.String("benchmark_id", benchmarkID),
			slog.String("error", err.Error()),
		)
	}
}

func (s *orchestratorService) publishBenchmarkFinished(ctx context.Context, benchmark *domain.Benchmark, result *domain.BenchmarkResult, elapsed int64) {
	if s.eventProducer == nil {
		return
	}

	eventCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err := s.eventProducer.PublishBenchmarkFinished(eventCtx, events.BenchmarkFinishedEvent{
		BenchmarkID:      benchmark.ID,
		SubmissionID:     benchmark.SubmissionID,
		CompositeScore:   result.CompositeScore,
		TPS:              result.TPS,
		P99LatencyMs:     result.P99LatencyMs,
		CorrectnessScore: result.CorrectnessScore,
		ElapsedSeconds:   elapsed,
		FinishedAt:       time.Now().UTC(),
	})
	if err != nil {
		s.logger.Warn("failed to publish benchmark finished event",
			slog.String("benchmark_id", benchmark.ID),
			slog.String("error", err.Error()),
		)
	}
}

func (s *orchestratorService) publishLeaderboardUpdated(ctx context.Context, benchmarkID string) {
	if s.eventProducer == nil {
		return
	}

	entries, err := s.repo.GetLeaderboard(ctx, 50)
	if err != nil {
		s.logger.Warn("failed to load leaderboard for event",
			slog.String("benchmark_id", benchmarkID),
			slog.String("error", err.Error()),
		)
		return
	}

	eventCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	err = s.eventProducer.PublishLeaderboardUpdated(eventCtx, events.LeaderboardUpdatedEvent{
		BenchmarkID: benchmarkID,
		UpdatedAt:   time.Now().UTC(),
		Entries:     toContractLeaderboard(entries),
	})
	if err != nil {
		s.logger.Warn("failed to publish leaderboard updated event",
			slog.String("benchmark_id", benchmarkID),
			slog.String("error", err.Error()),
		)
	}
}

func toContractBenchmarkConfig(config domain.BenchmarkConfig) contractbenchmark.Config {
	return contractbenchmark.Config{
		BotCount:        config.BotCount,
		DurationSeconds: config.DurationSeconds,
		OrdersPerSecond: config.OrdersPerSecond,
		Protocols:       config.Protocols,
	}
}

func toContractTelemetry(metrics domain.TelemetryMetrics) contractbenchmark.TelemetryMetrics {
	return contractbenchmark.TelemetryMetrics{
		CurrentTPS:              metrics.CurrentTPS,
		AvgLatencyMs:            metrics.AvgLatencyMs,
		TotalOrdersSent:         metrics.TotalOrdersSent,
		TotalOrdersAcknowledged: metrics.TotalOrdersAcknowledged,
		TotalErrors:             metrics.TotalErrors,
		P50LatencyMs:            metrics.P50LatencyMs,
		P90LatencyMs:            metrics.P90LatencyMs,
		P99LatencyMs:            metrics.P99LatencyMs,
		ActiveConnections:       metrics.ActiveConnections,
		CPUUsagePercent:         metrics.CPUUsagePercent,
		MemoryUsageMB:           metrics.MemoryUsageMB,
	}
}

func toContractLeaderboard(entries []*domain.LeaderboardEntry) []contractbenchmark.LeaderboardEntry {
	out := make([]contractbenchmark.LeaderboardEntry, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		out = append(out, contractbenchmark.LeaderboardEntry{
			Rank:             entry.Rank,
			SubmissionID:     entry.SubmissionID,
			BenchmarkID:      entry.BenchmarkID,
			TeamName:         entry.TeamName,
			TPS:              entry.TPS,
			P50LatencyMs:     entry.P50LatencyMs,
			P90LatencyMs:     entry.P90LatencyMs,
			P99LatencyMs:     entry.P99LatencyMs,
			CorrectnessScore: entry.CorrectnessScore,
			TotalOrders:      entry.TotalOrders,
			FailedOrders:     entry.FailedOrders,
			CompositeScore:   entry.CompositeScore,
		})
	}
	return out
}
