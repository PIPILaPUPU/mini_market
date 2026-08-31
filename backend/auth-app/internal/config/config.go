package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL    string
	JWTSecret      string
	JWTIssuer      string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	HTTPPort       string
	CookieSecure   bool
	CookieSameSite string
}

func Load() (Config, error) {
	config := Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTIssuer:      envOrDefault("JWT_ISSUER", "mini-market-auth"),
		HTTPPort:       envOrDefault("HTTP_PORT", "8080"),
		CookieSameSite: strings.ToLower(envOrDefault("COOKIE_SAME_SITE", "lax")),
	}
	if config.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if len(config.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 characters")
	}

	var err error
	if config.AccessTTL, err = duration("ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		return Config{}, err
	}
	if config.RefreshTTL, err = duration("REFRESH_TOKEN_TTL", 30*24*time.Hour); err != nil {
		return Config{}, err
	}
	if config.CookieSecure, err = boolean("COOKIE_SECURE", false); err != nil {
		return Config{}, err
	}
	switch config.CookieSameSite {
	case "lax", "strict", "none":
	default:
		return Config{}, errors.New("COOKIE_SAME_SITE must be lax, strict or none")
	}
	if config.CookieSameSite == "none" && !config.CookieSecure {
		return Config{}, errors.New("COOKIE_SECURE must be true when COOKIE_SAME_SITE=none")
	}
	return config, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}

func boolean(key string, fallback bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false", key)
	}
	return parsed, nil
}
