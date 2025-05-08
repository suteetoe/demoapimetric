package logger

import (
	"oauth-service/pkg/config"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// InitLogger initializes the global logger
func InitLogger(cfg *config.Config) {
	// Configure logger based on environment
	var logConfig zap.Config

	if cfg.Server.Env == "production" {
		// Production mode: structured JSON logs
		logConfig = zap.NewProductionConfig()
	} else {
		// Development mode: colorful, human-readable logs
		logConfig = zap.NewDevelopmentConfig()
		logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level from config
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Log.Level)); err != nil {
		// Default to info level if invalid
		level = zapcore.InfoLevel
	}
	logConfig.Level.SetLevel(level)

	// Build the logger
	var err error
	log, err = logConfig.Build()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	log.Info("Logger initialized", zap.String("level", level.String()))
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if log == nil {
		// Fallback if not initialized
		var err error
		log, err = zap.NewProduction()
		if err != nil {
			// This should never happen, but just in case
			panic("Failed to create fallback logger: " + err.Error())
		}
	}
	return log
}

// Middleware returns an Echo middleware that logs HTTP requests
func Middleware(logger *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Get request ID from context if available
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = c.Response().Header().Get("X-Request-ID")
			}

			// Set logger in context
			ctxLogger := logger.With(zap.String("request_id", requestID))
			c.Set("logger", ctxLogger)

			// Process the request
			err := next(c)

			// Log after request is processed
			latency := time.Since(start)

			// Create structured log entry
			fields := []zapcore.Field{
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.Int("status", c.Response().Status),
				zap.Duration("latency", latency),
				zap.String("ip", c.RealIP()),
				zap.String("user_agent", c.Request().UserAgent()),
			}

			// Add error if present
			if err != nil {
				fields = append(fields, zap.Error(err))
				ctxLogger.Error("HTTP request failed", fields...)
			} else {
				ctxLogger.Info("HTTP request completed", fields...)
			}

			return err
		}
	}
}
