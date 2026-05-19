package telemetry

import (
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Collector aggregates metrics from concurrent bot workers in a thread-safe way.
type Collector struct {
	mu sync.Mutex

	// Atomic counters for lock-free hot path.
	ordersSent         atomic.Int64
	ordersAcknowledged atomic.Int64
	totalErrors        atomic.Int64
	totalTimeouts      atomic.Int64

	// Latency samples — protected by mu.
	latencySamples []float64

	// TPS tracking — protected by mu.
	windowStart    time.Time
	windowRequests int64
}

// NewCollector creates an initialised telemetry collector.
func NewCollector() *Collector {
	return &Collector{
		windowStart:    time.Now(),
		latencySamples: make([]float64, 0, 4096),
	}
}

// Record registers the result of a single bot request.
func (c *Collector) Record(sent bool, latencyMs float64, statusCode int, timedOut bool, err error) {
	c.ordersSent.Add(1)

	if sent && statusCode >= 200 && statusCode < 300 {
		c.ordersAcknowledged.Add(1)
	}
	if timedOut {
		c.totalTimeouts.Add(1)
		c.totalErrors.Add(1)
		return
	}
	if err != nil || (statusCode >= 400) {
		c.totalErrors.Add(1)
		return
	}

	if latencyMs > 0 {
		c.mu.Lock()
		c.latencySamples = append(c.latencySamples, latencyMs)
		c.windowRequests++
		c.mu.Unlock()
	}
}

// Snapshot returns an atomic point-in-time view of the current metrics.
func (c *Collector) Snapshot() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.windowStart).Seconds()
	tps := 0.0
	if elapsed > 0 {
		tps = float64(c.windowRequests) / elapsed
	}

	// Reset the TPS window each snapshot.
	c.windowStart = now
	c.windowRequests = 0

	p50, p90, p99 := latencyPercentiles(c.latencySamples)
	avgLatency := 0.0
	if len(c.latencySamples) > 0 {
		sum := 0.0
		for _, v := range c.latencySamples {
			sum += v
		}
		avgLatency = sum / float64(len(c.latencySamples))
	}
	// Keep only the last 10k samples to bound memory.
	if len(c.latencySamples) > 10000 {
		c.latencySamples = c.latencySamples[len(c.latencySamples)-10000:]
	}

	return Metrics{
		CurrentTPS:              math.Round(tps*100) / 100,
		AvgLatencyMs:            math.Round(avgLatency*100) / 100,
		TotalOrdersSent:         int32(c.ordersSent.Load()),
		TotalOrdersAcknowledged: int32(c.ordersAcknowledged.Load()),
		TotalErrors:             int32(c.totalErrors.Load()),
		TotalTimeouts:           int32(c.totalTimeouts.Load()),
		P50LatencyMs:            p50,
		P90LatencyMs:            p90,
		P99LatencyMs:            p99,
	}
}

// Metrics is the snapshot type returned by Collector.Snapshot().
type Metrics struct {
	CurrentTPS              float64
	AvgLatencyMs            float64
	TotalOrdersSent         int32
	TotalOrdersAcknowledged int32
	TotalErrors             int32
	TotalTimeouts           int32
	P50LatencyMs            float64
	P90LatencyMs            float64
	P99LatencyMs            float64
}

func latencyPercentiles(samples []float64) (p50, p90, p99 float64) {
	if len(samples) == 0 {
		return 0, 0, 0
	}
	sorted := make([]float64, len(samples))
	copy(sorted, samples)
	sort.Float64s(sorted)

	p50 = percentile(sorted, 50)
	p90 = percentile(sorted, 90)
	p99 = percentile(sorted, 99)
	return
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(sorted))*p/100)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return math.Round(sorted[idx]*100) / 100
}
