package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// HealthCheck handles the health check endpoint
func HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"status":  "healthy",
		"service": "authen-service",
	})
}
