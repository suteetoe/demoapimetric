package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Hello is a simple handler that returns a welcome message
// Used for health check and root endpoints
func Hello(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Supplier Service API is running",
		"version": "1.0.0",
	})
}
