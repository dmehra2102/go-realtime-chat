package config

import "os"

type Config struct {
	AuthServicePort string
	ChatServicePort string
	AuthDBURL       string
	ChatDBURL       string
	RedisAddr       string
	RedisPassword   string
	JWTSecret       string
}

func LoadConfig() *Config {
	return &Config{
		AuthServicePort: getEnv("AUTH_SERVICE_PORT", "8001"),
		ChatServicePort: getEnv("CHAT_SERVICE_PORT", "8002"),
		AuthDBURL:       getEnv("AUTH_DB_URL", ""),
		ChatDBURL:       getEnv("CHAT_DB_URL", ""),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		JWTSecret:       getEnv("JWT_SECRET", "secret-key-for-development"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
