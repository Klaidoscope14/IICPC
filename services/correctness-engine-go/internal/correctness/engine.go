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

	// Compare expectedFills vs actualFills
	violations := e.compareFills(benchmarkID, expectedFills, actualFills)
	
	// Basic correctness score mapping
	score := 100.0
	if totalOrders > 0 {
		penaltyPerViolation := 100.0 / float64(totalOrders)
		score -= float64(violations) * penaltyPerViolation
	}
	
	if score < 0 {
		score = 0
	}
	
	if totalOrders == 0 {
		score = 0 // If no orders were sent, score is 0
	}

	return Result{
		Score:      score,
		Violations: violations,
	}, nil
}

func (e *Engine) compareFills(benchmarkID string, expected []ExpectedMatch, actual []correctness.ExecutionReport) int32 {
	var violations int32

	// This is a simplified comparison for the MVP.
	// It checks if the total filled quantity matches the expected matched quantity.
	// A full production engine would do a 1-to-1 match of expected match vs actual execution report.
	
	expectedQty := int32(0)
	for _, m := range expected {
		// Each match represents a trade between 2 orders, so the total volume matched is m.Qty
		expectedQty += m.Qty
	}
	
	actualQty := int32(0)
	for _, a := range actual {
		// Actual fills are reported per order. We divide by 2 to get the "matched volume"
		// Assuming both buyer and seller get an execution report.
		actualQty += a.FilledQty
	}
	
	// If actual qty doesn't equal 2x expected qty, something is wrong.
	if actualQty != expectedQty*2 {
		e.logger.Warn("fill volume mismatch",
			slog.String("benchmark_id", benchmarkID),
			slog.Int("expected_volume", int(expectedQty)),
			slog.Int("actual_reported_volume", int(actualQty/2)),
		)
		
		// Each unmatched unit is a violation
		diff := expectedQty - (actualQty / 2)
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
