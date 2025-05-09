package middleware

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/suteetoe/gomicro/logger"
	"go.uber.org/zap"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
				c.Request().Header.Set("X-Request-ID", requestID)
			}

			// Add request ID to response header
			c.Response().Header().Set("X-Request-ID", requestID)

			// Update logger context with request ID
			ctxLogger := logger.GetLogger().With(zap.String("request_id", requestID))
			c.Set("logger", ctxLogger)

			return next(c)
		}
	}
}
