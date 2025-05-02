package main

import (
	"os"
	"strconv"
	"time"

	"product-service/internal/handler"
	mid "product-service/internal/middleware"
	"product-service/pkg/database"
	applogger "product-service/pkg/logger"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Just log a warning, don't fail if .env file is not found
		// This allows the service to run in environments where env vars are set differently
		// such as production environments with proper environment configuration
		// The fallback values will be used in case env vars are not set
		// GCiampan 2023-03-15
		// log.Printf("Warning: .env file not found or error loading: %v\n", err)
	}

	// Initialize logger
	zapLogger, _ := zap.NewProduction()
	defer zapLogger.Sync()
	log := zapLogger.Sugar()

	// Initialize database
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "host=localhost user=postgres password=postgres dbname=product_db port=5432 sslmode=disable TimeZone=Asia/Bangkok"
	}

	dbConfig := database.DBConfig{
		DSN:             dbDSN,
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 30),
		ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", time.Hour),
	}

	err := database.Initialize(dbConfig)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(mid.RequestIDMiddleware)
	e.Use(applogger.Middleware(zapLogger))

	// Routes
	// Legacy route
	e.GET("/merchant/hello", handler.Hello)

	// Product API routes - Apply auth middleware to validate JWT and extract tenant ID
	productAPI := e.Group("/api/products", mid.AuthMiddleware)
	productAPI.GET("", handler.ListProducts)
	productAPI.GET("/:id", handler.GetProduct)
	productAPI.POST("", handler.CreateProduct)
	productAPI.PUT("/:id", handler.UpdateProduct)
	productAPI.DELETE("/:id", handler.DeleteProduct)

	// Category API routes - Apply auth middleware to validate JWT and extract tenant ID
	categoryAPI := e.Group("/api/categories", mid.AuthMiddleware)
	categoryAPI.GET("", handler.ListCategories)
	categoryAPI.GET("/:id", handler.GetCategory)
	categoryAPI.POST("", handler.CreateCategory)
	categoryAPI.PUT("/:id", handler.UpdateCategory)
	categoryAPI.DELETE("/:id", handler.DeleteCategory)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Infof("Starting product-service server on port %s", port)
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
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil && valueStr != "" {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if value, err := time.ParseDuration(valueStr); err == nil && valueStr != "" {
		return value
	}
	return defaultValue
}
