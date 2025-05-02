package prometheus

import (
	"product-service/pkg/config"
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

	// Product metrics
	ProductOperationsCounter prometheus.CounterVec

	// Category metrics
	CategoryOperationsCounter prometheus.CounterVec

	// Inventory metrics
	ProductInventoryGauge prometheus.GaugeVec

	// Product popularity metrics
	ProductViewsCounter prometheus.CounterVec
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

	// Product metrics
	ProductOperationsCounter = *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_operations_total",
			Help: "Total number of product operations",
		},
		[]string{"operation"},
	)

	// Category metrics
	CategoryOperationsCounter = *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_category_operations_total",
			Help: "Total number of category operations",
		},
		[]string{"operation"},
	)

	// Product inventory metrics
	ProductInventoryGauge = *promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: prefix + "_product_inventory",
			Help: "Current inventory level for products",
		},
		[]string{"product_id", "product_name", "category"},
	)

	// Product popularity metrics
	ProductViewsCounter = *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: prefix + "_product_views_total",
			Help: "Total number of product views",
		},
		[]string{"product_id", "category"},
	)
}

// TrackDBOperation returns a function that records the duration of a database operation
func TrackDBOperation(operationType string) func(startTime time.Time) {
	return func(startTime time.Time) {
		duration := time.Since(startTime).Seconds()
		DbOperationDuration.WithLabelValues(operationType).Observe(duration)
	}
}

// RecordProductOperation increments the counter for product operations
func RecordProductOperation(operation string) {
	ProductOperationsCounter.WithLabelValues(operation).Inc()
}

// RecordCategoryOperation increments the counter for category operations
func RecordCategoryOperation(operation string) {
	CategoryOperationsCounter.WithLabelValues(operation).Inc()
}

// UpdateProductInventory updates the gauge for product inventory
func UpdateProductInventory(productID string, productName string, category string, count float64) {
	ProductInventoryGauge.WithLabelValues(productID, productName, category).Set(count)
}

// RecordProductView increments the counter for product views
func RecordProductView(productID string, category string) {
	ProductViewsCounter.WithLabelValues(productID, category).Inc()
}
