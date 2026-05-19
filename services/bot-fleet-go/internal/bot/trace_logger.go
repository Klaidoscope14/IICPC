package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/iicpc/pkg/contracts/correctness"
)

// TraceLogger records bot interactions to a JSONL file for correctness engine replay.
type TraceLogger struct {
	file   *os.File
	enc    *json.Encoder
	mu     sync.Mutex
	closed bool
}

// NewTraceLogger creates a new trace logger writing to the specified path.
func NewTraceLogger(benchmarkID string, tracesDir string) (*TraceLogger, string, error) {
	if err := os.MkdirAll(tracesDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create traces directory: %w", err)
	}

	filePath := filepath.Join(tracesDir, fmt.Sprintf("%s.trace.jsonl", benchmarkID))
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open trace file: %w", err)
	}

	return &TraceLogger{
		file: file,
		enc:  json.NewEncoder(file),
	}, filePath, nil
}

// Log records a single trace event safely.
func (l *TraceLogger) Log(event correctness.TraceEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	return l.enc.Encode(event)
}

// Close closes the underlying file.
func (l *TraceLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	return l.file.Close()
}
