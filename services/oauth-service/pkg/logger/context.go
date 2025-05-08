package logger

import (
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const RequestIDKey = "X-Request-ID"

// FromContext retrieves the logger from echo.Context with the request ID
func FromContext(c echo.Context) *zap.Logger {
	// Try to get the logger from context first
	if logger, ok := c.Get("logger").(*zap.Logger); ok {
		return logger
	}

	// Otherwise, get the global logger and add request ID
	requestID, ok := c.Get(RequestIDKey).(string)
	if !ok {
		requestID = c.Request().Header.Get(RequestIDKey)
		if requestID == "" {
			requestID = "unknown"
		}
	}

	return GetLogger().With(zap.String("request_id", requestID))
}
