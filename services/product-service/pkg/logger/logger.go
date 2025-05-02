package logger

import (
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var (
	once     sync.Once
	instance *zap.Logger
)

func New() *zap.SugaredLogger {
	return GetLogger().Sugar()
}

func GetLogger() *zap.Logger {
	once.Do(func() {
		cfg := zap.NewProductionConfig()
		cfg.OutputPaths = []string{"stdout"}
		logger, err := cfg.Build()
		if err != nil {
			panic(err)
		}
		instance = logger
	})
	return instance
}

// Middleware returns an Echo middleware that logs HTTP requests
func Middleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Add request ID to context if available
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = c.Response().Header().Get("X-Request-ID")
			}

			// Process the request
			err := next(c)

			// Log after request is processed
			latency := time.Since(start)

			// Create structured log entry
			logger.Info("HTTP Request",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", latency),
				zap.String("request_id", requestID),
				zap.String("ip", c.RealIP()),
			)

			return err
		}
	}
}
