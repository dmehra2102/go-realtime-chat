package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AuthServicePort string
	ChatServicePort string
	RedisAddr       string
	RedisPassword   string
	JWTSecret       string
	Database        DatabaseConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

func (d DatabaseConfig) MigrationDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

func LoadConfig() *Config {
	return &Config{
		AuthServicePort: getEnv("AUTH_SERVICE_PORT", "8001"),
		ChatServicePort: getEnv("CHAT_SERVICE_PORT", "8002"),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		JWTSecret:       getEnv("JWT_SECRET", "secret-key-for-development"),
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			User:     getEnv("DB_USER", "hruser"),
			Password: getEnv("DB_PASSWORD", "hrpassword"),
			Name:     getEnv("DB_NAME", "hrmanagement"),
			Port:     parseIntOrDefault("DB_PORT"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func parseIntOrDefault(key string) int {
	value := os.Getenv(key)
	if i, err := strconv.Atoi(value); err == nil {
		return i
	}
	return 0
}
