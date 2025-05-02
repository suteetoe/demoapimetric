package logger

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// contextKey is a private type for context keys to prevent collisions
type contextKey int

const (
	// loggerKey is the key used to store the logger in the context
	loggerKey contextKey = iota
)

// WithLogger returns a copy of the context with the logger included
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from the context
func FromContext(c echo.Context) *zap.Logger {
	// Try to extract from Echo context first
	if l, ok := c.Get("logger").(*zap.Logger); ok {
		return l
	}

	// Then try to extract from Go context
	if l, ok := c.Request().Context().Value(loggerKey).(*zap.Logger); ok {
		return l
	}

	// Fall back to default logger
	return zap.L()
}
