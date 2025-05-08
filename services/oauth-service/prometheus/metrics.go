package prometheus

import (
	"oauth-service/pkg/config"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Client metrics
	ClientRegistrationCounter prometheus.Counter
	ActiveClientsGauge        prometheus.Gauge

	// Token metrics
	TokenRequestCounter        *prometheus.CounterVec
	TokensIssuedCounter        *prometheus.CounterVec
	TokensRevokedCounter       *prometheus.CounterVec
	TokensRefreshedCounter     prometheus.Counter
	InvalidTokenRequestCounter *prometheus.CounterVec
	ActiveTokensGauge          prometheus.Gauge

	// Database operation metrics
	DBOperationHistogram *prometheus.HistogramVec

	// Request metrics
	RequestDurationHistogram *prometheus.HistogramVec
	APIRequestCounter        *prometheus.CounterVec
	APIErrorCounter          *prometheus.CounterVec

	// Namespace prefix for metrics
	namespace string
)

// InitMetrics initializes all Prometheus metrics
func InitMetrics(cfg *config.Config) {
	namespace = cfg.Metrics.Prefix

	// Client metrics
	ClientRegistrationCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "client_registration_total",
		Help:      "Total number of client registrations",
	})

	ActiveClientsGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "active_clients",
		Help:      "Number of currently active clients",
	})

	// Token metrics
	TokenRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "token_request_total",
			Help:      "Total number of token requests",
		},
		[]string{"grant_type"},
	)

	TokensIssuedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tokens_issued_total",
			Help:      "Total number of tokens issued",
		},
		[]string{"grant_type", "token_type"},
	)

	TokensRevokedCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tokens_revoked_total",
			Help:      "Total number of tokens revoked",
		},
		[]string{"token_type", "reason"},
	)

	TokensRefreshedCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "tokens_refreshed_total",
		Help:      "Total number of tokens refreshed using refresh tokens",
	})

	InvalidTokenRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "invalid_token_request_total",
			Help:      "Total number of invalid token requests",
		},
		[]string{"error_type"},
	)

	ActiveTokensGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "active_tokens",
		Help:      "Number of currently active tokens",
	})

	// Database operation metrics
	DBOperationHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "db_operation_duration_seconds",
			Help:      "Duration of database operations in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// Request metrics
	RequestDurationHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	APIRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_requests_total",
			Help:      "Total number of API requests",
		},
		[]string{"method", "path"},
	)

	APIErrorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_errors_total",
			Help:      "Total number of API errors",
		},
		[]string{"method", "path", "status"},
	)
}

// MetricsMiddleware tracks request metrics
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Track API request count
			APIRequestCounter.With(prometheus.Labels{
				"method": c.Request().Method,
				"path":   c.Path(),
			}).Inc()

			// Process the request
			err := next(c)

			// Track request duration
			duration := time.Since(start).Seconds()
			status := c.Response().Status
			RequestDurationHistogram.With(prometheus.Labels{
				"method": c.Request().Method,
				"path":   c.Path(),
				"status": string(status),
			}).Observe(duration)

			// Track errors
			if status >= 400 {
				APIErrorCounter.With(prometheus.Labels{
					"method": c.Request().Method,
					"path":   c.Path(),
					"status": string(status),
				}).Inc()
			}

			return err
		}
	}
}

// HandlerFunc returns a HTTP handler for metrics endpoint
func HandlerFunc() echo.HandlerFunc {
	return echo.WrapHandler(promhttp.Handler())
}

// TrackDBOperation returns a function that tracks database operation duration
func TrackDBOperation(operation string) func(time.Time) {
	return func(startTime time.Time) {
		duration := time.Since(startTime).Seconds()
		DBOperationHistogram.With(prometheus.Labels{
			"operation": operation,
		}).Observe(duration)
	}
}

// RecordTokenIssued increments the tokens issued counter
func RecordTokenIssued(grantType, tokenType string) {
	TokensIssuedCounter.With(prometheus.Labels{
		"grant_type": grantType,
		"token_type": tokenType,
	}).Inc()
}

// RecordTokenRevoked increments the tokens revoked counter
func RecordTokenRevoked(tokenType, reason string) {
	TokensRevokedCounter.With(prometheus.Labels{
		"token_type": tokenType,
		"reason":     reason,
	}).Inc()
}
