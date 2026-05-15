package api

import "time"

// ErrorCode is a stable machine-readable API error identifier.
type ErrorCode string

const (
	ErrorInvalidInput       ErrorCode = "INVALID_INPUT"
	ErrorUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrorForbidden          ErrorCode = "FORBIDDEN"
	ErrorNotFound           ErrorCode = "NOT_FOUND"
	ErrorConflict           ErrorCode = "CONFLICT"
	ErrorPayloadTooLarge    ErrorCode = "PAYLOAD_TOO_LARGE"
	ErrorUnsupportedMedia   ErrorCode = "UNSUPPORTED_MEDIA_TYPE"
	ErrorRateLimited        ErrorCode = "RATE_LIMITED"
	ErrorInternal           ErrorCode = "INTERNAL_ERROR"
	ErrorServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// ErrorDetail carries optional structured context for one API error.
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ErrorResponse is the shared error payload returned by REST APIs.
type ErrorResponse struct {
	Success   bool      `json:"success"`
	Error     APIError  `json:"error"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type APIError struct {
	Code    ErrorCode     `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// Response is the common success envelope for APIs that benefit from metadata.
type Response[T any] struct {
	Success   bool      `json:"success"`
	Data      T         `json:"data"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// PageRequest captures cursor/limit pagination at service boundaries.
type PageRequest struct {
	Limit     int    `json:"limit" form:"limit"`
	PageToken string `json:"page_token,omitempty" form:"page_token"`
}

// PageResponse is the shared shape for paginated list responses.
type PageResponse[T any] struct {
	Items         []T    `json:"items"`
	NextPageToken string `json:"next_page_token,omitempty"`
	Count         int    `json:"count"`
}

// HealthResponse gives every service a consistent health payload.
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Version   string            `json:"version,omitempty"`
	Checks    map[string]string `json:"checks,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

func NewResponse[T any](data T, requestID string) Response[T] {
	return Response[T]{
		Success:   true,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

func NewErrorResponse(code ErrorCode, message string, requestID string, details ...ErrorDetail) ErrorResponse {
	return ErrorResponse{
		Success: false,
		Error: APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}
