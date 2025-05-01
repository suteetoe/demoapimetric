package main

import (
    "github.com/labstack/echo/v4"
    mid "product-service/internal/middleware"
    "product-service/internal/handler"
)

func main() {
    e := echo.New()
    e.Use(mid.RequestIDMiddleware)
    e.GET("/merchant/hello", handler.Hello)
    e.Logger.Fatal(e.Start(":8082"))
}
