package handler

import (
	"net/http"
	"product-service/pkg/logger"

	"github.com/labstack/echo/v4"
)

func Hello(c echo.Context) error {
	log := logger.FromContext(c)
	log.Info("Hello from product-service")
	return c.JSON(http.StatusOK, echo.Map{"message": "hello from product-service"})
}
