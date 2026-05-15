package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	contractapi "github.com/iicpc/pkg/contracts/api"
	"github.com/iicpc/pkg/logging"
)

func WriteGinError(c *gin.Context, status int, code contractapi.ErrorCode, message string, details ...contractapi.ErrorDetail) {
	c.AbortWithStatusJSON(status, contractapi.NewErrorResponse(code, message, RequestID(c.Request), details...))
}

func WriteHTTPError(w http.ResponseWriter, r *http.Request, status int, code contractapi.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(contractapi.NewErrorResponse(code, message, RequestID(r)))
}

func RequestID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if id := logging.RequestIDFromContext(r.Context()); id != "" {
		return id
	}
	return r.Header.Get("X-Request-ID")
}

func ErrorCodeForStatus(status int) contractapi.ErrorCode {
	switch status {
	case http.StatusBadRequest:
		return contractapi.ErrorInvalidInput
	case http.StatusUnauthorized:
		return contractapi.ErrorUnauthorized
	case http.StatusForbidden:
		return contractapi.ErrorForbidden
	case http.StatusNotFound:
		return contractapi.ErrorNotFound
	case http.StatusConflict:
		return contractapi.ErrorConflict
	case http.StatusRequestEntityTooLarge:
		return contractapi.ErrorPayloadTooLarge
	case http.StatusUnsupportedMediaType:
		return contractapi.ErrorUnsupportedMedia
	case http.StatusTooManyRequests:
		return contractapi.ErrorRateLimited
	case http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return contractapi.ErrorServiceUnavailable
	default:
		if status >= 500 {
			return contractapi.ErrorInternal
		}
		return contractapi.ErrorInvalidInput
	}
}
