package main

import (
	"fmt"
	"merchant-service/internal/handler"
	"merchant-service/internal/model"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/suteetoe/gomicro/config"
	"github.com/suteetoe/gomicro/database"
	"github.com/suteetoe/gomicro/jwtutil"
	"github.com/suteetoe/gomicro/logger"
	"github.com/suteetoe/gomicro/metrics" // Import the new metrics package
	"github.com/suteetoe/gomicro/middleware"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Warning: .env file not found or error loading: %v\n", err)
	}

	// Load configuration using gomicro
	conf, err := config.Load("merchant")
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	err = logger.InitLogger(&logger.LogConfig{
		Level:       conf.Log.Level,
		Environment: conf.Server.Env,
		ServiceName: conf.ServiceName,
	})
	if err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}
	log := logger.GetLogger()

	// Initialize database connection using the DBConfig from the conf object directly
	_, err = database.InitDB(&conf.DB)
	if err != nil {
		log.Fatal("Failed to initialize database")
	}

	// Run migrations for merchant models
	if err := database.MigrateModels(&model.Merchant{}); err != nil {
		log.Fatal("Failed to migrate database models")
	}

	// Initialize JWT utility
	jwtConfig := &jwtutil.JWTConfig{
		SigningKey:      conf.JWT.SigningKey,
		ExpirationHours: conf.JWT.ExpirationHours,
	}
	jwt := jwtutil.NewJWTUtil(jwtConfig)

	// Initialize HTTP metrics
	httpMetrics := metrics.NewHTTPMetrics(conf.ServiceName)

	// Initialize Echo framework
	e := echo.New()

	// Apply middleware
	e.Use(middleware.RequestIDMiddleware())
	e.Use(logger.Middleware())
	e.Use(httpMetrics.Middleware()) // Use the centralized metrics middleware

	// Metrics endpoint
	e.GET("/metrics", echo.WrapHandler(metrics.GetPrometheusHandler()))

	// Public routes
	e.GET("/merchant/hello", handler.Hello) // Public endpoint, doesn't need auth

	// Secured routes - require authentication
	merchants := e.Group("/merchants")
	merchants.Use(middleware.JWTAuthMiddleware(jwt)) // Apply auth middleware to all merchant routes

	merchants.POST("", handler.CreateMerchant)
	merchants.GET("/:id", handler.GetMerchant)
	merchants.GET("", handler.ListMerchantsByOwner)

	// Start server
	log.Info("Starting merchant-service on port " + conf.Server.Port)
	e.Logger.Fatal(e.Start(":" + conf.Server.Port))
}
