package telemetry

import (
	"context"
	"log/slog"
	"time"
)

// Publisher periodically publishes telemetry snapshots via a callback.
type Publisher struct {
	collector   *Collector
	interval    time.Duration
	benchmarkID string
	logger      *slog.Logger
	onSnapshot  func(benchmarkID string, m Metrics)
}

// NewPublisher creates a Publisher that calls onSnapshot every interval.
func NewPublisher(
	collector *Collector,
	benchmarkID string,
	interval time.Duration,
	logger *slog.Logger,
	onSnapshot func(benchmarkID string, m Metrics),
) *Publisher {
	return &Publisher{
		collector:   collector,
		interval:    interval,
		benchmarkID: benchmarkID,
		logger:      logger,
		onSnapshot:  onSnapshot,
	}
}

// Run starts the publish loop, blocking until ctx is cancelled.
func (p *Publisher) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Publish one final snapshot before exiting.
			snap := p.collector.Snapshot()
			p.onSnapshot(p.benchmarkID, snap)
			p.logger.Info("telemetry publisher stopped",
				slog.String("benchmark_id", p.benchmarkID),
				slog.Int("total_sent", int(snap.TotalOrdersSent)),
				slog.Int("total_errors", int(snap.TotalErrors)),
			)
			return
		case <-ticker.C:
			snap := p.collector.Snapshot()
			p.logger.Info("telemetry snapshot",
				slog.String("type", "telemetry_metric"),
				slog.String("benchmark_id", p.benchmarkID),
				slog.Float64("tps", snap.CurrentTPS),
				slog.Float64("avg_latency_ms", snap.AvgLatencyMs),
				slog.Float64("p99_latency_ms", snap.P99LatencyMs),
				slog.Int("total_sent", int(snap.TotalOrdersSent)),
				slog.Int("total_errors", int(snap.TotalErrors)),
			)
			p.onSnapshot(p.benchmarkID, snap)
		}
	}
}
