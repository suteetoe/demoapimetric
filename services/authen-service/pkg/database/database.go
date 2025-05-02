package database

import (
	"auth-service/internal/model"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// DBConfig holds the database configuration
type DBConfig struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        logger.LogLevel
}

// Initialize initializes the database connection with the provided configuration
func Initialize(config DBConfig) error {
	var err error

	// Set default log level if not specified
	logLevel := config.LogLevel
	if logLevel == 0 {
		logLevel = logger.Info
	}

	// Connect to the database with DisableAutoPrepare option to prevent "prepared statement already exists" errors
	pgConfig := postgres.Config{
		DSN:                  config.DSN,
		PreferSimpleProtocol: true, // Disables implicit prepared statement usage
	}

	DB, err = gorm.Open(postgres.New(pgConfig), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return err
	}

	// Configure connection pool
	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
		return err
	}

	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}

	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}

	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}

	// AutoMigrate will automatically create or update the table structure based on our models
	err = DB.AutoMigrate(&model.User{}, &model.Tenant{}, &model.UserTenant{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
		return err
	}

	fmt.Println("Database connected and migrated successfully")
	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
