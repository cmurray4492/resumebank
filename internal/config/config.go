// Package config loads application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env            string // "development" or "production"
	Port           string
	DatabaseURL    string
	SessionSecret  string
	CookieSecure   bool
	UploadDir      string
	MaxUploadBytes int64
	BaseURL        string // absolute origin (no trailing slash), used to build sitemap URLs
	OllamaURL      string // base URL of a local Ollama server, for embeddings
	EmbedModel     string // Ollama embedding model name
	AutoMigrate    bool   // apply pending DB migrations on server startup (see cmd/server)
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
	cfg.BaseURL = strings.TrimRight(getEnv("BASE_URL", "http://localhost:"+cfg.Port), "/")
	cfg.OllamaURL = strings.TrimRight(getEnv("OLLAMA_URL", "http://localhost:11434"), "/")
	cfg.EmbedModel = getEnv("OLLAMA_EMBED_MODEL", "nomic-embed-text")
	cfg.AutoMigrate = getEnvBool("AUTO_MIGRATE", false)

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
