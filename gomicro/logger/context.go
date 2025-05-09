package logger

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type contextKey string

const loggerKey contextKey = "logger"

// FromContext retrieves the logger from the context
func FromContext(ctx context.Context) *zap.Logger {
	logger, ok := ctx.Value(loggerKey).(*zap.Logger)
	if !ok {
		return GetLogger()
	}
	return logger
}

// WithContext adds the logger to the context
func WithContext(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromEcho retrieves the logger from the Echo context
func FromEcho(c echo.Context) *zap.Logger {
	logger, ok := c.Get("logger").(*zap.Logger)
	if !ok {
		return GetLogger()
	}
	return logger
}
