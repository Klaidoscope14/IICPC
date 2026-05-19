package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/correctness-engine-go/config"
	"github.com/iicpc/correctness-engine-go/internal/correctness"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/pkg/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.NewLogger("correctness-engine")

	// Setup Redpanda producer for publishing correctness scores
	producer, err := events.NewProducer(cfg.RedpandaBrokers, logger)
	if err != nil {
		logger.Warn("failed to connect to redpanda, running without event publishing", slog.String("error", err.Error()))
	} else {
		defer producer.Close()
	}

	engine := correctness.NewEngine(logger)

	// Setup Redpanda consumer to listen for traces
	consumer, err := events.NewConsumerWithOptions(
		cfg.RedpandaBrokers,
		"correctness-engine-group",
		[]string{events.TopicTraceAvailable},
		logger,
		events.ConsumerOptions{PartitionConcurrency: 4},
	)
	
	if err != nil {
		logger.Warn("failed to connect to redpanda consumer", slog.String("error", err.Error()))
	} else {
		events.RegisterJSONHandler[events.TraceAvailableEvent](consumer, events.TopicTraceAvailable, func(ctx context.Context, key string, event events.TraceAvailableEvent) error {
			logger.Info("trace available event received, starting evaluation",
				slog.String("benchmark_id", event.BenchmarkID),
				slog.String("file_path", event.FilePath),
			)

			result, err := engine.EvaluateTrace(event.BenchmarkID, event.FilePath)
			if err != nil {
				logger.Error("trace evaluation failed", slog.String("error", err.Error()))
				return nil // Don't return error to prevent infinite retry loop for unreadable files
			}

			logger.Info("trace evaluated",
				slog.String("benchmark_id", event.BenchmarkID),
				slog.Float64("score", result.Score),
				slog.Int("violations", int(result.Violations)),
			)

			if producer != nil {
				pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				evalEvent := events.CorrectnessEvaluatedEvent{
					BenchmarkID:      event.BenchmarkID,
					CorrectnessScore: result.Score,
					TotalViolations:  result.Violations,
					EvaluatedAt:      time.Now().UTC(),
				}

				if err := producer.PublishCorrectnessEvaluated(pubCtx, evalEvent); err != nil {
					logger.Warn("failed to publish correctness.evaluated event", slog.String("error", err.Error()))
				}
			}

			return nil
		})

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			if err := consumer.Start(ctx); err != nil {
				logger.Error("consumer exited with error", slog.String("error", err.Error()))
			}
		}()
		defer consumer.Close()
	}

	// Setup HTTP server for health checks
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		redpandaStatus := "healthy"
		if producer == nil {
			redpandaStatus = "unavailable"
		} else {
			pingCtx, pingCancel := context.WithTimeout(c.Request.Context(), 500*time.Millisecond)
			defer pingCancel()
			if err := producer.Ping(pingCtx); err != nil {
				redpandaStatus = "unhealthy"
			}
		}
		
		httpStatus := http.StatusOK
		if redpandaStatus == "unhealthy" {
			httpStatus = http.StatusServiceUnavailable
		}
		
		c.JSON(httpStatus, gin.H{
			"status":  "healthy",
			"service": "correctness-engine",
			"version": "v1",
			"dependencies": gin.H{
				"redpanda": redpandaStatus,
			},
			"timestamp": time.Now().UTC(),
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("correctness-engine service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("correctness-engine server error: %v", err)
	}
}
