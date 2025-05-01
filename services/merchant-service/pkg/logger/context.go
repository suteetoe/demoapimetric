package logger

import (
    "github.com/labstack/echo/v4"
    "go.uber.org/zap"
)

const RequestIDKey = "X-Request-ID"

func FromContext(c echo.Context) *zap.Logger {
    reqID, ok := c.Get(RequestIDKey).(string)
    if !ok {
        reqID = "unknown"
    }
    return GetLogger().With(zap.String("request_id", reqID))
}
