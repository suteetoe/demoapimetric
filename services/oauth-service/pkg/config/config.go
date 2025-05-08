package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OAuth    OAuthConfig
	JWT      JWTConfig
	Log      LogConfig
	Metrics  MetricsConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port string
	Env  string
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        string
}

// OAuthConfig holds OAuth2-related configuration
type OAuthConfig struct {
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	SigningKey     string
	ExpirationTime time.Duration
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level string
}

// MetricsConfig holds metrics-related configuration
type MetricsConfig struct {
	Prefix string
}

// Load loads the application configuration from environment variables
func Load() (*Config, error) {
	// Load environment variables from .env file if it exists
	_ = godotenv.Load()

	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8084"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "oauth_db"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 100),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 1*time.Hour),
			LogLevel:        getEnv("DB_LOG_LEVEL", "info"),
		},
		OAuth: OAuthConfig{
			AccessTokenExpiration:  getEnvAsDuration("OAUTH_ACCESS_TOKEN_EXPIRATION", 1*time.Hour),
			RefreshTokenExpiration: getEnvAsDuration("OAUTH_REFRESH_TOKEN_EXPIRATION", 7*24*time.Hour),
		},
		JWT: JWTConfig{
			SigningKey:     getEnv("JWT_SIGNING_KEY", "oauthservicesecretkey"),
			ExpirationTime: getEnvAsDuration("JWT_EXPIRATION_HOURS", 24*time.Hour),
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		Metrics: MetricsConfig{
			Prefix: getEnv("METRICS_PREFIX", "oauth"),
		},
	}, nil
}

// Helper functions to get environment variables with defaults
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
