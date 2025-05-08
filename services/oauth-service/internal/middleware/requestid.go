package middleware

import (
	"oauth-service/pkg/logger"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Check if request already has a request ID
		requestID := c.Request().Header.Get(logger.RequestIDKey)

		// If not, generate a new one
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to request and response headers
		c.Request().Header.Set(logger.RequestIDKey, requestID)
		c.Response().Header().Set(logger.RequestIDKey, requestID)

		// Store in context for internal use
		c.Set(logger.RequestIDKey, requestID)

		// Call the next handler
		return next(c)
	}
}
