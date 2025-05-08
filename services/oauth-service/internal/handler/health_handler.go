package handler

import (
	"net/http"
	"oauth-service/pkg/database"
	"oauth-service/pkg/logger"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// HealthCheck handles the health check endpoint
func HealthCheck(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Health check requested")

	// Basic response
	response := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}

	// Check database connection if requested
	if c.QueryParam("check") == "db" {
		sqlDB, err := database.GetDB().DB()
		if err != nil {
			log.Error("Database connection error", zap.Error(err))
			response["status"] = "error"
			response["db_status"] = "error"
			response["db_error"] = "Failed to get database connection"
			return c.JSON(http.StatusInternalServerError, response)
		}

		// Ping database to check connection
		if err := sqlDB.Ping(); err != nil {
			log.Error("Database ping error", zap.Error(err))
			response["status"] = "error"
			response["db_status"] = "error"
			response["db_error"] = "Failed to ping database"
			return c.JSON(http.StatusInternalServerError, response)
		}

		// Database is healthy
		response["db_status"] = "ok"
	}

	return c.JSON(http.StatusOK, response)
}

// Hello returns a simple welcome message
func Hello(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Welcome to OAuth2 Service API",
		"version": "1.0.0",
	})
}
