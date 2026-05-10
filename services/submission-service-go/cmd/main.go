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
	"github.com/iicpc/submission-service-go/internal/repository"
	"github.com/iicpc/submission-service-go/internal/service"
	"github.com/iicpc/submission-service-go/internal/storage"
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

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Wire up events and storage
	producer, err := events.NewProducer([]string{"localhost:19092"}, logger)
	if err != nil {
		logger.Warn("failed to connect to redpanda, running without event producer", "error", err)
	} else {
		defer producer.Close()
	}

	localStorage, err := storage.NewLocalStorage("/tmp/iicpc-storage")
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Wire up dependencies.
	submissionRepo := repository.NewPostgresSubmissionRepository(db)
	submissionService := service.NewSubmissionService(submissionRepo, producer, localStorage)
	submissionHandler := handler.NewSubmissionHandler(submissionService)

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
