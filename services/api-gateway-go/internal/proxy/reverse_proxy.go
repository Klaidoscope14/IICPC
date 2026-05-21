package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/api-gateway-go/internal/httpx"
	contractapi "github.com/iicpc/pkg/contracts/api"
)

const maxNormalizedErrorBody = 64 << 10

// ReverseProxy forwards requests to a backend service.
type ReverseProxy struct {
	targetURL   *url.URL
	serviceName string
	proxy       *httputil.ReverseProxy
}

type Option func(*ReverseProxy)

func WithServiceName(name string) Option {
	return func(p *ReverseProxy) {
		p.serviceName = name
	}
}

// NewReverseProxy creates a websocket-aware proxy that forwards to the backend URL.
func NewReverseProxy(target string, opts ...Option) (*ReverseProxy, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	p := &ReverseProxy{
		targetURL:   u,
		serviceName: u.Host,
	}
	for _, opt := range opts {
		opt(p)
	}

	rp := httputil.NewSingleHostReverseProxy(u)
	defaultDirector := rp.Director
	rp.Director = func(req *http.Request) {
		requestID := httpx.RequestID(req)
		originalHost := req.Host
		proto := forwardedProto(req)
		defaultDirector(req)
		req.Host = u.Host
		req.Header.Set("X-Forwarded-Host", originalHost)
		req.Header.Set("X-Forwarded-Proto", proto)
		if requestID != "" {
			req.Header.Set("X-Request-ID", requestID)
		}
	}
	rp.Transport = optimizedTransport()
	rp.FlushInterval = -1
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		httpx.WriteHTTPError(w, r, http.StatusBadGateway, contractapi.ErrorServiceUnavailable, p.serviceName+" unavailable")
	}
	rp.ModifyResponse = func(resp *http.Response) error {
		// Strip CORS headers set by upstream services to prevent
		// duplication with the gateway's own CORS middleware.
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Expose-Headers")
		return p.normalizeBackendError(resp)
	}
	p.proxy = rp

	return p, nil
}

// Handler returns a Gin handler that proxies the request to the backend.
func (p *ReverseProxy) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if requestID := httpx.RequestID(c.Request); requestID != "" {
			c.Request.Header.Set("X-Request-ID", requestID)
		}
		p.proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func (p *ReverseProxy) normalizeBackendError(resp *http.Response) error {
	if resp.StatusCode < http.StatusBadRequest || resp.StatusCode == http.StatusSwitchingProtocols {
		return nil
	}

	body, truncated, err := readLimited(resp.Body, maxNormalizedErrorBody)
	if err != nil {
		return nil
	}
	_ = resp.Body.Close()

	message := backendErrorMessage(body)
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	payload := contractapi.NewErrorResponse(
		httpx.ErrorCodeForStatus(resp.StatusCode),
		message,
		httpx.RequestID(resp.Request),
	)
	encoded, err := json.Marshal(payload)
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		return nil
	}
	if truncated {
		resp.Header.Set("X-Gateway-Error-Body-Truncated", "true")
	}

	resp.Body = io.NopCloser(bytes.NewReader(encoded))
	resp.ContentLength = int64(len(encoded))
	resp.Header.Set("Content-Type", "application/json")
	resp.Header.Set("X-Content-Type-Options", "nosniff")
	resp.Header.Del("Content-Encoding")
	resp.Header.Del("Content-Length")
	return nil
}

func optimizedTransport() http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
}

func forwardedProto(req *http.Request) string {
	if req.TLS != nil {
		return "https"
	}
	return "http"
}

func readLimited(r io.Reader, limit int64) ([]byte, bool, error) {
	limited := &io.LimitedReader{R: r, N: limit + 1}
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > limit {
		return body[:limit], true, nil
	}
	return body, false, nil
}

func backendErrorMessage(body []byte) string {
	var payload struct {
		Error interface{} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		switch value := payload.Error.(type) {
		case string:
			return value
		case map[string]interface{}:
			if message, ok := value["message"].(string); ok {
				return message
			}
		}
	}

	message := strings.TrimSpace(string(body))
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}
