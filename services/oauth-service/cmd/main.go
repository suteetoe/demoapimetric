package main

import (
	"oauth-service/internal/handler"
	"oauth-service/internal/middleware"
	"oauth-service/pkg/config"
	"oauth-service/pkg/database"
	"oauth-service/pkg/logger"
	"oauth-service/prometheus"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	log.Info("Starting OAuth service...", zap.String("environment", cfg.Server.Env))

	// Initialize database (now includes migrations automatically)
	if err := database.InitDB(cfg); err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}
	log.Info("Database connection established and migrations completed")

	// Initialize token handler with configuration
	handler.InitTokenHandler(cfg)

	// Initialize Prometheus metrics
	prometheus.InitMetrics(cfg)
	log.Info("Prometheus metrics initialized")

	// Initialize Echo framework
	e := echo.New()

	// Apply global middleware - order matters
	e.Use(echomiddleware.Recover()) // Add recovery middleware
	e.Use(echomiddleware.CORS())    // Add CORS middleware
	e.Use(middleware.RequestIDMiddleware)
	e.Use(logger.Middleware(log))
	e.Use(prometheus.MetricsMiddleware())

	// Public routes - no authentication required
	e.GET("/", handler.Hello)
	e.GET("/health", handler.HealthCheck)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// OAuth2 routes
	oauth := e.Group("/oauth")

	// Client registration and management
	clients := oauth.Group("/clients")
	clients.POST("", handler.RegisterClient)
	clients.GET("/:id", handler.GetClient, middleware.ClientAuthMiddleware)

	// Token endpoints
	oauth.POST("/token", handler.IssueToken, middleware.ClientAuthMiddleware)
	oauth.POST("/revoke", handler.RevokeToken, middleware.ClientAuthMiddleware)
	oauth.POST("/introspect", handler.ValidateToken, middleware.ClientAuthMiddleware)

	// Protected resource endpoints
	api := e.Group("/api")
	api.Use(middleware.BearerTokenMiddleware) // All API routes require a valid access token

	// Add protected API endpoints here
	// For example:
	// api.GET("/user", handler.GetUserInfo)

	// Start server
	port := cfg.Server.Port
	log.Info("Starting server", zap.String("port", port))
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
