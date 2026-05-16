package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/validation-service-go/config"
	"github.com/iicpc/validation-service-go/internal/consumer"
	"github.com/iicpc/validation-service-go/internal/domain"
	"github.com/iicpc/validation-service-go/internal/handler"
	"github.com/iicpc/validation-service-go/internal/repository"
	"github.com/iicpc/validation-service-go/internal/service"
	"github.com/iicpc/validation-service-go/internal/storage"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("Starting validation-service-go")

	cfg := config.LoadConfig()

	// 1. Connect to Database
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Tune connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 2. Initialize Redpanda Publisher (for validation.completed)
	producer, err := events.NewProducer(cfg.RedpandaBrokers, logger)
	if err != nil {
		log.Fatalf("Failed to initialize Redpanda producer: %v", err)
	}
	defer producer.Close()

	// 3. Initialize dependencies
	repo := repository.NewPostgresValidationRepository(db)
	storageClient := storage.NewLocalStorageClient() // shared volume mount

	// Customize contract limits from config
	contract := domain.DefaultContract
	contract.MaxExtractedBytes = cfg.MaxExtractedBytes
	contract.MaxFileCount = cfg.MaxFileCount

	valService := service.NewValidationService(repo, storageClient, &contract, producer)

	// 4. Initialize Redpanda Consumer (for submission.created)
	subConsumer := consumer.NewSubmissionConsumer(valService)
	eventConsumer, err := events.NewConsumerWithOptions(
		cfg.RedpandaBrokers,
		cfg.ConsumerGroupID,
		[]string{events.TopicSubmissionCreated},
		logger,
		events.ConsumerOptions{PartitionConcurrency: 8},
	)
	if err != nil {
		log.Fatalf("Failed to initialize Redpanda consumer: %v", err)
	}

	eventConsumer.RegisterHandler(events.TopicSubmissionCreated, func(ctx context.Context, topic string, key string, value []byte) error {
		return subConsumer.HandleMessage(ctx, value)
	})

	// Start consuming in the background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		slog.Info("Starting to consume events", "topic", events.TopicSubmissionCreated)
		if err := eventConsumer.Start(ctx); err != nil {
			slog.Error("Consumer exited with error", "error", err)
		}
	}()

	// 5. Initialize HTTP Handlers (for status API)
	valHandler := handler.NewValidationHandler(repo, valService)

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(middleware.SecurityHeaders())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		status := "SERVING"
		deps := make(map[string]string)

		if err := db.PingContext(ctx); err != nil {
			status = "NOT_SERVING"
			deps["database"] = "unhealthy"
		} else {
			deps["database"] = "healthy"
		}

		if err := producer.Ping(ctx); err != nil {
			status = "NOT_SERVING"
			deps["redpanda"] = "unhealthy"
		} else {
			deps["redpanda"] = "healthy"
		}

		code := http.StatusOK
		if status == "NOT_SERVING" {
			code = http.StatusServiceUnavailable
		}

		c.JSON(code, gin.H{
			"status":       status,
			"service":      "validation-service-go",
			"version":      "1.0.0",
			"dependencies": deps,
		})
	})

	// API Routes
	api := router.Group("/api/v1")
	{
		validations := api.Group("/validations")
		{
			validations.GET("", valHandler.ListResults)
			validations.GET("/contract", valHandler.GetContract)
			validations.GET("/:id", valHandler.GetResult)
			validations.GET("/:id/report", valHandler.GetReport)
			validations.POST("/:id/trigger", valHandler.TriggerValidation)
		}
	}

	// 6. Start HTTP server gracefully
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		slog.Info("HTTP server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen error: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down service...")

	// Cancel the consumer context
	cancel()

	// Shutdown the HTTP server with a timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	slog.Info("Server exiting")
}
