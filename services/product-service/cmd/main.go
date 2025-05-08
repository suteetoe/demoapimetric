package main

import (
	"net/http"
	"product-service/internal/handler"
	mid "product-service/internal/middleware"
	"product-service/pkg/config"
	"product-service/pkg/database"
	"product-service/pkg/jwtutil"
	"product-service/pkg/logger"
	"product-service/pkg/oauth"
	"product-service/prometheus"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		// Just log a warning, don't fail if .env file is not found
		// This allows the service to run in environments where env vars are set differently
		// such as production environments with proper environment configuration
		// The fallback values will be used in case env vars are not set
	}

	// Load configuration
	appConfig, err := config.Load()
	if err != nil {
		// Can't use structured logger yet since it's not initialized
		panic("Failed to load configuration: " + err.Error())
	}

	// Initialize logger
	logger.InitLogger(appConfig)
	log := logger.GetLogger()
	defer log.Sync()

	log.Info("Starting product-service",
		zap.String("environment", appConfig.Server.Env),
		zap.String("port", appConfig.Server.Port))

	// Initialize JWT utility (for legacy support)
	jwtutil.Initialize(&appConfig.JWT)
	log.Info("JWT utility initialized")

	// Initialize Prometheus metrics
	prometheus.InitMetrics(appConfig)
	log.Info("Prometheus metrics initialized",
		zap.String("metrics_prefix", appConfig.Metrics.Prefix))

	// Initialize database
	err = database.InitDB(appConfig)
	if err != nil {
		log.Fatal("Failed to initialize database", zap.Error(err))
	}
	log.Info("Database connection established")

	// Initialize OAuth client if enabled
	var oauthClient *oauth.Client
	if appConfig.OAuth.Enabled {
		oauthClient = oauth.NewClient(
			appConfig.OAuth.BaseURL,
			appConfig.OAuth.ClientID,
			appConfig.OAuth.ClientSecret,
			log.With(zap.String("component", "oauth_client")),
		)
		log.Info("OAuth client initialized",
			zap.String("oauth_base_url", appConfig.OAuth.BaseURL),
			zap.String("oauth_client_id", appConfig.OAuth.ClientID))

		// Initialize the OAuth client for use in handlers
		handler.InitOAuthClient(oauthClient)
	}

	// Initialize Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Recover())
	e.Use(mid.RequestIDMiddleware)
	e.Use(mid.MetricsMiddleware)

	// Routes
	// Metrics endpoint
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Legacy route
	e.GET("/merchant/hello", handler.Hello)

	// Example route that uses OAuth for service-to-service communication
	if appConfig.OAuth.Enabled {
		e.GET("/example/suppliers", handler.GetSuppliersExample)
	}

	// Product API routes
	productAPI := e.Group("/api/products")

	// Choose authentication method based on config
	if appConfig.OAuth.Enabled && oauthClient != nil {
		// Use OAuth2 authentication
		log.Info("Using OAuth2 authentication for API routes")
		productAPI.Use(oauth.Middleware(oauthClient, []string{"read", "write"}))
	} else {
		// Use legacy JWT authentication
		log.Info("Using legacy JWT authentication for API routes")
		productAPI.Use(mid.AuthMiddleware)
	}

	productAPI.GET("", handler.ListProducts)
	productAPI.GET("/:id", handler.GetProduct)
	productAPI.POST("", handler.CreateProduct)
	productAPI.PUT("/:id", handler.UpdateProduct)
	productAPI.DELETE("/:id", handler.DeleteProduct)

	// Category API routes
	categoryAPI := e.Group("/api/categories")

	// Choose authentication method based on config
	if appConfig.OAuth.Enabled && oauthClient != nil {
		// Use OAuth2 authentication
		categoryAPI.Use(oauth.Middleware(oauthClient, []string{"product:read", "product:write"}))
	} else {
		// Use legacy JWT authentication
		categoryAPI.Use(mid.AuthMiddleware)
	}

	categoryAPI.GET("", handler.ListCategories)
	categoryAPI.GET("/:id", handler.GetCategory)
	categoryAPI.POST("", handler.CreateCategory)
	categoryAPI.PUT("/:id", handler.UpdateCategory)
	categoryAPI.DELETE("/:id", handler.DeleteCategory)

	// Start server
	port := appConfig.Server.Port
	log.Info("Starting server", zap.String("port", port))
	if err := e.Start(":" + port); err != nil {
		log.Fatal("Server error", zap.Error(err))
	}
}
