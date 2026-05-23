package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iicpc/bot-fleet-go/config"
	"github.com/iicpc/bot-fleet-go/internal/bot"
	"github.com/iicpc/bot-fleet-go/internal/fleet"
	"github.com/iicpc/bot-fleet-go/internal/telemetry"
	contractbenchmark "github.com/iicpc/pkg/contracts/benchmark"
	"github.com/iicpc/pkg/events"
	"github.com/iicpc/pkg/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.NewLogger("bot-fleet")

	// Redpanda producer (for telemetry.snapshot and benchmark.completed events).
	producer, err := events.NewProducer(cfg.RedpandaBrokers, logger)
	if err != nil {
		logger.Warn("failed to connect to redpanda, running without event publishing", slog.String("error", err.Error()))
	} else {
		defer producer.Close()
	}

	// onSnapshot publishes a telemetry snapshot event.
	onSnapshot := func(benchmarkID string, m telemetry.Metrics) {
		if producer == nil {
			return
		}
		pubCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		event := events.TelemetrySnapshotEvent{
			BenchmarkID: benchmarkID,
			Timestamp:   time.Now().UTC(),
			Metrics: contractbenchmark.TelemetryMetrics{
				CurrentTPS:              m.CurrentTPS,
				AvgLatencyMs:            m.AvgLatencyMs,
				TotalOrdersSent:         m.TotalOrdersSent,
				TotalOrdersAcknowledged: m.TotalOrdersAcknowledged,
				TotalErrors:             m.TotalErrors,
				P50LatencyMs:            m.P50LatencyMs,
				P90LatencyMs:            m.P90LatencyMs,
				P99LatencyMs:            m.P99LatencyMs,
			},
		}
		if err := producer.PublishTelemetrySnapshot(pubCtx, event); err != nil {
			logger.Warn("failed to publish telemetry snapshot",
				slog.String("benchmark_id", benchmarkID),
				slog.String("error", err.Error()),
			)
		}
	}

	runner := fleet.NewRunner(logger, onSnapshot)

	// Redpanda consumer: listens for benchmark.started events.
	consumer, err := events.NewConsumerWithOptions(
		cfg.RedpandaBrokers,
		"bot-fleet-group",
		[]string{events.TopicBenchmarkStarted},
		logger,
		events.ConsumerOptions{PartitionConcurrency: 4},
	)
	if err != nil {
		logger.Warn("failed to connect to redpanda consumer, bot fleet will not auto-start", slog.String("error", err.Error()))
	} else {
		events.RegisterJSONHandler[events.BenchmarkStartedEvent](consumer, events.TopicBenchmarkStarted, func(ctx context.Context, key string, event events.BenchmarkStartedEvent) error {
			logger.Info("benchmark started event received, spawning bot fleet",
				slog.String("benchmark_id", event.BenchmarkID),
				slog.String("submission_id", event.SubmissionID),
				slog.String("service_url", event.ServiceURL),
			)

			// Parse custom config
			var orderProfile bot.OrderProfile
			botCount := cfg.DefaultBotConcurrency
			duration := int32(cfg.DefaultDurationSeconds)
			ops := int32(cfg.DefaultOrdersPerSecond)

			presetName := event.Config.Preset
			if presetName != "" && presetName != "custom" {
				p := fleet.GetPreset(presetName)
				botCount = p.BotConcurrency
				duration = p.DurationSeconds
				ops = p.OrdersPerSecond
				orderProfile = p.Profile
			} else if presetName == "custom" {
				// Use values from config if custom
				if event.Config.BotCount > 0 {
					botCount = int(event.Config.BotCount)
				}
				if event.Config.DurationSeconds > 0 {
					duration = event.Config.DurationSeconds
				}
				if event.Config.OrdersPerSecond > 0 {
					ops = event.Config.OrdersPerSecond
				}
				orderProfile = fleet.DefaultProfile
			} else {
				orderProfile = fleet.DefaultProfile
			}

			runCfg := fleet.RunConfig{
				BenchmarkID:     event.BenchmarkID,
				SubmissionID:    event.SubmissionID,
				ServiceURL:      event.ServiceURL,
				BotConcurrency:  botCount,
				DurationSeconds: duration,
				OrdersPerSecond: ops,
				HTTPTimeoutMs:   cfg.BotHTTPTimeoutMs,
				TracesDir:       cfg.TracesDir,
				OrderProfile:    orderProfile,
			}

			go func() {
				runResult := runner.Run(context.Background(), runCfg)
				finalMetrics := runResult.Metrics

				if producer == nil {
					return
				}
				pubCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				finishedEvent := events.BenchmarkFinishedEvent{
					BenchmarkID:    event.BenchmarkID,
					SubmissionID:   event.SubmissionID,
					TPS:            float64(finalMetrics.TotalOrdersSent) / float64(duration),
					P50LatencyMs:   finalMetrics.P50LatencyMs,
					P90LatencyMs:   finalMetrics.P90LatencyMs,
					P99LatencyMs:   finalMetrics.P99LatencyMs,
					TotalOrders:    finalMetrics.TotalOrdersSent,
					FailedOrders:   finalMetrics.TotalErrors,
					ElapsedSeconds: int64(duration),
					FinishedAt:     time.Now().UTC(),
				}
				if err := producer.PublishBenchmarkFinished(pubCtx, finishedEvent); err != nil {
					logger.Warn("failed to publish benchmark.completed event",
						slog.String("benchmark_id", event.BenchmarkID),
						slog.String("error", err.Error()),
					)
				} else {
					logger.Info("benchmark.completed event published",
						slog.String("benchmark_id", event.BenchmarkID),
						slog.String("submission_id", event.SubmissionID),
					)
				}

				if runResult.TracePath != "" {
					traceEvent := events.TraceAvailableEvent{
						BenchmarkID: event.BenchmarkID,
						FilePath:    runResult.TracePath,
						CreatedAt:   time.Now().UTC(),
					}
					if err := producer.PublishTraceAvailable(pubCtx, traceEvent); err != nil {
						logger.Warn("failed to publish trace_available event", slog.String("error", err.Error()))
					} else {
						logger.Info("benchmark.trace_available event published", slog.String("file_path", runResult.TracePath))
					}
				}
			}()

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

	// HTTP server: health endpoint.
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
			"service": "bot-fleet",
			"version": "v1",
			"dependencies": gin.H{
				"redpanda": redpandaStatus,
			},
			"presets":   fleet.PresetNames(),
			"timestamp": time.Now().UTC(),
		})
	})

	// Checkpoint endpoint: dry-run a single order against a given URL for testing.
	router.POST("/api/v1/fleet/dryrun", func(c *gin.Context) {
		var req struct {
			ServiceURL string `json:"service_url" binding:"required"`
			BotCount   int    `json:"bot_count"`
			Duration   int32  `json:"duration_seconds"`
			OPS        int32  `json:"orders_per_second"`
			Preset     string `json:"preset"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var orderProfile bot.OrderProfile = fleet.DefaultProfile
		if req.Preset != "" {
			p := fleet.GetPreset(req.Preset)
			orderProfile = p.Profile
			if req.BotCount == 0 {
				req.BotCount = p.BotConcurrency
			}
			if req.Duration == 0 {
				req.Duration = p.DurationSeconds
			}
			if req.OPS == 0 {
				req.OPS = p.OrdersPerSecond
			}
		}
		if req.BotCount <= 0 {
			req.BotCount = cfg.DefaultBotConcurrency
		}
		if req.Duration <= 0 {
			req.Duration = int32(cfg.DefaultDurationSeconds)
		}
		if req.OPS <= 0 {
			req.OPS = int32(cfg.DefaultOrdersPerSecond)
		}

		benchmarkID := uuid.New().String()
		runCfg := fleet.RunConfig{
			BenchmarkID:     benchmarkID,
			SubmissionID:    "dryrun",
			ServiceURL:      req.ServiceURL,
			BotConcurrency:  req.BotCount,
			DurationSeconds: req.Duration,
			OrdersPerSecond: req.OPS,
			HTTPTimeoutMs:   cfg.BotHTTPTimeoutMs,
			TracesDir:       cfg.TracesDir,
			OrderProfile:    orderProfile,
		}

		go func() {
			runner.Run(context.Background(), runCfg)
		}()

		c.JSON(http.StatusAccepted, gin.H{
			"benchmark_id": benchmarkID,
			"message":      "dry run started",
			"config": gin.H{
				"bot_count":         req.BotCount,
				"duration_seconds":  req.Duration,
				"orders_per_second": req.OPS,
				"service_url":       req.ServiceURL,
			},
		})
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("bot-fleet service listening on :%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("bot-fleet server error: %v", err)
	}
}
