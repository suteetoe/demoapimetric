package main

import (
	"time"

	"supplier-service/internal/handler"
	"supplier-service/internal/middleware"
	"supplier-service/pkg/config"
	"supplier-service/pkg/database"
	"supplier-service/pkg/jwtutil"
	"supplier-service/pkg/logger"
	"supplier-service/prometheus"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/suteetoe/gomicro/metrics" // Import the gomicro metrics package
	"go.uber.org/zap"
)

func main() {
	// Load configuration from .env file and environment variables
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger with config
	logger.InitLogger(cfg)
	log := logger.GetLogger()
	log.Info("Starting supplier service...", zap.String("environment", cfg.Server.Env))

	// Initialize JWT utilities
	jwtutil.Initialize(&cfg.JWT)
	log.Info("JWT utilities initialized")

	// Initialize Prometheus metrics
	prometheus.InitMetrics(cfg)
	log.Info("Prometheus metrics initialized")

	// Initialize HTTP metrics from gomicro
	httpMetrics := metrics.NewHTTPMetrics("supplier-service")
	log.Info("gomicro HTTP metrics initialized")

	// Initialize database and run migrations
	if err := database.InitDB(cfg); err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}
	log.Info("Database connection established and migrations completed", zap.String("db_host", cfg.DB.Host), zap.String("db_name", cfg.DB.DBName))

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())
	e.Use(middleware.RequestIDMiddleware)
	e.Use(httpMetrics.Middleware()) // Add gomicro metrics middleware

	// Request logging middleware
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Calculate duration
			duration := time.Since(start).Seconds()
			status := c.Response().Status

			// Log request details
			log := logger.FromContext(c)
			log.Info("HTTP Request",
				zap.String("method", c.Request().Method),
				zap.String("path", c.Request().URL.Path),
				zap.Int("status", status),
				zap.Float64("duration_s", duration),
				zap.String("ip", c.RealIP()),
			)

			// Update legacy Prometheus metrics for backward compatibility
			// (gomicro metrics middleware now handles the standardized metrics)
			prometheus.HttpRequestsTotal.WithLabelValues(
				c.Request().Method,
				c.Request().URL.Path,
				string(rune(status)),
			).Inc()

			prometheus.HttpRequestDuration.WithLabelValues(
				c.Request().Method,
				c.Request().URL.Path,
				string(rune(status)),
			).Observe(duration)

			return err
		}
	})

	// Routes
	// Public routes that don't require authentication
	e.GET("/", handler.Hello)
	e.GET("/health", handler.Hello)

	// Prometheus metrics endpoint
	e.GET("/metrics", echo.WrapHandler(metrics.GetPrometheusHandler())) // Use gomicro metrics handler

	// API routes that require authentication
	api := e.Group("/api")
	api.Use(middleware.AuthMiddleware)

	// Supplier endpoints with tenant context requirement
	suppliers := api.Group("/suppliers")
	suppliers.Use(middleware.RequireTenantContext)

	// Register supplier routes
	suppliers.POST("", handler.CreateSupplier)
	suppliers.GET("", handler.ListSuppliers)
	suppliers.GET("/:id", handler.GetSupplier)
	suppliers.PUT("/:id", handler.UpdateSupplier)
	suppliers.DELETE("/:id", handler.DeleteSupplier)

	// Start server
	port := cfg.Server.Port
	log.Info("Starting server", zap.String("port", port))
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
