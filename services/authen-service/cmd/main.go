package main

import (
	"auth-service/internal/handler"
	"auth-service/internal/middleware"
	"auth-service/pkg/config"
	"auth-service/pkg/database"
	"auth-service/pkg/jwtutil"
	"auth-service/pkg/logger"
	"auth-service/prometheus"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
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
	log.Info("Starting authentication service...", zap.String("environment", cfg.Server.Env))

	// Initialize database
	if err := database.InitDB(cfg); err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}
	log.Info("Database connection established")

	// Initialize JWT utility
	jwtutil.Initialize(&cfg.JWT)
	log.Info("JWT utility initialized")

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
	e.GET("/health", handler.HealthCheck)
	e.GET("/metrics", handler.MetricsHandler)

	// Authentication routes - these don't belong under /api since they're for getting access to the API
	auth := e.Group("/auth")
	auth.POST("/login", handler.Login)
	auth.POST("/register", handler.Register)

	// API routes - all require authentication
	api := e.Group("/api")
	api.Use(middleware.AuthMiddleware)

	// User management
	users := api.Group("/users")
	users.GET("/profile", handler.GetProfile)
	users.PATCH("/profile", handler.UpdateProfile)
	users.POST("/change-password", handler.ChangePassword)

	// Tenant selection - after login but before accessing tenant-specific resources
	tenantAuth := api.Group("/tenant-auth")
	tenantAuth.POST("/select", handler.SelectTenant)
	tenantAuth.POST("/switch", handler.SwitchTenant)
	tenantAuth.POST("/default", handler.SetDefaultTenant)

	// Tenant management - doesn't require tenant context
	tenants := api.Group("/tenants")
	tenants.POST("", handler.CreateTenant)
	tenants.GET("", handler.ListUserTenants)

	// Tenant-specific operations - requires tenant context
	tenantSpecific := api.Group("/tenants")
	tenantSpecific.Use(middleware.RequireTenantContext)
	tenantSpecific.GET("/:id", handler.GetTenant)

	// Tenant user management - requires tenant context
	tenantUsers := api.Group("/tenant-users")
	tenantUsers.Use(middleware.RequireTenantContext)
	tenantUsers.POST("", handler.AddUserToTenant)
	tenantUsers.DELETE("/:tenant_id/:user_id", handler.RemoveUserFromTenant)

	// Get server port from configuration
	port := cfg.Server.Port

	// Start server
	log.Info("Starting server", zap.String("port", port))
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Failed to start server", zap.Error(err))
	}
}
