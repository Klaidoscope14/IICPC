package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/benchmark-orchestrator-go/config"
	"github.com/iicpc/benchmark-orchestrator-go/internal/container"
	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
	"github.com/iicpc/benchmark-orchestrator-go/internal/handler"
	"github.com/iicpc/benchmark-orchestrator-go/internal/repository"
	"github.com/iicpc/benchmark-orchestrator-go/internal/service"
	"github.com/iicpc/benchmark-orchestrator-go/internal/storage"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/pkg/logging"
	"github.com/iicpc/pkg/metrics"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/pkg/server"
	"github.com/iicpc/pkg/telemetry"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewLogger("benchmark-orchestrator")

	// Initialize OpenTelemetry Tracer
	tp, err := telemetry.InitTracerProvider("benchmark-orchestrator")
	if err != nil {
		logger.Warn("Failed to initialize telemetry", "error", err)
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				logger.Error("Error shutting down tracer provider", "error", err)
			}
		}()
	}

	// Connect to PostgreSQL.
	db, err := sqlx.Connect("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize Docker Manager
	containerMgr, err := container.NewDockerManager(logger)
	if err != nil {
		logger.Warn("Failed to initialize docker manager, using simulated deployments", "error", err)
	}

	// Wire up dependencies.
	repo := repository.NewPostgresRepository(db)
	scoring := service.NewScoringService(domain.DefaultScoringWeights())
	storageClient := storage.NewLocalStorageClient()

	producer, err := events.NewProducer(cfg.Redpanda.Brokers, logger)
	if err != nil {
		logger.Warn("Failed to initialize event producer", "error", err)
	}
	if producer != nil {
		defer producer.Close()
	}

	orchestratorOptions := service.Options{
		BuildTimeout:       time.Duration(cfg.Sandbox.BuildTimeoutSeconds) * time.Second,
		DeployTimeout:      time.Duration(cfg.Sandbox.DeployTimeoutSeconds) * time.Second,
		HealthProbeTimeout: time.Duration(cfg.Sandbox.HealthProbeTimeoutSeconds) * time.Second,
		IdleContainerTTL:   time.Duration(cfg.Sandbox.IdleContainerTTLSeconds) * time.Second,
		RestartAttempts:    cfg.Sandbox.RestartAttempts,
		SandboxNetworkMode: cfg.Sandbox.NetworkMode,
		SandboxBindHost:    cfg.Sandbox.BindHost,
		SandboxServiceHost: cfg.Sandbox.ServiceHost,
	}
	orchestratorService := service.NewOrchestratorService(repo, scoring, containerMgr, producer, storageClient, logger, orchestratorOptions)
	orchestratorHandler := handler.NewOrchestratorHandler(orchestratorService)

	// Initialize Redpanda Consumer
	consumer, err := events.NewConsumerWithOptions(
		cfg.Redpanda.Brokers,
		"orchestrator-group",
		[]string{events.TopicValidationCompleted, events.TopicCorrectnessEvaluated, events.TopicBenchmarkFinished, events.TopicEngineReady, events.TopicTelemetrySnapshot},
		logger,
		events.ConsumerOptions{PartitionConcurrency: 8},
	)
	if err != nil {
		logger.Warn("Failed to initialize event consumer", "error", err)
	} else {
		// Register handler
		events.RegisterJSONHandler[events.ValidationCompletedEvent](consumer, events.TopicValidationCompleted, func(ctx context.Context, key string, event events.ValidationCompletedEvent) error {
			if event.Status != "passed" {
				logger.Info("Skipping deployment, validation did not pass", "submission_id", event.SubmissionID, "status", event.Status)
				return nil
			}

			logger.Info("Received validation completed event, triggering build and deploy", "submission_id", event.SubmissionID)
			_, err := orchestratorService.BuildAndDeploy(ctx, event.SubmissionID)
			return err
		})

		// Register correctness handler
		events.RegisterJSONHandler[events.CorrectnessEvaluatedEvent](consumer, events.TopicCorrectnessEvaluated, func(ctx context.Context, key string, event events.CorrectnessEvaluatedEvent) error {
			logger.Info("Received correctness evaluated event, updating score", "benchmark_id", event.BenchmarkID)
			return orchestratorService.ProcessCorrectnessEvaluated(ctx, event)
		})

		// Register benchmark finished handler
		events.RegisterJSONHandler[events.BenchmarkFinishedEvent](consumer, events.TopicBenchmarkFinished, func(ctx context.Context, key string, event events.BenchmarkFinishedEvent) error {
			logger.Info("Received benchmark finished event, creating result row", "benchmark_id", event.BenchmarkID)
			return orchestratorService.ProcessBenchmarkFinished(ctx, event)
		})

		// Register engine ready handler
		events.RegisterJSONHandler[events.EngineReadyEvent](consumer, events.TopicEngineReady, func(ctx context.Context, key string, event events.EngineReadyEvent) error {
			logger.Info("Received engine ready event, starting benchmark", "deployment_id", event.DeploymentID)
			config := domain.BenchmarkConfig{
				BotCount:        10,
				DurationSeconds: 60,
				OrdersPerSecond: 100,
			}
			_, err := orchestratorService.StartBenchmark(ctx, event.SubmissionID, event.DeploymentID, config)
			return err
		})

		// Register telemetry snapshot handler
		events.RegisterJSONHandler[events.TelemetrySnapshotEvent](consumer, events.TopicTelemetrySnapshot, func(ctx context.Context, key string, event events.TelemetrySnapshotEvent) error {
			return orchestratorService.ProcessTelemetrySnapshot(ctx, event)
		})

		// Start consumer in background
		go func() {
			if err := consumer.Start(context.Background()); err != nil {
				logger.Error("Event consumer error", "error", err)
			}
		}()
		defer consumer.Close()
	}

	// Set up router with middleware.
	router := gin.Default()
	router.Use(telemetry.Middleware("benchmark-orchestrator"))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger(logger))

	// Prometheus metrics.
	m := metrics.NewMetrics("benchmark-orchestrator")
	router.Use(m.Middleware())
	router.GET("/metrics", metrics.Handler())

	// Register routes.
	orchestratorHandler.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 750*time.Millisecond)
		defer cancel()

		statusCode := http.StatusOK
		dependencies := gin.H{
			"database": "healthy",
			"redpanda": "disabled",
			"docker":   "disabled",
		}

		if err := db.PingContext(ctx); err != nil {
			statusCode = http.StatusServiceUnavailable
			dependencies["database"] = "unhealthy"
		}
		if producer != nil {
			pingCtx, pingCancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
			if err := producer.Ping(pingCtx); err != nil {
				statusCode = http.StatusServiceUnavailable
				dependencies["redpanda"] = "unhealthy"
			} else {
				dependencies["redpanda"] = "healthy"
			}
			pingCancel()
		}
		if containerMgr != nil {
			dependencies["docker"] = "configured"
		}

		c.JSON(statusCode, gin.H{
			"status":       mapStatus(statusCode),
			"service":      "benchmark-orchestrator",
			"dependencies": dependencies,
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Block until shutdown signal.
	server.RunGracefully(srv, "benchmark-orchestrator")
}

func mapStatus(statusCode int) string {
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return "healthy"
	}
	return "unhealthy"
}
