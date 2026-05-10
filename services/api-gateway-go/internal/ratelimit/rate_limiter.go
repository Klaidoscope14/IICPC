package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientEntry struct {
	count    int
	resetAt  time.Time
}

// RateLimiter implements a simple per-IP rate limiter using a fixed window.
type RateLimiter struct {
	maxPerMinute int
	mu           sync.Mutex
	clients      map[string]*clientEntry
}

// NewRateLimiter creates a rate limiter with the given max requests per minute.
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		maxPerMinute: maxPerMinute,
		clients:      make(map[string]*clientEntry),
	}

	// Background cleanup every 5 minutes.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Middleware returns a Gin middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		rl.mu.Lock()
		entry, exists := rl.clients[clientIP]
		now := time.Now()

		if !exists || now.After(entry.resetAt) {
			entry = &clientEntry{
				count:   0,
				resetAt: now.Add(1 * time.Minute),
			}
			rl.clients[clientIP] = entry
		}

		entry.count++
		remaining := rl.maxPerMinute - entry.count
		rl.mu.Unlock()

		// Set rate limit headers.
		c.Header("X-RateLimit-Limit", string(rune(rl.maxPerMinute)))
		c.Header("X-RateLimit-Remaining", string(rune(max(0, remaining))))

		if remaining < 0 {
			retryAfter := time.Until(entry.resetAt).Seconds()
			c.Header("Retry-After", string(rune(int(retryAfter))))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int(retryAfter),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, entry := range rl.clients {
		if now.After(entry.resetAt) {
			delete(rl.clients, ip)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
