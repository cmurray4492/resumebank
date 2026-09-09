// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Env            string // "development" or "production"
	Port           string
	DatabaseURL    string
	SessionSecret  string
	CookieSecure   bool
	UploadDir      string
	MaxUploadBytes int64
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:           getEnv("ENV", "development"),
		Port:          getEnv("PORT", "8080"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		SessionSecret: os.Getenv("SESSION_SECRET"),
		CookieSecure:  getEnvBool("COOKIE_SECURE", false),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}

	maxMB := getEnvInt("MAX_UPLOAD_MB", 10)
	cfg.MaxUploadBytes = int64(maxMB) * 1024 * 1024

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
