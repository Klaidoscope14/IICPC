package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/api-gateway-go/config"
	"github.com/iicpc/api-gateway-go/internal/httpx"
	gatewaymiddleware "github.com/iicpc/api-gateway-go/internal/middleware"
	"github.com/iicpc/api-gateway-go/internal/proxy"
	"github.com/iicpc/api-gateway-go/internal/ratelimit"
	contractapi "github.com/iicpc/pkg/contracts/api"
	"github.com/iicpc/pkg/logging"
	"github.com/iicpc/pkg/metrics"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/pkg/server"
)

func main() {
	cfg := config.Load()
	logger := logging.NewLogger("api-gateway")

	// Create reverse proxies for backend services.
	submissionProxy, err := proxy.NewReverseProxy(cfg.SubmissionServiceURL, proxy.WithServiceName("submission-service"))
	if err != nil {
		log.Fatalf("Failed to create submission proxy: %v", err)
	}

	validationProxy, err := proxy.NewReverseProxy(cfg.ValidationServiceURL, proxy.WithServiceName("validation-service"))
	if err != nil {
		log.Fatalf("Failed to create validation proxy: %v", err)
	}

	orchestratorProxy, err := proxy.NewReverseProxy(cfg.OrchestratorURL, proxy.WithServiceName("benchmark-orchestrator"))
	if err != nil {
		log.Fatalf("Failed to create orchestrator proxy: %v", err)
	}

	// Set up router with middleware.
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(gin.Recovery())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger(logger))

	// Prometheus metrics.
	m := metrics.NewMetrics("api-gateway")
	router.Use(m.Middleware())
	router.GET("/metrics", metrics.Handler())

	// Rate limiting.
	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitPerMinute)
	router.Use(rateLimiter.Middleware())
	router.Use(gatewaymiddleware.OptionalBearerAuth(cfg.AuthToken))
	router.Use(gatewaymiddleware.RequestValidator(cfg.MaxBodyBytes))

	// Route: Submission service endpoints.
	v1 := router.Group("/api/v1")
	{
		v1.Any("/submissions", submissionProxy.Handler())
		v1.Any("/submissions/*path", submissionProxy.Handler())

		v1.Any("/validations", validationProxy.Handler())
		v1.Any("/validations/*path", validationProxy.Handler())

		// Route: Orchestrator endpoints.
		v1.Any("/deployments", orchestratorProxy.Handler())
		v1.Any("/deployments/*path", orchestratorProxy.Handler())
		v1.Any("/benchmarks", orchestratorProxy.Handler())
		v1.Any("/benchmarks/*path", orchestratorProxy.Handler())
		v1.Any("/leaderboard", orchestratorProxy.Handler())
	}

	// WebSocket passthrough.
	router.Any("/ws/*path", orchestratorProxy.Handler())

	// Health check.
	router.GET("/health", func(c *gin.Context) {
		checks := checkBackends(c.Request.Context(), map[string]string{
			"submission-service":     cfg.SubmissionServiceURL,
			"validation-service":     cfg.ValidationServiceURL,
			"benchmark-orchestrator": cfg.OrchestratorURL,
		})
		status := "healthy"
		httpStatus := http.StatusOK
		for _, checkStatus := range checks {
			if checkStatus != "ok" {
				status = "degraded"
				httpStatus = http.StatusServiceUnavailable
				break
			}
		}
		c.JSON(httpStatus, contractapi.HealthResponse{
			Status:    status,
			Service:   "api-gateway",
			Version:   "v1",
			Checks:    checks,
			Timestamp: time.Now().UTC(),
		})
	})

	router.NoRoute(func(c *gin.Context) {
		httpx.WriteGinError(c, http.StatusNotFound, contractapi.ErrorNotFound, "route not found")
	})
	router.NoMethod(func(c *gin.Context) {
		httpx.WriteGinError(c, http.StatusMethodNotAllowed, contractapi.ErrorInvalidInput, "method not allowed")
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("API Gateway listening on :%s\n", cfg.Port)
	server.RunGracefully(srv, "api-gateway")
}

func checkBackends(parent context.Context, backends map[string]string) map[string]string {
	ctx, cancel := context.WithTimeout(parent, 1200*time.Millisecond)
	defer cancel()

	type result struct {
		name   string
		status string
	}

	results := make(chan result, len(backends))
	var wg sync.WaitGroup
	for name, baseURL := range backends {
		wg.Add(1)
		go func(name, baseURL string) {
			defer wg.Done()
			results <- result{name: name, status: backendHealth(ctx, baseURL)}
		}(name, baseURL)
	}
	wg.Wait()
	close(results)

	checks := make(map[string]string, len(backends))
	for res := range results {
		checks[res.name] = res.status
	}
	return checks
}

func backendHealth(ctx context.Context, baseURL string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		return "invalid_url"
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Sprintf("http_%d", resp.StatusCode)
	}
	return "ok"
}
