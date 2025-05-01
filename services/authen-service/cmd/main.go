package main

import (
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	"auth-service/pkg/database"
	applogger "auth-service/pkg/logger"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found or error loading: %v\n", err)
	}

	// Initialize logger
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	log := zapLogger.Sugar()

	// Get database configuration from environment variables
	dbConfig := database.DBConfig{
		DSN:             getEnv("DB_DSN", "root:password@tcp(localhost:3306)/auth_service?charset=utf8mb4&parseTime=True&loc=Local"),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 1*time.Hour),
		LogLevel:        getEnvAsLogLevel("DB_LOG_LEVEL", logger.Info),
	}

	// Initialize database
	if err := database.Initialize(dbConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Echo framework
	e := echo.New()
	e.Use(middleware.RequestIDMiddleware)
	e.Use(applogger.Middleware(zapLogger))

	// Register routes
	e.POST("/auth/login", handler.Login)
	e.POST("/auth/register", handler.Register)
	e.POST("/auth/associate-merchant", handler.AssociateMerchant)
	e.GET("/metrics", handler.MetricsHandler)

	// Get server port from environment variable
	port := getEnv("SERVER_PORT", "8081")

	// Start server
	log.Infof("Starting server on :%s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsLogLevel(key string, defaultValue logger.LogLevel) logger.LogLevel {
	valueStr := getEnv(key, "")
	switch valueStr {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return defaultValue
	}
}
