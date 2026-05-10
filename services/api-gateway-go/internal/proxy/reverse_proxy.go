package proxy

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

// ReverseProxy forwards requests to a backend service.
type ReverseProxy struct {
	targetURL  *url.URL
	httpClient *http.Client
}

// NewReverseProxy creates a proxy that forwards to the given backend URL.
func NewReverseProxy(target string) (*ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return &ReverseProxy{
		targetURL: u,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Handler returns a Gin handler that proxies the request to the backend.
func (p *ReverseProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build the backend URL.
		targetURL := *p.targetURL
		targetURL.Path = c.Request.URL.Path
		targetURL.RawQuery = c.Request.URL.RawQuery

		// Create the proxied request.
		req, err := http.NewRequestWithContext(
			c.Request.Context(),
			c.Request.Method,
			targetURL.String(),
			c.Request.Body,
		)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create proxy request"})
			return
		}

		// Copy headers.
		for key, values := range c.Request.Header {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}
		req.Header.Set("X-Forwarded-For", c.ClientIP())
		req.Header.Set("X-Request-ID", c.GetHeader("X-Request-ID"))

		// Execute.
		resp, err := p.httpClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "backend service unavailable"})
			return
		}
		defer resp.Body.Close()

		// Copy response headers.
		for key, values := range resp.Header {
			for _, v := range values {
				c.Header(key, v)
			}
		}

		// Copy response body.
		c.Status(resp.StatusCode)
		io.Copy(c.Writer, resp.Body)
	}
}
