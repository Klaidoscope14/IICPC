package correctness

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/iicpc/pkg/contracts/correctness"
)

// Engine replays trace files and computes the correctness score.
type Engine struct {
	logger *slog.Logger
}

func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{logger: logger}
}

// Result contains the final score and number of violations found.
type Result struct {
	Score      float64
	Violations int32
}

// EvaluateTrace reads the JSONL trace file, reconstructs state, and compares expected vs actual fills.
func (e *Engine) EvaluateTrace(benchmarkID string, filePath string) (Result, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Result{}, fmt.Errorf("failed to open trace file: %w", err)
	}
	defer file.Close()

	orderbook := NewOrderbook()
	scanner := bufio.NewScanner(file)

	var expectedFills []ExpectedMatch
	var actualFills []correctness.ExecutionReport
	
	totalOrders := 0

	for scanner.Scan() {
		var event correctness.TraceEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			e.logger.Warn("failed to parse trace event", slog.String("error", err.Error()))
			continue
		}

		switch event.EventType {
		case correctness.TraceEventOrderSent:
			if event.OrderSent != nil {
				totalOrders++
				matches := orderbook.ProcessOrder(event.OrderSent, event.Timestamp)
				expectedFills = append(expectedFills, matches...)
			}
		case correctness.TraceEventExecution:
			if event.Execution != nil && event.Execution.FilledQty > 0 {
				actualFills = append(actualFills, *event.Execution)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Result{}, fmt.Errorf("error reading trace file: %w", err)
	}

	expectedVolume := matchedVolume(expectedFills)
	actualVolume := reportedMatchedVolume(actualFills)
	violations := e.compareFills(benchmarkID, expectedVolume, actualVolume)

	score := volumeScore(expectedVolume, actualVolume)
	if totalOrders == 0 {
		score = 0
	}

	return Result{
		Score:      score,
		Violations: violations,
	}, nil
}

func matchedVolume(expected []ExpectedMatch) int32 {
	volume := int32(0)
	for _, match := range expected {
		volume += match.Qty
	}
	return volume
}

func reportedMatchedVolume(actual []correctness.ExecutionReport) int32 {
	volume := int32(0)
	for _, report := range actual {
		volume += report.FilledQty
	}
	return volume / 2
}

func volumeScore(expectedVolume int32, actualVolume int32) float64 {
	if expectedVolume == 0 && actualVolume == 0 {
		return 100
	}

	denominator := expectedVolume
	if actualVolume > denominator {
		denominator = actualVolume
	}
	if denominator <= 0 {
		return 0
	}

	diff := expectedVolume - actualVolume
	if diff < 0 {
		diff = -diff
	}

	score := 100.0 * (1.0 - float64(diff)/float64(denominator))
	if score < 0 {
		return 0
	}
	return score
}

func (e *Engine) compareFills(benchmarkID string, expectedVolume int32, actualVolume int32) int32 {
	var violations int32

	// This is a simplified comparison for the MVP.
	// It checks if the total filled quantity matches the expected matched quantity.
	// A full production engine would do a 1-to-1 match of expected match vs actual execution report.

	if actualVolume != expectedVolume {
		e.logger.Warn("fill volume mismatch",
			slog.String("benchmark_id", benchmarkID),
			slog.Int("expected_volume", int(expectedVolume)),
			slog.Int("actual_reported_volume", int(actualVolume)),
		)

		// Each unmatched unit is a violation
		diff := expectedVolume - actualVolume
		if diff < 0 {
			diff = -diff
		}
		violations += diff
	}

	if violations == 0 {
		e.logger.Info("trace mathematically perfect", slog.String("benchmark_id", benchmarkID))
	} else {
		e.logger.Warn("trace contained correctness violations",
			slog.String("benchmark_id", benchmarkID),
			slog.Int("violations", int(violations)),
		)
	}

	return violations
}
