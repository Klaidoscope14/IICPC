package metrics

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metric collectors for an HTTP service.
type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	ActiveRequests  prometheus.Gauge
}

// NewMetrics creates and registers Prometheus metrics for the given service.
func NewMetrics(serviceName string) *Metrics {
	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests.",
			ConstLabels: prometheus.Labels{"service": serviceName},
		},
		[]string{"method", "path", "status"},
	)
	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:        "http_request_duration_seconds",
			Help:        "HTTP request duration in seconds.",
			ConstLabels: prometheus.Labels{"service": serviceName},
			Buckets:     prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
	activeRequests := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name:        "http_active_requests",
			Help:        "Number of active HTTP requests.",
			ConstLabels: prometheus.Labels{"service": serviceName},
		},
	)

	m := &Metrics{
		RequestsTotal: registerOrReuse(
			requestsTotal,
			"counter",
		),
		RequestDuration: registerOrReuse(
			requestDuration,
			"histogram",
		),
		ActiveRequests: registerOrReuse(
			activeRequests,
			"gauge",
		),
	}
	return m
}

func registerOrReuse[T prometheus.Collector](collector T, kind string) T {
	if err := prometheus.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			existing, ok := alreadyRegistered.ExistingCollector.(T)
			if ok {
				return existing
			}
		}
		panic(fmt.Sprintf("failed to register prometheus %s: %v", kind, err))
	}
	return collector
}

// Middleware returns a Gin middleware that records Prometheus metrics per request.
func (m *Metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath() // Use route pattern, not actual path (avoids cardinality explosion).
		if path == "" {
			path = "unknown"
		}

		m.ActiveRequests.Inc()
		defer m.ActiveRequests.Dec()

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		duration := time.Since(start).Seconds()

		m.RequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		m.RequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}

// Handler returns the Prometheus HTTP handler for the /metrics endpoint.
func Handler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
