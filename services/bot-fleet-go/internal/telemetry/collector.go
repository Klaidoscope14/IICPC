package telemetry

import (
	"math"
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

	// Latency histogram — protected by mu. Buckets are cumulative for the run
	// so final p50/p90/p99 stay stable without sorting hot-path samples.
	latencyBuckets []uint64
	latencyCount   uint64
	latencySumMs   float64

	// TPS tracking — protected by mu.
	windowStart    time.Time
	windowRequests int64
}

// NewCollector creates an initialised telemetry collector.
func NewCollector() *Collector {
	return &Collector{
		windowStart:    time.Now(),
		latencyBuckets: make([]uint64, latencyBucketCount),
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
		c.latencyBuckets[latencyBucketIndex(latencyMs)]++
		c.latencyCount++
		c.latencySumMs += latencyMs
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

	p50, p90, p99 := c.latencyPercentilesLocked()
	avgLatency := 0.0
	if c.latencyCount > 0 {
		avgLatency = c.latencySumMs / float64(c.latencyCount)
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

const (
	fineBucketWidthMs   = 0.1
	fineBucketMaxMs     = 1000.0
	midBucketWidthMs    = 1.0
	midBucketMaxMs      = 10000.0
	coarseBucketWidthMs = 10.0
	coarseBucketMaxMs   = 60000.0

	fineBucketCount    = int(fineBucketMaxMs / fineBucketWidthMs)
	midBucketCount     = int((midBucketMaxMs - fineBucketMaxMs) / midBucketWidthMs)
	coarseBucketCount  = int((coarseBucketMaxMs - midBucketMaxMs) / coarseBucketWidthMs)
	latencyBucketCount = fineBucketCount + midBucketCount + coarseBucketCount + 1
)

func (c *Collector) latencyPercentilesLocked() (p50, p90, p99 float64) {
	if c.latencyCount == 0 {
		return 0, 0, 0
	}

	p50 = percentileFromBuckets(c.latencyBuckets, c.latencyCount, 50)
	p90 = percentileFromBuckets(c.latencyBuckets, c.latencyCount, 90)
	p99 = percentileFromBuckets(c.latencyBuckets, c.latencyCount, 99)
	return
}

func percentileFromBuckets(buckets []uint64, total uint64, p float64) float64 {
	if total == 0 {
		return 0
	}
	threshold := uint64(math.Ceil(float64(total) * p / 100))
	var seen uint64
	for idx, count := range buckets {
		seen += count
		if seen >= threshold {
			return math.Round(latencyBucketValue(idx)*100) / 100
		}
	}
	return coarseBucketMaxMs
}

func latencyBucketIndex(ms float64) int {
	switch {
	case ms < fineBucketMaxMs:
		idx := int(ms / fineBucketWidthMs)
		if idx < 0 {
			return 0
		}
		return idx
	case ms < midBucketMaxMs:
		return fineBucketCount + int((ms-fineBucketMaxMs)/midBucketWidthMs)
	case ms < coarseBucketMaxMs:
		return fineBucketCount + midBucketCount + int((ms-midBucketMaxMs)/coarseBucketWidthMs)
	default:
		return latencyBucketCount - 1
	}
}

func latencyBucketValue(idx int) float64 {
	switch {
	case idx < fineBucketCount:
		return (float64(idx) + 0.5) * fineBucketWidthMs
	case idx < fineBucketCount+midBucketCount:
		return fineBucketMaxMs + (float64(idx-fineBucketCount)+0.5)*midBucketWidthMs
	case idx < fineBucketCount+midBucketCount+coarseBucketCount:
		return midBucketMaxMs + (float64(idx-fineBucketCount-midBucketCount)+0.5)*coarseBucketWidthMs
	default:
		return coarseBucketMaxMs
	}
}
