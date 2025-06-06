package middleware

import (
	"auth-service/pkg/logger"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check if request already has an ID from upstream services
		requestID := c.Request().Header.Get("X-Request-ID")
		if requestID == "" {
			// Generate a unique request ID if not present
			requestID = uuid.New().String()
		}

		// Add the request ID to the context
		c.Set("request_id", requestID)

		// Add request ID to response headers
		c.Response().Header().Set("X-Request-ID", requestID)

		// Add request ID to the logger
		log := logger.FromContext(c).With(zap.String("request_id", requestID))
		c.Set("logger", log)

		// Call the next handler
		return next(c)
	}
}
