package prometheus

import (
	"supplier-service/pkg/config"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	HttpRequestsTotal   prometheus.CounterVec
	HttpRequestDuration prometheus.HistogramVec

	// Authentication metrics
	AuthAttemptsCounter prometheus.Counter
	AuthSuccessCounter  prometheus.Counter
	AuthErrorsCounter   prometheus.Counter

	// Tenant context metrics
	TenantContextMissingCounter prometheus.Counter

	// Database operation metrics
	DbOperationDuration prometheus.HistogramVec

	// Supplier metrics
	SupplierOperationsCounter prometheus.CounterVec

	// Tenant specific metrics
	SuppliersPerTenantGauge prometheus.GaugeVec

	// Active tenants using the supplier service
	ActiveTenantsGauge prometheus.Gauge
)

// InitMetrics initializes Prometheus metrics with configuration
func InitMetrics(config *config.Config) {
	// Use metric prefix from configuration
	prefix := config.Metrics.Prefix

	// HTTP request metrics
	HttpRequestsTotal = *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP request duration
	HttpRequestDuration = *promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// Authentication metrics
	AuthAttemptsCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: prefix + "_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
	)

	AuthSuccessCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: prefix + "_auth_success_total",
			Help: "Total number of successful authentications",
		},
	)

	AuthErrorsCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: prefix + "_auth_errors_total",
			Help: "Total number of authentication errors",
		},
	)

	// Tenant context metrics
	TenantContextMissingCounter = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: prefix + "_tenant_context_missing_total",
			Help: "Total number of requests without tenant context",
		},
	)

	// Database operation metrics
	DbOperationDuration = *promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    prefix + "_db_operation_duration_seconds",
			Help:    "Duration of database operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation_type"},
	)

	// Supplier metrics
	SupplierOperationsCounter = *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_operations_total",
			Help: "Total number of supplier operations",
		},
		[]string{"operation"},
	)

	// Tenant specific metrics
	SuppliersPerTenantGauge = *promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_suppliers_per_tenant",
			Help: "Number of suppliers per tenant",
		},
		[]string{"tenant_id", "tenant_name"},
	)

	// Active tenants using the supplier service
	ActiveTenantsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: prefix + "_active_tenants",
			Help: "Number of active tenants using the supplier service",
		},
	)
}

// TrackDBOperation returns a function that records the duration of a database operation
func TrackDBOperation(operationType string) func(startTime time.Time) {
	return func(startTime time.Time) {
		duration := time.Since(startTime).Seconds()
		DbOperationDuration.WithLabelValues(operationType).Observe(duration)
	}
}

// RecordSupplierOperation increments the counter for supplier operations
func RecordSupplierOperation(operation string) {
	SupplierOperationsCounter.WithLabelValues(operation).Inc()
}

// UpdateSuppliersPerTenant updates the gauge for suppliers per tenant
func UpdateSuppliersPerTenant(tenantID uint, tenantName string, count int) {
	SuppliersPerTenantGauge.WithLabelValues(
		string(rune(tenantID)), // Convert uint to string
		tenantName,
	).Set(float64(count))
}

// UpdateActiveTenants updates the active tenants gauge
func UpdateActiveTenants(count int) {
	ActiveTenantsGauge.Set(float64(count))
}
