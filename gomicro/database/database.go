package database

import (
	"fmt"
	"log"

	"github.com/suteetoe/gomicro/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB is the global database instance
var DB *gorm.DB

// InitDB initializes the database connection with configuration
func InitDB(dbConfig *config.DBConfig) (*gorm.DB, error) {
	var err error

	// Configure Postgres options
	pgConfig := postgres.Config{
		DSN:                  dbConfig.GetDSN(),
		PreferSimpleProtocol: true, // Disables implicit prepared statement usage
	}

	// Open connection
	DB, err = gorm.Open(postgres.New(pgConfig), &gorm.Config{
		Logger: logger.Default.LogMode(dbConfig.LogLevel),
	})
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		return nil, err
	}

	// Get generic database object SQL
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Failed to get database object: %v", err)
		return nil, err
	}

	// Set connection pool settings from config
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(dbConfig.ConnMaxLifetime)

	fmt.Println("Database connected successfully")

	return DB, nil
}

// MigrateModels runs migrations for the provided models
func MigrateModels(models ...interface{}) error {
	if DB == nil {
		return fmt.Errorf("database is not initialized")
	}

	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
