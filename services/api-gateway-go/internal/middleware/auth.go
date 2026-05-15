package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/api-gateway-go/internal/httpx"
	contractapi "github.com/iicpc/pkg/contracts/api"
)

// OptionalBearerAuth is disabled when token is empty. When configured, it
// protects API and WebSocket routes with a static bearer token for the MVP.
func OptionalBearerAuth(token string) gin.HandlerFunc {
	token = strings.TrimSpace(token)

	return func(c *gin.Context) {
		if token == "" || c.Request.Method == http.MethodOptions || isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		got := bearerToken(c)
		if got == "" {
			httpx.WriteGinError(c, http.StatusUnauthorized, contractapi.ErrorUnauthorized, "missing bearer token")
			return
		}

		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			httpx.WriteGinError(c, http.StatusUnauthorized, contractapi.ErrorUnauthorized, "invalid bearer token")
			return
		}

		c.Next()
	}
}

func isPublicPath(path string) bool {
	return path == "/health" || path == "/metrics"
}

func bearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	}
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		return strings.TrimSpace(c.Query("access_token"))
	}
	return ""
}
