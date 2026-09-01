package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	JWTIssuer   string
	HTTPPort    string
}

func Load() (Config, error) {
	config := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTIssuer:   envOrDefault("JWT_ISSUER", "mini-market-auth"),
		HTTPPort:    envOrDefault("HTTP_PORT", "8081"),
	}
	if config.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(config.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 characters")
	}
	return config, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
