# Migration Guide

This guide explains how to migrate an existing microservice to use the shared `gomicro` package.

## Migration Steps

### 1. Update go.mod File

First, add the `gomicro` package as a dependency:

```sh
cd your-service
go get github.com/suteetoe/gomicro
```

### 2. Replace Configuration

Replace service-specific configuration with the shared configuration:

**Before:**
```go
import "your-service/pkg/config"

// Load configuration
conf, err := config.Load()
```

**After:**
```go
import "github.com/suteetoe/gomicro/config"

// Load configuration with service name
conf, err := config.Load("your-service-name")
```

### 3. Replace Database Connection

**Before:**
```go
import "your-service/pkg/database"

// Initialize database
db, err := database.InitDB(&conf.DB)
```

**After:**
```go
import "github.com/suteetoe/gomicro/database"

// Create database configuration
dbConfig := &database.DatabaseConfig{
    Host:            conf.DB.Host,
    Port:            conf.DB.Port,
    User:            conf.DB.User,
    Password:        conf.DB.Password,
    DBName:          conf.DB.DBName,
    SSLMode:         conf.DB.SSLMode,
    MaxIdleConns:    conf.DB.MaxIdleConns,
    MaxOpenConns:    conf.DB.MaxOpenConns,
    ConnMaxLifetime: conf.DB.ConnMaxLifetime,
    LogLevel:        conf.DB.LogLevel,
}

// Initialize database
db, err := database.InitDB(dbConfig)
```

### 4. Replace JWT Utilities

**Before:**
```go
import "your-service/pkg/jwtutil"

// Initialize JWT utilities
jwtutil.Initialize(&conf.JWT)

// Generate token
token, err := jwtutil.GenerateToken(email, userID)

// Validate token
claims, err := jwtutil.ValidateToken(tokenString)
```

**After:**
```go
import "github.com/suteetoe/gomicro/jwtutil"

// Create JWT configuration
jwtConfig := &jwtutil.JWTConfig{
    SigningKey:      conf.JWT.SigningKey,
    ExpirationHours: conf.JWT.ExpirationHours,
}

// Create JWT utility
jwt := jwtutil.NewJWTUtil(jwtConfig)

// Generate token
token, err := jwt.GenerateToken(email, userID)

// Validate token
claims, err := jwt.ValidateToken(tokenString)
```

### 5. Replace Logger

**Before:**
```go
import "your-service/pkg/logger"

// Initialize logger
logger.InitLogger(conf)

// Get logger
log := logger.GetLogger()
```

**After:**
```go
import "github.com/suteetoe/gomicro/logger"

// Initialize logger
err := logger.InitLogger(&logger.LogConfig{
    Level:       conf.Log.Level,
    Environment: conf.Server.Env,
    ServiceName: conf.ServiceName,
})

// Get logger
log := logger.GetLogger()
```

### 6. Replace Middleware

**Before:**
```go
import "your-service/internal/middleware"

// Initialize Echo with middleware
e.Use(middleware.RequestID())
e.Use(middleware.Logger())

// JWT middleware
protected := e.Group("/api")
protected.Use(middleware.JWTAuth())
```

**After:**
```go
import (
    "github.com/suteetoe/gomicro/middleware"
    "github.com/suteetoe/gomicro/logger"
    "github.com/suteetoe/gomicro/jwtutil"
)

// Initialize Echo with middleware
e.Use(middleware.RequestIDMiddleware())
e.Use(logger.Middleware())

// JWT middleware
jwtConfig := &jwtutil.JWTConfig{
    SigningKey:      conf.JWT.SigningKey,
    ExpirationHours: conf.JWT.ExpirationHours,
}
jwt := jwtutil.NewJWTUtil(jwtConfig)

protected := e.Group("/api")
protected.Use(middleware.JWTAuthMiddleware(jwt))
```

## Example: Full Migration of main.go

Here is a complete example of migrating a main.go file from using service-specific packages to using the shared gomicro package:

**Before:**

```go
package main

import (
	"log"
	
	"github.com/labstack/echo/v4"
	"your-service/pkg/config"
	"your-service/pkg/database"
	"your-service/pkg/logger"
	"your-service/internal/middleware"
	"your-service/internal/handler"
)

func main() {
	// Load configuration
	conf, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	logger.InitLogger(conf)
	log := logger.GetLogger()
	
	// Initialize database
	db, err := database.InitDB(&conf.DB)
	if err != nil {
		log.Fatal("Failed to initialize database", err)
	}
	
	// Initialize JWT utilities
	jwtutil.Initialize(&conf.JWT)
	
	// Initialize Echo
	e := echo.New()
	
	// Apply middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	
	// Set up routes
	h := handler.NewHandler(db)
	e.GET("/health", h.HealthCheck)
	
	// Protected routes
	api := e.Group("/api")
	api.Use(middleware.JWTAuth())
	api.GET("/resource", h.GetResource)
	
	// Start server
	log.Info("Starting server on port " + conf.Server.Port)
	e.Logger.Fatal(e.Start(":" + conf.Server.Port))
}
```

**After:**

```go
package main

import (
	"log"
	
	"github.com/labstack/echo/v4"
	"github.com/suteetoe/gomicro/config"
	"github.com/suteetoe/gomicro/database"
	"github.com/suteetoe/gomicro/jwtutil"
	"github.com/suteetoe/gomicro/logger"
	"github.com/suteetoe/gomicro/middleware"
	
	"your-service/internal/handler"
)

func main() {
	// Load configuration
	conf, err := config.Load("your-service-name")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Initialize logger
	err = logger.InitLogger(&logger.LogConfig{
		Level:       conf.Log.Level,
		Environment: conf.Server.Env,
		ServiceName: conf.ServiceName,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	log := logger.GetLogger()
	
	// Initialize database
	dbConfig := &database.DatabaseConfig{
		Host:            conf.DB.Host,
		Port:            conf.DB.Port,
		User:            conf.DB.User,
		Password:        conf.DB.Password,
		DBName:          conf.DB.DBName,
		SSLMode:         conf.DB.SSLMode,
		MaxIdleConns:    conf.DB.MaxIdleConns,
		MaxOpenConns:    conf.DB.MaxOpenConns,
		ConnMaxLifetime: conf.DB.ConnMaxLifetime,
		LogLevel:        conf.DB.LogLevel,
	}
	db, err := database.InitDB(dbConfig)
	if err != nil {
		log.Fatal("Failed to initialize database", err)
	}
	
	// Initialize JWT utility
	jwtConfig := &jwtutil.JWTConfig{
		SigningKey:      conf.JWT.SigningKey,
		ExpirationHours: conf.JWT.ExpirationHours,
	}
	jwt := jwtutil.NewJWTUtil(jwtConfig)
	
	// Initialize Echo
	e := echo.New()
	
	// Apply middleware
	e.Use(middleware.RequestIDMiddleware())
	e.Use(logger.Middleware())
	
	// Set up routes
	h := handler.NewHandler(db)
	e.GET("/health", h.HealthCheck)
	
	// Protected routes
	api := e.Group("/api")
	api.Use(middleware.JWTAuthMiddleware(jwt))
	api.GET("/resource", h.GetResource)
	
	// Start server
	log.Info("Starting server on port " + conf.Server.Port)
	e.Logger.Fatal(e.Start(":" + conf.Server.Port))
}
```

## Benefits of Migration

1. **Consistency**: All services use the same configuration, logging, database, and JWT implementations.
2. **Maintainability**: Updates to shared code are made in one place and automatically apply to all services.
3. **Reduced Duplication**: Eliminates duplicate code across services.
4. **Standardization**: Enforces common patterns across all microservices.
5. **Easier Onboarding**: New developers only need to learn one set of utilities.

## Migration Testing

After migrating, thoroughly test your service to ensure:

1. Configuration is correctly loaded
2. Database connections work properly
3. JWT tokens are correctly generated and validated
4. Logging functionality works as expected
5. Middleware correctly processes requests

We recommend migrating one service at a time, starting with a less critical service to validate the approach.