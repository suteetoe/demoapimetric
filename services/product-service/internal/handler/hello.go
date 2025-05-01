package handler

import (
    "github.com/labstack/echo/v4"
    "product-service/pkg/logger"
    "net/http"
)

func Hello(c echo.Context) error {
    log := logger.FromContext(c)
    log.Info("Hello from product-service")
    return c.JSON(http.StatusOK, echo.Map{"message": "hello from merchant"})
}
