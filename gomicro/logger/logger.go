package logger

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogConfig holds logger configuration
type LogConfig struct {
	Level       string
	Environment string
	ServiceName string
}

var log *zap.Logger

// InitLogger initializes the logger with configuration
func InitLogger(config *LogConfig) error {
	// Configure logger based on configured log level
	var level zapcore.Level
	switch config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	var err error
	if config.Environment == "production" {
		// Production logger configuration
		prodConfig := zap.NewProductionConfig()
		prodConfig.Level = zap.NewAtomicLevelAt(level)
		prodConfig.EncoderConfig.TimeKey = "timestamp"
		prodConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		log, err = prodConfig.Build(zap.Fields(
			zap.String("service", config.ServiceName),
			zap.String("environment", config.Environment),
		))
	} else {
		// Development logger configuration with colors and human-friendly output
		devConfig := zap.NewDevelopmentConfig()
		devConfig.Level = zap.NewAtomicLevelAt(level)
		devConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		log, err = devConfig.Build(zap.Fields(
			zap.String("service", config.ServiceName),
			zap.String("environment", config.Environment),
		))
	}

	if err != nil {
		// Can't use the logger here, so using a panic
		return err
	}

	// Replace the global logger
	zap.ReplaceGlobals(log)
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	return log
}

// Middleware returns an Echo middleware that logs HTTP requests
func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Add request ID to context if available
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = c.Response().Header().Get("X-Request-ID")
			}

			// Set logger in context
			ctxLogger := log.With(zap.String("request_id", requestID))
			c.Set("logger", ctxLogger)

			// Process the request
			err := next(c)

			// Log after request is processed
			latency := time.Since(start)

			// Create structured log entry
			ctxLogger.Info("HTTP Request",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", latency),
				zap.String("ip", c.RealIP()),
			)

			return err
		}
	}
}
