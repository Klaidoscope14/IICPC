package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/pkg/logging"
	"github.com/iicpc/pkg/metrics"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/pkg/server"
	"github.com/iicpc/submission-service-go/config"
	"github.com/iicpc/submission-service-go/internal/handler"
	"github.com/iicpc/submission-service-go/internal/publisher"
	"github.com/iicpc/submission-service-go/internal/repository"
	"github.com/iicpc/submission-service-go/internal/service"
	"github.com/iicpc/submission-service-go/internal/storage"
	"github.com/iicpc/submission-service-go/internal/validation"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger := logging.NewLogger("submission-service")

	db, err := sqlx.Connect("postgres", cfg.Database.DSN())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Tune connection pool for concurrency.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Wire up Redpanda producer (durable event pipeline).
	producer, err := events.NewProducer(cfg.Redpanda.Brokers, logger)
	if err != nil {
		logger.Warn("failed to connect to redpanda, running without event producer", "error", err)
	} else {
		defer producer.Close()
	}

	// Wire up Redis publisher (real-time notifications).
	var redisPublisher *publisher.RedisPublisher
	redisPub, err := publisher.NewRedisPublisher(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger)
	if err != nil {
		logger.Warn("failed to connect to redis, running without redis publisher", "error", err)
	} else {
		redisPublisher = redisPub
		defer redisPublisher.Close()
	}

	// Initialize storage.
	localStorage, err := storage.NewLocalStorage(cfg.Storage.BasePath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize upload validator.
	validator := validation.NewUploadValidator(cfg.Storage.MaxUploadBytes, nil)

	// Wire up dependencies.
	submissionRepo := repository.NewPostgresSubmissionRepository(db)
	submissionService := service.NewSubmissionService(submissionRepo, producer, redisPublisher, localStorage, validator, logger)
	submissionHandler := handler.NewSubmissionHandler(submissionService, cfg.Storage.MaxUploadBytes)

	// Set up router with middleware.
	router := gin.Default()
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger(logger))

	// Prometheus metrics.
	m := metrics.NewMetrics("submission-service")
	router.Use(m.Middleware())
	router.GET("/metrics", metrics.Handler())

	// Register routes.
	submissionHandler.RegisterRoutes(router)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "submission-service"})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Block until shutdown signal.
	server.RunGracefully(srv, "submission-service")
}

