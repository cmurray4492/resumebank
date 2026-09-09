package config

import "testing"

func withEnv(t *testing.T, kv map[string]string, fn func()) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
	fn()
}

func TestLoad_RequiresDatabaseURLAndSessionSecret(t *testing.T) {
	withEnv(t, map[string]string{"DATABASE_URL": "", "SESSION_SECRET": ""}, func() {
		if _, err := Load(); err == nil {
			t.Error("expected an error when DATABASE_URL and SESSION_SECRET are unset")
		}
	})
}

func TestLoad_Defaults(t *testing.T) {
	withEnv(t, map[string]string{
		"DATABASE_URL":   "postgres://localhost/db",
		"SESSION_SECRET": "secret",
	}, func() {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.Env != "development" {
			t.Errorf("Env = %q, want development", cfg.Env)
		}
		if cfg.Port != "8080" {
			t.Errorf("Port = %q, want 8080", cfg.Port)
		}
		if cfg.CookieSecure {
			t.Error("expected CookieSecure to default to false")
		}
		if cfg.AutoMigrate {
			t.Error("expected AutoMigrate to default to false")
		}
		if cfg.BaseURL != "http://localhost:8080" {
			t.Errorf("BaseURL = %q, want http://localhost:8080", cfg.BaseURL)
		}
		if cfg.OllamaURL != "http://localhost:11434" {
			t.Errorf("OllamaURL = %q, want http://localhost:11434", cfg.OllamaURL)
		}
		if cfg.EmbedModel != "nomic-embed-text" {
			t.Errorf("EmbedModel = %q, want nomic-embed-text", cfg.EmbedModel)
		}
	})
}

func TestLoad_Overrides(t *testing.T) {
	withEnv(t, map[string]string{
		"DATABASE_URL":   "postgres://localhost/db",
		"SESSION_SECRET": "secret",
		"PORT":           "9090",
		"BASE_URL":       "https://resumebank.biz/",
		"COOKIE_SECURE":  "true",
		"AUTO_MIGRATE":   "true",
		"MAX_UPLOAD_MB":  "25",
	}, func() {
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.BaseURL != "https://resumebank.biz" {
			t.Errorf("BaseURL = %q, want trailing slash trimmed", cfg.BaseURL)
		}
		if !cfg.CookieSecure {
			t.Error("expected CookieSecure to be true")
		}
		if !cfg.AutoMigrate {
			t.Error("expected AutoMigrate to be true")
		}
		if cfg.MaxUploadBytes != 25*1024*1024 {
			t.Errorf("MaxUploadBytes = %d, want %d", cfg.MaxUploadBytes, 25*1024*1024)
		}
	})
}
