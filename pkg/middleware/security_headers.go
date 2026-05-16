package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SecurityHeadersConfig struct {
	ContentSecurityPolicy string
	FrameOptions          string
	ReferrerPolicy        string
	PermissionsPolicy     string
	HSTS                  bool
	HSTSMaxAgeSeconds     int
}

func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; object-src 'none'; frame-ancestors 'none'; base-uri 'self'",
		FrameOptions:          "DENY",
		ReferrerPolicy:        "no-referrer",
		PermissionsPolicy:     "camera=(), microphone=(), geolocation=()",
		HSTS:                  true,
		HSTSMaxAgeSeconds:     31536000,
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return SecurityHeadersWithConfig(DefaultSecurityHeadersConfig())
}

func SecurityHeadersWithConfig(cfg SecurityHeadersConfig) gin.HandlerFunc {
	if cfg.HSTSMaxAgeSeconds <= 0 {
		cfg.HSTSMaxAgeSeconds = 31536000
	}

	return func(c *gin.Context) {
		h := c.Writer.Header()
		setIfMissing(h, "X-Content-Type-Options", "nosniff")
		setIfMissing(h, "X-Frame-Options", cfg.FrameOptions)
		setIfMissing(h, "Referrer-Policy", cfg.ReferrerPolicy)
		setIfMissing(h, "Permissions-Policy", cfg.PermissionsPolicy)
		setIfMissing(h, "Content-Security-Policy", cfg.ContentSecurityPolicy)
		if cfg.HSTS && isHTTPS(c.Request) {
			setIfMissing(h, "Strict-Transport-Security", "max-age="+strconv.Itoa(cfg.HSTSMaxAgeSeconds)+"; includeSubDomains")
		}
		c.Next()
	}
}

func setIfMissing(h http.Header, key string, value string) {
	if value != "" && h.Get(key) == "" {
		h.Set(key, value)
	}
}

func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}
