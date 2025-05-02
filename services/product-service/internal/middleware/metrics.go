package middleware

import (
	"product-service/prometheus"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// MetricsMiddleware adds prometheus metrics to track HTTP requests
func MetricsMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Start timer for request duration
		start := time.Now()

		// Process request
		err := next(c)

		// Calculate request duration
		duration := time.Since(start).Seconds()

		// Get request details
		method := c.Request().Method
		path := c.Path()
		status := strconv.Itoa(c.Response().Status)

		// Record metrics
		prometheus.HttpRequestsTotal.WithLabelValues(method, path, status).Inc()
		prometheus.HttpRequestDuration.WithLabelValues(method, path, status).Observe(duration)

		return err
	}
}
