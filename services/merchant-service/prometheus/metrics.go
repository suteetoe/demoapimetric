package prometheus

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Existing counters
var CreateMerchantCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "merchant_create_total",
		Help: "Total number of merchant creations",
	},
)

var GetMerchantCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "merchant_get_total",
		Help: "Total number of merchant retrievals",
	},
)

var ListMerchantsCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "merchant_list_total",
		Help: "Total number of merchant listing requests",
	},
)

var RequestDurationHistogram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "merchant_http_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)

func init() {
	prometheus.MustRegister(CreateMerchantCounter)
	prometheus.MustRegister(GetMerchantCounter)
	prometheus.MustRegister(ListMerchantsCounter)
	prometheus.MustRegister(RequestDurationHistogram)
}

func GetPrometheusHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware is an Echo middleware function that records HTTP request metrics
func MetricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			// Record metrics after the request is processed
			method := c.Request().Method
			path := c.Path()

			// Record the request duration
			duration := time.Since(start).Seconds()
			RequestDurationHistogram.WithLabelValues(method, path).Observe(duration)

			return err
		}
	}
}
