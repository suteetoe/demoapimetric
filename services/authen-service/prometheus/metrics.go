package prometheus

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Counter metrics
var (
	// Login counters
	LoginCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_login_total",
			Help: "Total number of login attempts",
		},
	)

	// Registration counters
	RegisterCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_register_total",
			Help: "Total number of user registrations",
		},
	)

	// Tenant selection counter
	TenantSelectionCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_tenant_selection_total",
			Help: "Total number of tenant selections after login",
		},
	)

	// Tenant operation counter
	TenantOperationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_tenant_operations_total",
			Help: "Total number of tenant operations",
		},
		[]string{"operation"}, // operation can be "create", "access", "update", "delete", etc.
	)

	// HTTP request counter by endpoint and status
	HTTPRequestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_http_requests_total",
			Help: "Total number of HTTP requests by endpoint and status",
		},
		[]string{"endpoint", "method", "status"},
	)

	// Error counters
	AuthErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_errors_total",
			Help: "Total number of authentication errors",
		},
		[]string{"type"}, // type can be "login_failure", "invalid_token", "db_error" etc.
	)

	// Tenant-specific error counter
	TenantErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_tenant_errors_total",
			Help: "Total number of tenant-related errors",
		},
		[]string{"tenant_id", "error_type"}, // Track errors by tenant
	)

	// Auth operation counter
	AuthOperationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_operations_total",
			Help: "Total number of authentication operations",
		},
		[]string{"operation"}, // operation can be "profile_access", "profile_update", "password_change", etc.
	)
)

// Histogram metrics
var (
	// Request duration
	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "method", "status"},
	)

	// Database operation duration
	DBOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_db_operation_duration_seconds",
			Help:    "Duration of database operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"}, // operation can be "query", "insert", "update", "delete"
	)

	// Tenant operation duration
	TenantOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_tenant_operation_duration_seconds",
			Help:    "Duration of tenant operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "tenant_id"}, // Track performance by tenant
	)
)

// Gauge metrics
var (
	// Active tokens
	ActiveTokensGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "auth_active_tokens",
			Help: "Number of currently active authentication tokens",
		},
	)

	// System info
	InfoGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "auth_info",
			Help: "Information about the authentication service",
		},
		[]string{"version"},
	)

	// Active tenants
	ActiveTenantsGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "auth_active_tenants",
			Help: "Number of currently active tenants",
		},
	)

	// Users per tenant
	UsersPerTenantGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "auth_users_per_tenant",
			Help: "Number of users per tenant",
		},
		[]string{"tenant_id", "tenant_name"},
	)
)

func init() {
	// Register counters
	prometheus.MustRegister(LoginCounter)
	prometheus.MustRegister(RegisterCounter)
	prometheus.MustRegister(TenantSelectionCounter)
	prometheus.MustRegister(TenantOperationCounter)
	prometheus.MustRegister(HTTPRequestCounter)
	prometheus.MustRegister(AuthErrorCounter)
	prometheus.MustRegister(TenantErrorCounter)
	prometheus.MustRegister(AuthOperationCounter)

	// Register histograms
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(DBOperationDuration)
	prometheus.MustRegister(TenantOperationDuration)

	// Register gauges
	prometheus.MustRegister(ActiveTokensGauge)
	prometheus.MustRegister(InfoGauge)
	prometheus.MustRegister(ActiveTenantsGauge)
	prometheus.MustRegister(UsersPerTenantGauge)

	// Set initial service info
	InfoGauge.With(prometheus.Labels{"version": "1.0.0"}).Set(1)
}

// GetPrometheusHandler returns an HTTP handler for the Prometheus metrics
func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// DBMetricsMiddleware measures database operation durations
func TrackDBOperation(operation string) func(time.Time) {
	startTime := time.Now()
	return func(endTime time.Time) {
		duration := time.Since(startTime).Seconds()
		DBOperationDuration.With(prometheus.Labels{
			"operation": operation,
		}).Observe(duration)
	}
}

// TrackTenantOperation measures tenant operation durations
func TrackTenantOperation(operation string, tenantID uint) func(time.Time) {
	startTime := time.Now()
	return func(endTime time.Time) {
		duration := time.Since(startTime).Seconds()
		TenantOperationDuration.With(prometheus.Labels{
			"operation": operation,
			"tenant_id": strconv.FormatUint(uint64(tenantID), 10),
		}).Observe(duration)
	}
}

// MetricsMiddleware creates a middleware function that captures metrics for each request
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Execute the request handler
			err := next(c)

			// Record request duration
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(c.Response().Status)
			endpoint := c.Path()
			method := c.Request().Method

			// Record metrics
			RequestDuration.With(prometheus.Labels{
				"endpoint": endpoint,
				"method":   method,
				"status":   status,
			}).Observe(duration)

			HTTPRequestCounter.With(prometheus.Labels{
				"endpoint": endpoint,
				"method":   method,
				"status":   status,
			}).Inc()

			return err
		}
	}
}

// IncreaseActiveTokens increments the active tokens gauge
func IncreaseActiveTokens() {
	ActiveTokensGauge.Inc()
}

// DecreaseActiveTokens decrements the active tokens gauge
func DecreaseActiveTokens() {
	ActiveTokensGauge.Dec()
}

// RecordAuthError records an authentication error by type
func RecordAuthError(errorType string) {
	AuthErrorCounter.With(prometheus.Labels{"type": errorType}).Inc()
}

// RecordTenantError records a tenant-related error
func RecordTenantError(tenantID uint, errorType string) {
	TenantErrorCounter.With(prometheus.Labels{
		"tenant_id":  strconv.FormatUint(uint64(tenantID), 10),
		"error_type": errorType,
	}).Inc()
}

// RecordTenantOperation records a tenant operation
func RecordTenantOperation(operation string) {
	TenantOperationCounter.With(prometheus.Labels{"operation": operation}).Inc()
}

// RecordAuthOperation records an authentication operation by type
func RecordAuthOperation(operation string) {
	AuthOperationCounter.With(prometheus.Labels{"operation": operation}).Inc()
}

// UpdateActiveTenants updates the active tenants gauge
func UpdateActiveTenants(count int) {
	ActiveTenantsGauge.Set(float64(count))
}

// UpdateUsersPerTenant updates the users per tenant gauge
func UpdateUsersPerTenant(tenantID uint, tenantName string, count int) {
	UsersPerTenantGauge.With(prometheus.Labels{
		"tenant_id":   strconv.FormatUint(uint64(tenantID), 10),
		"tenant_name": tenantName,
	}).Set(float64(count))
}
