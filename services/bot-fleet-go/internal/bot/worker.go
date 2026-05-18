package bot

import (
	"context"
	"log/slog"
	"time"
)

// WorkerConfig controls the behaviour of a single bot worker.
type WorkerConfig struct {
	ServiceURL   string
	BenchmarkID  string
	SubmissionID string
	WorkerID     int
	// Rate gate: the worker sleeps between requests to honour OPS targets.
	// Inter-request delay = 1s / (ordersPerSecond / numWorkers).
	InterRequestDelay time.Duration
	HTTPTimeoutMs     int
}

// RunWorker is a single bot goroutine. It sends orders until ctx is cancelled,
// pushing each Result into the results channel.
func RunWorker(ctx context.Context, cfg WorkerConfig, results chan<- Result, logger *slog.Logger) {
	client := NewHTTPClient(cfg.ServiceURL, cfg.HTTPTimeoutMs)

	logger.Debug("bot spawned",
		slog.String("benchmark_id", cfg.BenchmarkID),
		slog.Int("worker_id", cfg.WorkerID),
		slog.String("service_url", cfg.ServiceURL),
	)

	ticker := time.NewTicker(cfg.InterRequestDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			order := RandomOrder()

			logger.Debug("request sent",
				slog.String("benchmark_id", cfg.BenchmarkID),
				slog.Int("worker_id", cfg.WorkerID),
				slog.String("order_type", string(order.Type)),
				slog.String("symbol", string(order.Symbol)),
			)

			res := client.Send(ctx, order)

			switch {
			case res.TimedOut:
				logger.Debug("request timeout",
					slog.String("benchmark_id", cfg.BenchmarkID),
					slog.Int("worker_id", cfg.WorkerID),
					slog.String("request_id", res.RequestID),
				)
			case res.Err != nil:
				logger.Debug("request failure",
					slog.String("benchmark_id", cfg.BenchmarkID),
					slog.Int("worker_id", cfg.WorkerID),
					slog.String("error", res.Err.Error()),
				)
			default:
				logger.Debug("response received",
					slog.String("benchmark_id", cfg.BenchmarkID),
					slog.Int("worker_id", cfg.WorkerID),
					slog.Int("status", res.StatusCode),
					slog.Float64("latency_ms", res.LatencyMs),
				)
			}

			select {
			case results <- res:
			default:
				// Drop result if channel is full rather than blocking.
			}
		}
	}
}
