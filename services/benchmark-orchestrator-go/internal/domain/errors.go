package domain

import "errors"

// Sentinel errors for the orchestrator domain.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrInternal     = errors.New("internal error")
)
