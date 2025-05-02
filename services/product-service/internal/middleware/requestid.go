package middleware

import (
	"product-service/pkg/logger"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Generate a unique request ID
		requestID := uuid.New().String()

		// Add it to the request headers
		c.Request().Header.Set("X-Request-ID", requestID)
		c.Response().Header().Set("X-Request-ID", requestID)

		// Add the request ID to the context
		c.Set("request_id", requestID)

		// Add request ID to logger context
		log := logger.GetLogger().With(zap.String("request_id", requestID))
		c.Set("logger", log)

		// Pass to the next middleware/handler
		return next(c)
	}
}
