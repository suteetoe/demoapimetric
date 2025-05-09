package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/suteetoe/gomicro/logger"
)

func Hello(c echo.Context) error {
	log := logger.FromEcho(c)
	log.Info("Hello from merchant-service")
	return c.JSON(http.StatusOK, echo.Map{"message": "hello from merchant"})
}
