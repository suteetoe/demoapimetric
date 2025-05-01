package main

import (
	"fmt"
	"merchant-service/internal/handler"
	mid "merchant-service/internal/middleware"
	"merchant-service/pkg/database"
	applogger "merchant-service/pkg/logger"
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
		DSN:             getEnv("DB_DSN", "postgresql://postgres.qbgyhktoqpnptoidzqcf:Bbutuwbb9BmvAh0J@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres"),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 1*time.Hour),
		LogLevel:        getEnvAsLogLevel("DB_LOG_LEVEL", logger.Info),
	}

	// Initialize database
	if err := database.Initialize(dbConfig); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	e := echo.New()
	e.Use(mid.RequestIDMiddleware)
	e.Use(applogger.Middleware(zapLogger))

	// Public routes
	e.GET("/merchant/hello", handler.Hello) // Public endpoint, doesn't need auth

	// Secured routes - require authentication
	merchants := e.Group("/merchants")
	merchants.Use(mid.AuthMiddleware) // Apply auth middleware to all merchant routes

	merchants.POST("", handler.CreateMerchant)
	merchants.GET("/:id", handler.GetMerchant)
	merchants.GET("", handler.ListMerchantsByOwner)

	// Merchant Users routes
	merchants.POST("/:merchant_id/users", handler.AddUserToMerchant)
	merchants.DELETE("/:merchant_id/users/:user_id", handler.RemoveUserFromMerchant)
	merchants.GET("/:merchant_id/users", handler.ListMerchantUsers)

	// User-specific merchant routes
	e.GET("/user/merchants", handler.GetUserMerchants, mid.AuthMiddleware)

	// Get server port from environment variable
	port := getEnv("SERVER_PORT", "8082")

	// Start server
	log.Infof("Starting server on :%s", port)
	e.Logger.Fatal(e.Start(":" + port))
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
