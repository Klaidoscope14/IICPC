package domain

import "errors"

// Sentinel errors for use across service and handler layers.
// Handlers can check these with errors.Is() to return appropriate HTTP status codes.
var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrAlreadyExists = errors.New("already exists")
	ErrInternal      = errors.New("internal error")
)
