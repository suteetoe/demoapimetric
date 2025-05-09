package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// RequestCounter counts all HTTP requests with labels
	RequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"service", "method", "path", "status"},
	)

	// RequestDurationHistogram records request duration in seconds
	RequestDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "path", "status"},
	)

	// Status code category counters
	StatusOkCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_status_2xx_total",
			Help: "Total number of 2xx (success) responses",
		},
		[]string{"service"},
	)

	StatusClientErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_status_4xx_total",
			Help: "Total number of 4xx (client error) responses",
		},
		[]string{"service"},
	)

	StatusServerErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_status_5xx_total",
			Help: "Total number of 5xx (server error) responses",
		},
		[]string{"service"},
	)

	// StatusCodeCategoryCounter with detailed labels
	StatusCodeCategoryCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_status_category_total",
			Help: "Total number of responses by status category (2xx, 4xx, 5xx)",
		},
		[]string{"service", "category", "method", "path"},
	)
)

// HTTPMetrics holds configuration and state for HTTP metrics collection
type HTTPMetrics struct {
	ServiceName string
	initialized bool
}

// NewHTTPMetrics creates a new HTTP metrics collector for a specific service
func NewHTTPMetrics(serviceName string) *HTTPMetrics {
	m := &HTTPMetrics{
		ServiceName: serviceName,
	}
	m.register()
	return m
}

// register registers the prometheus metrics if they haven't been registered already
func (m *HTTPMetrics) register() {
	if !m.initialized {
		prometheus.MustRegister(RequestCounter)
		prometheus.MustRegister(RequestDurationHistogram)
		prometheus.MustRegister(StatusOkCounter)
		prometheus.MustRegister(StatusClientErrorCounter)
		prometheus.MustRegister(StatusServerErrorCounter)
		prometheus.MustRegister(StatusCodeCategoryCounter)
		m.initialized = true
	}
}

// incrementStatusCounter increments the appropriate status counter based on the HTTP status code
func (m *HTTPMetrics) incrementStatusCounter(status int, method, path string) {
	category := ""

	if status >= 200 && status < 300 {
		StatusOkCounter.WithLabelValues(m.ServiceName).Inc()
		category = "2xx"
	} else if status >= 400 && status < 500 {
		StatusClientErrorCounter.WithLabelValues(m.ServiceName).Inc()
		category = "4xx"
	} else if status >= 500 && status < 600 {
		StatusServerErrorCounter.WithLabelValues(m.ServiceName).Inc()
		category = "5xx"
	}

	if category != "" {
		StatusCodeCategoryCounter.WithLabelValues(m.ServiceName, category, method, path).Inc()
	}
}

// Middleware creates an Echo middleware function that records HTTP request metrics
func (m *HTTPMetrics) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			// Record metrics after the request is processed
			status := c.Response().Status
			method := c.Request().Method
			path := c.Path()
			statusStr := strconv.Itoa(status)

			// Increment the request counter
			RequestCounter.WithLabelValues(m.ServiceName, method, path, statusStr).Inc()

			// Increment status category counters
			m.incrementStatusCounter(status, method, path)

			// Record the request duration
			duration := time.Since(start).Seconds()
			RequestDurationHistogram.WithLabelValues(m.ServiceName, method, path, statusStr).Observe(duration)

			return err
		}
	}
}

// GetPrometheusHandler returns an HTTP handler for exposing Prometheus metrics
func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}
