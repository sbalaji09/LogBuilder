package config

import (
	"os"
	"strconv"
)

/*
this file allows for configuration settings to be loaded from environment variables with failback defaults for each parameter

ensures the app can be configured for different environments without changing code
*/
type Config struct {
	DatabaseURL   string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	ServerPort    string
	LogLevel      string
	Environment   string
	JWTSecret     string
	JWTIssuer     string
}

// creates a new Config object, using getEnv to check if the environment variable exists
func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://loguser:logpass123@localhost:5432/logs?sslmode=disable"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		Environment:   getEnv("ENVIRONMENT", "development"),
		JWTSecret:     getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-in-production"),
		JWTIssuer:     getEnv("JWT_ISSUER", "log-analytics-system"),
	}
}

// returns the value of an environment variable and null if not set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// tries to convert the environment variable to a key if need be
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
