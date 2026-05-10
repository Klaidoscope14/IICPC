package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/api-gateway-go/config"
	"github.com/iicpc/api-gateway-go/internal/proxy"
	"github.com/iicpc/api-gateway-go/internal/ratelimit"
	"github.com/iicpc/pkg/logging"
	"github.com/iicpc/pkg/metrics"
	"github.com/iicpc/pkg/middleware"
	"github.com/iicpc/pkg/server"
)

func main() {
	cfg := config.Load()
	logger := logging.NewLogger("api-gateway")

	// Create reverse proxies for backend services.
	submissionProxy, err := proxy.NewReverseProxy(cfg.SubmissionServiceURL)
	if err != nil {
		log.Fatalf("Failed to create submission proxy: %v", err)
	}

	orchestratorProxy, err := proxy.NewReverseProxy(cfg.OrchestratorURL)
	if err != nil {
		log.Fatalf("Failed to create orchestrator proxy: %v", err)
	}

	// Set up router with middleware.
	router := gin.Default()
	router.Use(middleware.CORS())
	router.Use(middleware.RequestLogger(logger))

	// Prometheus metrics.
	m := metrics.NewMetrics("api-gateway")
	router.Use(m.Middleware())
	router.GET("/metrics", metrics.Handler())

	// Rate limiting.
	rateLimiter := ratelimit.NewRateLimiter(cfg.RateLimitPerMinute)
	router.Use(rateLimiter.Middleware())

	// Request size limit (10MB).
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
		c.Next()
	})

	// Route: Submission service endpoints.
	router.Any("/api/v1/submissions", submissionProxy.Handler())
	router.Any("/api/v1/submissions/*path", submissionProxy.Handler())

	// Route: Orchestrator endpoints.
	router.Any("/api/v1/deployments", orchestratorProxy.Handler())
	router.Any("/api/v1/deployments/*path", orchestratorProxy.Handler())
	router.Any("/api/v1/benchmarks", orchestratorProxy.Handler())
	router.Any("/api/v1/benchmarks/*path", orchestratorProxy.Handler())
	router.Any("/api/v1/leaderboard", orchestratorProxy.Handler())

	// WebSocket passthrough.
	router.Any("/ws/*path", orchestratorProxy.Handler())

	// Health check.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "api-gateway",
			"backends": gin.H{
				"submission_service": cfg.SubmissionServiceURL,
				"orchestrator":      cfg.OrchestratorURL,
			},
		})
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
