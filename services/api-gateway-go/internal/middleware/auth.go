package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iicpc/api-gateway-go/internal/httpx"
	contractapi "github.com/iicpc/pkg/contracts/api"
)

// JWTAuthMiddleware validates the JWT token and injects claims into headers.
func JWTAuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodOptions || isPublicPath(c.Request.Method, c.Request.URL.Path) {
			c.Next()
			return
		}

		tokenString := bearerToken(c)
		if tokenString == "" {
			httpx.WriteGinError(c, http.StatusUnauthorized, contractapi.ErrorUnauthorized, "missing bearer token")
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			httpx.WriteGinError(c, http.StatusUnauthorized, contractapi.ErrorUnauthorized, "invalid token")
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			var resolvedContestantID string
			if userID, ok := claims["user_id"].(string); ok {
				c.Request.Header.Set("X-User-ID", userID)
				resolvedContestantID = userID
			}
			if role, ok := claims["role"].(string); ok {
				c.Request.Header.Set("X-User-Role", role)
			}
			if teamID, ok := claims["team_id"].(string); ok && teamID != "" {
				c.Request.Header.Set("X-Team-ID", teamID)
				resolvedContestantID = teamID
			}
			if resolvedContestantID != "" {
				c.Request.Header.Set("X-Contestant-ID", resolvedContestantID)
			}
		} else {
			httpx.WriteGinError(c, http.StatusUnauthorized, contractapi.ErrorUnauthorized, "invalid token claims")
			c.Abort()
			return
		}

		c.Next()
	}
}

func isPublicPath(method, path string) bool {
	if method == http.MethodGet && path == "/api/v1/admin/hackathon/dates" {
		return true
	}
	return path == "/health" || path == "/metrics" || strings.HasPrefix(path, "/api/auth/") || path == "/api/v1/admin/login"
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
