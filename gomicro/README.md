# Gomicro

A shared Go package for microservices with common utilities and patterns.

## Overview

`gomicro` is a shared package designed to be used across all microservices in your application. It provides standardized implementations for:

- Configuration management
- Database connectivity
- JWT authentication
- Logging
- Middleware (authentication, request ID)

## Installation

Add the package to your Go module:

```sh
go get github.com/suteetoe/gomicro
```

## Usage

### Configuration

```go
import "github.com/suteetoe/gomicro/config"

// Initialize configuration for your service
conf, err := config.Load("your-service-name")
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Access configuration properties
port := conf.Server.Port
```

### Database

```go
import (
    "github.com/suteetoe/gomicro/config"
    "github.com/suteetoe/gomicro/database"
)

// Initialize configuration
conf, _ := config.Load("your-service-name")

// Create database configuration object
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

// Initialize the database connection
db, err := database.InitDB(dbConfig)
if err != nil {
    log.Fatalf("Failed to connect to database: %v", err)
}

// Migrate your models
err = database.MigrateModels(&YourModel{}, &AnotherModel{})
if err != nil {
    log.Fatalf("Failed to migrate models: %v", err)
}

// Use the database in your application
db := database.GetDB()
```

### JWT Authentication

```go
import (
    "github.com/suteetoe/gomicro/config"
    "github.com/suteetoe/gomicro/jwtutil"
)

// Initialize configuration
conf, _ := config.Load("your-service-name")

// Create JWT configuration
jwtConfig := &jwtutil.JWTConfig{
    SigningKey:      conf.JWT.SigningKey,
    ExpirationHours: conf.JWT.ExpirationHours,
}

// Create JWT utility
jwt := jwtutil.NewJWTUtil(jwtConfig)

// Generate a token
token, err := jwt.GenerateToken("user@example.com", 123)

// Validate a token
claims, err := jwt.ValidateToken(tokenString)
if err != nil {
    // Handle invalid token
}
```

### Logging

```go
import (
    "github.com/suteetoe/gomicro/config"
    "github.com/suteetoe/gomicro/logger"
)

// Initialize configuration
conf, _ := config.Load("your-service-name")

// Initialize logger
err := logger.InitLogger(&logger.LogConfig{
    Level:       conf.Log.Level,
    Environment: conf.Server.Env,
    ServiceName: conf.ServiceName,
})
if err != nil {
    log.Fatalf("Failed to initialize logger: %v", err)
}

// Get logger and use it
log := logger.GetLogger()
log.Info("Application starting...")
```

### Middleware

```go
import (
    "github.com/labstack/echo/v4"
    "github.com/suteetoe/gomicro/jwtutil"
    "github.com/suteetoe/gomicro/logger"
    "github.com/suteetoe/gomicro/middleware"
)

// Set up Echo with middleware
e := echo.New()

// Add request ID middleware
e.Use(middleware.RequestIDMiddleware())

// Add logger middleware
e.Use(logger.Middleware())

// Add JWT authentication middleware (to protected routes group)
jwtConfig := &jwtutil.JWTConfig{
    SigningKey:      "your-signing-key",
    ExpirationHours: 24,
}
jwt := jwtutil.NewJWTUtil(jwtConfig)

// Create a group for protected routes
protected := e.Group("/api")
protected.Use(middleware.JWTAuthMiddleware(jwt))
```

## Example Service Structure

Here's an example of how to structure a new microservice using the `gomicro` package:

```
your-service/
├── cmd/
│   └── main.go
├── internal/
│   ├── handler/
│   │   └── handlers.go
│   ├── model/
│   │   └── models.go
│   └── service/
│       └── services.go
├── go.mod
└── go.sum
```

Example `main.go`:

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
    "your-service/internal/model"
)

func main() {
    // Load configuration
    conf, err := config.Load("your-service")
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

    // Run migrations
    err = database.MigrateModels(&model.YourModel{})
    if err != nil {
        log.Fatal("Failed to run migrations", err)
    }

    // Initialize JWT utility
    jwtConfig := &jwtutil.JWTConfig{
        SigningKey:      conf.JWT.SigningKey,
        ExpirationHours: conf.JWT.ExpirationHours,
    }
    jwt := jwtutil.NewJWTUtil(jwtConfig)

    // Initialize Echo framework
    e := echo.New()

    // Apply middleware
    e.Use(middleware.RequestIDMiddleware())
    e.Use(logger.Middleware())

    // Initialize handlers
    h := handler.NewHandler(db)

    // Set up routes
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