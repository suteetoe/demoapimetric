package middleware

import (
    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
)

const RequestIDKey = "X-Request-ID"

func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
    return func(c echo.Context) error {
        requestID := c.Request().Header.Get(RequestIDKey)
        if requestID == "" {
            requestID = uuid.New().String()
        }
        c.Set(RequestIDKey, requestID)
        c.Response().Header().Set(RequestIDKey, requestID)
        return next(c)
    }
}
