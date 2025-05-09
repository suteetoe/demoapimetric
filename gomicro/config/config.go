package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/gorm/logger"
)

// DBConfig holds database configuration
type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        logger.LogLevel
}

// GetDSN returns the PostgreSQL connection string
func (c *DBConfig) GetDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
	Env  string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	SigningKey      string
	ExpirationHours int
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level string
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Prefix string
}

// Config holds all configuration
type Config struct {
	ServiceName string
	DB          DBConfig
	Server      ServerConfig
	JWT         JWTConfig
	Log         LogConfig
	Metrics     MetricsConfig
}

// Load loads configuration from environment variables without service name prefix
func Load(serviceName string) (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Not returning error as .env file is optional
		fmt.Printf("Warning: .env file not found, using environment variables\n")
	}

	// Initialize config struct with values from environment
	config := &Config{
		ServiceName: serviceName,
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "password"),
			DBName:          getEnv("DB_NAME", serviceName),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 1*time.Hour),
			LogLevel:        getEnvAsLogLevel("DB_LOG_LEVEL", logger.Info),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Env:  getEnv("APP_ENV", "development"),
		},
		JWT: JWTConfig{
			SigningKey:      getEnv("JWT_SIGNING_KEY", "defaultsecretkey"),
			ExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Metrics: MetricsConfig{
			Prefix: getEnv("METRICS_PREFIX", serviceName),
		},
	}

	return config, nil
}

// LogConfig returns the configuration as a zap logger-friendly format
func (c *Config) LogConfig() []zap.Field {
	return []zap.Field{
		zap.String("service", c.ServiceName),
		zap.String("environment", c.Server.Env),
		zap.String("db_host", c.DB.Host),
		zap.String("db_port", c.DB.Port),
		zap.String("db_user", c.DB.User),
		zap.String("db_name", c.DB.DBName),
		zap.String("server_port", c.Server.Port),
	}
}

// Helper function to mask sensitive information in DSN
func maskDSN(dsn string) string {
	// A simple implementation that hides password details
	return "***MASKED***"
}

// Helper function to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// Helper function to get environment variables as integers
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// Helper function to get environment variables as durations
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// Helper function to get environment variables as log levels
func getEnvAsLogLevel(key string, defaultValue logger.LogLevel) logger.LogLevel {
	valueStr := getEnv(key, "")
	switch valueStr {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return defaultValue
	}
}
