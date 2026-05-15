package middleware

import (
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/api-gateway-go/internal/httpx"
	contractapi "github.com/iicpc/pkg/contracts/api"
)

func RequestValidator(maxBodyBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") && c.Request.URL.Path != "/api/v1" {
			httpx.WriteGinError(c, http.StatusNotFound, contractapi.ErrorNotFound, "unsupported API version")
			return
		}

		if maxBodyBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		}

		if isWebSocketUpgrade(c.Request) {
			c.Next()
			return
		}

		if err := validateContentType(c.Request); err != nil {
			httpx.WriteGinError(c, http.StatusUnsupportedMediaType, contractapi.ErrorUnsupportedMedia, err.Error())
			return
		}
		if err := validatePagination(c.Request); err != nil {
			httpx.WriteGinError(c, http.StatusBadRequest, contractapi.ErrorInvalidInput, err.Error())
			return
		}

		c.Next()
	}
}

func validateContentType(r *http.Request) error {
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return nil
	}
	if r.ContentLength == 0 {
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return errString("content-type header is required")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errString("invalid content-type header")
	}

	if r.Method == http.MethodPost && r.URL.Path == "/api/v1/submissions" {
		if mediaType == "multipart/form-data" {
			return nil
		}
		return errString("submission uploads require multipart/form-data")
	}

	if mediaType != "application/json" {
		return errString("request body must be application/json")
	}
	return nil
}

func validatePagination(r *http.Request) error {
	query := r.URL.Query()
	for _, key := range []string{"page", "page_size", "limit"} {
		value := query.Get(key)
		if value == "" {
			continue
		}
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return errString(key + " must be a positive integer")
		}
		if key != "page" && n > 1000 {
			return errString(key + " is too large")
		}
	}
	return nil
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket") && strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

type errString string

func (e errString) Error() string { return string(e) }
