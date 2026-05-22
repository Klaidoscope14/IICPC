package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Checker function returns an error if a dependency is unhealthy.
type Checker func(ctx context.Context) error

// Dependency map of named dependencies and their checker functions.
type Dependencies map[string]Checker

// Handler returns a gin.HandlerFunc that concurrently checks all dependencies
// and responds with a JSON payload in standard format.
func Handler(serviceName string, deps Dependencies, timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		results := make(map[string]string, len(deps))
		var mu sync.Mutex
		var wg sync.WaitGroup

		overallStatus := http.StatusOK

		for name, check := range deps {
			wg.Add(1)
			go func(name string, check Checker) {
				defer wg.Done()
				
				statusStr := "healthy"
				err := check(ctx)
				
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					statusStr = "unhealthy"
					overallStatus = http.StatusServiceUnavailable
				}
				results[name] = statusStr
			}(name, check)
		}

		// Wait for all checks to complete or timeout
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			// Completed within timeout
		case <-ctx.Done():
			// Timeout reached before all finished
			overallStatus = http.StatusServiceUnavailable
			mu.Lock()
			for name := range deps {
				if _, ok := results[name]; !ok {
					results[name] = "timeout"
				}
			}
			mu.Unlock()
		}

		c.JSON(overallStatus, gin.H{
			"status":       mapStatus(overallStatus),
			"service":      serviceName,
			"dependencies": results,
		})
	}
}

func mapStatus(statusCode int) string {
	if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
		return "healthy"
	}
	return "unhealthy"
}
