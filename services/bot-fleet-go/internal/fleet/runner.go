package fleet

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/iicpc/bot-fleet-go/internal/bot"
	"github.com/iicpc/bot-fleet-go/internal/telemetry"
)

// RunConfig holds everything the fleet runner needs.
type RunConfig struct {
	BenchmarkID     string
	SubmissionID    string
	ServiceURL      string
	BotConcurrency  int
	DurationSeconds int32
	OrdersPerSecond int32
	HTTPTimeoutMs   int
	TracesDir       string
	OrderProfile    bot.OrderProfile
}

// FinalMetrics is returned when the run completes.
type FinalMetrics = telemetry.Metrics

// RunResult contains the final metrics and the path to the trace file (if generated).
type RunResult struct {
	Metrics   FinalMetrics
	TracePath string
}

// Runner orchestrates the worker pool and telemetry collection.
type Runner struct {
	logger     *slog.Logger
	onSnapshot func(benchmarkID string, m telemetry.Metrics)
}

// NewRunner creates a fleet Runner.
func NewRunner(logger *slog.Logger, onSnapshot func(benchmarkID string, m telemetry.Metrics)) *Runner {
	return &Runner{
		logger:     logger,
		onSnapshot: onSnapshot,
	}
}

// Run starts the fleet, blocks for the benchmark duration, and returns final metrics.
func (r *Runner) Run(ctx context.Context, cfg RunConfig) RunResult {
	duration := time.Duration(cfg.DurationSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	var traceLogger *bot.TraceLogger
	var traceFilePath string
	var err error
	if cfg.TracesDir != "" {
		traceLogger, traceFilePath, err = bot.NewTraceLogger(cfg.BenchmarkID, cfg.TracesDir)
		if err != nil {
			r.logger.Error("failed to initialize trace logger, proceeding without tracing", slog.String("error", err.Error()))
			traceLogger = nil
			traceFilePath = ""
		} else {
			defer traceLogger.Close()
		}
	}

	if traceLogger != nil {
		wsClient := bot.NewWSClient(cfg.ServiceURL, traceLogger, r.logger)
		go func() {
			if err := wsClient.Run(runCtx); err != nil {
				r.logger.Error("websocket client error", slog.String("error", err.Error()))
			}
		}()
	}

	collector := telemetry.NewCollector()

	// Calculate inter-request delay per worker so aggregate OPS ≈ cfg.OrdersPerSecond.
	// Each worker fires once per (numWorkers / OPS) seconds.
	interRequestDelay := time.Second // default 1 OPS per worker
	if cfg.OrdersPerSecond > 0 && cfg.BotConcurrency > 0 {
		totalDelayNs := int64(time.Second) * int64(cfg.BotConcurrency) / int64(cfg.OrdersPerSecond)
		if totalDelayNs > 0 {
			interRequestDelay = time.Duration(totalDelayNs)
		}
	}

	// Results channel — buffered to avoid blocking workers.
	resultsCh := make(chan bot.Result, cfg.BotConcurrency*10)

	r.logger.Info("bot fleet starting",
		slog.String("benchmark_id", cfg.BenchmarkID),
		slog.String("submission_id", cfg.SubmissionID),
		slog.String("service_url", cfg.ServiceURL),
		slog.Int("bot_concurrency", cfg.BotConcurrency),
		slog.Int("orders_per_second", int(cfg.OrdersPerSecond)),
		slog.Int("duration_seconds", int(cfg.DurationSeconds)),
	)

	// Launch workers.
	var wg sync.WaitGroup
	for i := 0; i < cfg.BotConcurrency; i++ {
		wg.Add(1)
		workerCfg := bot.WorkerConfig{
			ServiceURL:        cfg.ServiceURL,
			BenchmarkID:       cfg.BenchmarkID,
			SubmissionID:      cfg.SubmissionID,
			WorkerID:          i + 1,
			InterRequestDelay: interRequestDelay,
			HTTPTimeoutMs:     cfg.HTTPTimeoutMs,
			TraceLogger:       traceLogger,
			OrderProfile:      cfg.OrderProfile,
		}
		go func(wcfg bot.WorkerConfig) {
			defer wg.Done()
			bot.RunWorker(runCtx, wcfg, resultsCh, r.logger)
		}(workerCfg)
	}

	// Results drainer — feeds results into the collector.
	var drainWG sync.WaitGroup
	drainWG.Add(1)
	go func() {
		defer drainWG.Done()
		for res := range resultsCh {
			sent := res.StatusCode > 0
			collector.Record(sent, res.LatencyMs, res.StatusCode, res.TimedOut, res.Err)
		}
	}()

	// Telemetry publisher — emits a snapshot every second.
	telPublisher := telemetry.NewPublisher(collector, cfg.BenchmarkID, time.Second, r.logger, r.onSnapshot)
	go telPublisher.Run(runCtx)

	// Wait for all workers to finish (context deadline fires).
	wg.Wait()
	close(resultsCh)
	drainWG.Wait()

	final := collector.Snapshot()

	r.logger.Info("bot fleet completed",
		slog.String("benchmark_id", cfg.BenchmarkID),
		slog.Int("total_sent", int(final.TotalOrdersSent)),
		slog.Int("total_acked", int(final.TotalOrdersAcknowledged)),
		slog.Int("total_errors", int(final.TotalErrors)),
		slog.Float64("p99_latency_ms", final.P99LatencyMs),
	)

	return RunResult{
		Metrics:   final,
		TracePath: traceFilePath,
	}
}
