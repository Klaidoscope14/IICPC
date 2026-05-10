package service

import (
	"math"

	"github.com/iicpc/benchmark-orchestrator-go/internal/domain"
)

// ScoringService computes composite benchmark scores for leaderboard ranking.
type ScoringService struct {
	weights domain.ScoringWeights
}

// NewScoringService creates a ScoringService with the given weights.
func NewScoringService(weights domain.ScoringWeights) *ScoringService {
	return &ScoringService{weights: weights}
}

// ComputeScore calculates a composite score from benchmark metrics.
//
// Formula:
//   - TPS component:       normalized TPS (higher is better)
//   - Latency component:   inverse normalized latency (lower is better)
//   - Correctness component: success rate percentage
//
// Each component is weighted according to the configured ScoringWeights.
// The final score is scaled to 0-100.
func (s *ScoringService) ComputeScore(metrics domain.TelemetryMetrics) float64 {
	// TPS score: log-scale normalization (cap at 100k TPS for normalization).
	tpsScore := 0.0
	if metrics.CurrentTPS > 0 {
		tpsScore = math.Min(math.Log10(metrics.CurrentTPS)/5.0, 1.0) // log10(100000) = 5
	}

	// Latency score: inverse relationship (lower latency = higher score).
	// p99 latency is used as it represents worst-case behavior.
	latencyScore := 0.0
	if metrics.P99LatencyMs > 0 {
		// 0.1ms → score 1.0, 10ms → score 0.5, 1000ms → score ~0.1
		latencyScore = math.Min(1.0/(1.0+math.Log10(metrics.P99LatencyMs+1)), 1.0)
	}

	// Correctness score: ratio of acknowledged to sent orders.
	correctnessScore := 0.0
	if metrics.TotalOrdersSent > 0 {
		successRate := float64(metrics.TotalOrdersAcknowledged) / float64(metrics.TotalOrdersSent)
		errorPenalty := float64(metrics.TotalErrors) / float64(metrics.TotalOrdersSent)
		correctnessScore = math.Max(0, successRate-errorPenalty)
	}

	// Weighted composite score scaled to 0-100.
	composite := (s.weights.TPSWeight*tpsScore +
		s.weights.LatencyWeight*latencyScore +
		s.weights.CorrectnessWeight*correctnessScore) * 100.0

	return math.Round(composite*100) / 100 // Round to 2 decimal places.
}
