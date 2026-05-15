package ratelimit

import (
	"hash/fnv"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iicpc/api-gateway-go/internal/httpx"
	contractapi "github.com/iicpc/pkg/contracts/api"
)

const shardCount = 64

type clientEntry struct {
	tokens float64
	seenAt time.Time
}

type shard struct {
	mu      sync.Mutex
	clients map[string]*clientEntry
}

// RateLimiter implements a sharded token bucket limiter keyed by client IP.
type RateLimiter struct {
	maxPerMinute int
	refillPerSec float64
	shards       [shardCount]shard
}

// NewRateLimiter creates a rate limiter with the given max requests per minute.
func NewRateLimiter(maxPerMinute int) *RateLimiter {
	if maxPerMinute <= 0 {
		maxPerMinute = 600
	}

	rl := &RateLimiter{
		maxPerMinute: maxPerMinute,
		refillPerSec: float64(maxPerMinute) / 60.0,
	}
	for i := range rl.shards {
		rl.shards[i].clients = make(map[string]*clientEntry)
	}

	go rl.cleanupLoop()
	return rl
}

// Middleware returns a Gin middleware that enforces the rate limit.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		allowed, remaining, retryAfter := rl.allow(c.ClientIP(), time.Now())

		c.Header("X-RateLimit-Limit", strconv.Itoa(rl.maxPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds()+0.999)))
			httpx.WriteGinError(c, http.StatusTooManyRequests, contractapi.ErrorRateLimited, "rate limit exceeded")
			return
		}

		c.Next()
	}
}

func (rl *RateLimiter) allow(clientID string, now time.Time) (bool, int, time.Duration) {
	s := rl.shardFor(clientID)
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, exists := s.clients[clientID]
	if !exists {
		entry = &clientEntry{tokens: float64(rl.maxPerMinute), seenAt: now}
		s.clients[clientID] = entry
	}

	elapsed := now.Sub(entry.seenAt).Seconds()
	entry.tokens = minFloat(float64(rl.maxPerMinute), entry.tokens+elapsed*rl.refillPerSec)
	entry.seenAt = now

	if entry.tokens < 1 {
		wait := time.Duration((1 - entry.tokens) / rl.refillPerSec * float64(time.Second))
		return false, 0, wait
	}

	entry.tokens--
	return true, int(entry.tokens), 0
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for now := range ticker.C {
		for i := range rl.shards {
			rl.cleanupShard(&rl.shards[i], now)
		}
	}
}

func (rl *RateLimiter) cleanupShard(s *shard, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for clientID, entry := range s.clients {
		if now.Sub(entry.seenAt) > 10*time.Minute {
			delete(s.clients, clientID)
		}
	}
}

func (rl *RateLimiter) shardFor(clientID string) *shard {
	h := fnv.New32a()
	_, _ = h.Write([]byte(clientID))
	return &rl.shards[h.Sum32()%shardCount]
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
