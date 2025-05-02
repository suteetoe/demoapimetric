package database

import (
	"fmt"
	"log"
	"product-service/internal/model"
	"product-service/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

// InitDB initializes the database connection with configuration and runs migrations
func InitDB(config *config.Config) error {
	var err error

	// Configure GORM logger
	logLevel := logger.Info
	if config.Server.Env == "development" {
		logLevel = logger.Info
	} else {
		logLevel = logger.Error
	}

	// Create DSN string
	dsn := config.DB.GetDSN()

	// Configure Postgres options
	pgConfig := postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // Disables implicit prepared statement usage
	}

	// Open connection
	db, err = gorm.Open(postgres.New(pgConfig), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return err
	}

	// Get generic database object SQL
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database object: %v", err)
		return err
	}

	// Set connection pool settings from config
	sqlDB.SetMaxIdleConns(config.DB.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.DB.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.DB.ConnMaxLifetime)

	fmt.Println("Database connected successfully")

	// Run migrations
	if err := db.AutoMigrate(&model.Product{}, &model.ProductCategory{}); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return db
}
