package domain

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrInvalidInput     = errors.New("invalid input")
	ErrInternal         = errors.New("internal error")
	ErrExtractionFailed = errors.New("zip extraction failed")
	ErrPathTraversal    = errors.New("path traversal detected")
	ErrSizeBomb         = errors.New("extracted size exceeds limit")
	ErrTooManyFiles     = errors.New("too many files in archive")
	ErrDangerousFile    = errors.New("dangerous file blocked")
)
