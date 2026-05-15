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
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/pkg/logging"
	"github.com/iicpc/pkg/metrics"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/pkg/server"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewLogger("benchmark-orchestrator")

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

	producer, err := events.NewProducer(cfg.Redpanda.Brokers, logger)
	if err != nil {
		logger.Warn("Failed to initialize event producer", "error", err)
	}
	if producer != nil {
		defer producer.Close()
	}

	orchestratorService := service.NewOrchestratorService(repo, scoring, containerMgr, producer, logger)
	orchestratorHandler := handler.NewOrchestratorHandler(orchestratorService)

	// Initialize Redpanda Consumer
	consumer, err := events.NewConsumerWithOptions(
		cfg.Redpanda.Brokers,
		"orchestrator-group",
		[]string{events.TopicSubmissionCreated},
		logger,
		events.ConsumerOptions{PartitionConcurrency: 8},
	)
	if err != nil {
		logger.Warn("Failed to initialize event consumer", "error", err)
	} else {
		// Register handler
		events.RegisterJSONHandler[events.SubmissionCreatedEvent](consumer, events.TopicSubmissionCreated, func(ctx context.Context, key string, event events.SubmissionCreatedEvent) error {
			logger.Info("Received submission created event, triggering deployment", "submission_id", event.SubmissionID)

			// Trigger deployment
			ports := []string{"8080"}
			limits := domain.ResourceLimits{CPUMilli: 500, MemoryMB: 512}
			_, err := orchestratorService.DeploySubmission(ctx, event.SubmissionID, event.ContainerImage, ports, limits)
			return err
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
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger(logger))

	// Prometheus metrics.
	m := metrics.NewMetrics("benchmark-orchestrator")
	router.Use(m.Middleware())
	router.GET("/metrics", metrics.Handler())

	// Register routes.
	orchestratorHandler.RegisterRoutes(router)

	wsHandler := handler.NewWebSocketHandler(orchestratorService, logger)
	wsHandler.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "benchmark-orchestrator"})
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
