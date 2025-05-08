package database

import (
	"fmt"
	"oauth-service/internal/model"
	"oauth-service/pkg/config"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db *gorm.DB
)

// InitDB initializes the database connection
func InitDB(cfg *config.Config) error {
	// Set up GORM logger configuration
	var logLevel logger.LogLevel
	if cfg.Server.Env == "development" {
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}

	// Override log level if explicitly set in config
	switch cfg.Database.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	}

	// Build DSN from config
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Configure Postgres options
	pgConfig := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // Disables implicit prepared statement usage
	}

	// Configure GORM and open connection
	var err error
	db, err = gorm.Open(postgres.New(pgConfig), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool parameters
	sqlDB.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	// Get logger for migration logging
	var log *zap.Logger
	if appLogger, err := zap.NewProduction(); err == nil {
		log = appLogger
	} else {
		// Use default logger if we can't get the app logger
		log, _ = zap.NewProduction()
	}

	// Run migrations
	start := time.Now()
	log.Info("Starting database migration...")

	// Migrate the schema for OAuth models
	if err := db.AutoMigrate(
		&model.Client{},
		&model.AccessToken{},
		&model.RefreshToken{},
	); err != nil {
		log.Error("Database migration failed", zap.Error(err))
		return fmt.Errorf("failed to migrate database schema: %w", err)
	}

	log.Info("Database migration completed successfully",
		zap.Duration("duration", time.Since(start)))

	fmt.Println("Database connected successfully")

	return nil
}

// GetDB returns a reference to the database instance
func GetDB() *gorm.DB {
	return db
}

// MigrateSchema is maintained for backward compatibility
// Use InitDB for new code as it now handles migrations automatically
func MigrateSchema(log *zap.Logger, models ...interface{}) error {
	if db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	start := time.Now()
	log.Info("Starting custom database migration...")

	if err := db.AutoMigrate(models...); err != nil {
		log.Error("Custom database migration failed", zap.Error(err))
		return fmt.Errorf("failed to migrate custom schema: %w", err)
	}

	log.Info("Custom database migration completed successfully",
		zap.Duration("duration", time.Since(start)))
	return nil
}
