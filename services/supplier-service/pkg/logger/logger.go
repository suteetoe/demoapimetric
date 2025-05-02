package logger

import (
	"supplier-service/pkg/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// InitLogger initializes the logger with configuration
func InitLogger(config *config.Config) {
	// Get logger environment from config
	env := config.Server.Env
	logLevel := config.Log.Level

	// Configure logger based on configured log level
	var level zapcore.Level
	switch logLevel {
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
	if env == "production" {
		// Production logger configuration
		prodConfig := zap.NewProductionConfig()
		prodConfig.Level = zap.NewAtomicLevelAt(level)
		prodConfig.EncoderConfig.TimeKey = "timestamp"
		prodConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

		log, err = prodConfig.Build(zap.Fields(
			config.LogConfig()...,
		))
	} else {
		// Development logger configuration with colors and human-friendly output
		devConfig := zap.NewDevelopmentConfig()
		devConfig.Level = zap.NewAtomicLevelAt(level)
		devConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

		log, err = devConfig.Build(zap.Fields(
			config.LogConfig()...,
		))
	}

	if err != nil {
		// Can't use the logger here, so using a panic
		panic("failed to initialize logger: " + err.Error())
	}

	// Replace the global logger
	zap.ReplaceGlobals(log)
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	return log
}
