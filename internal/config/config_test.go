package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars that might interfere
	for _, key := range []string{"DB_HOST", "DB_PORT", "SERVER_PORT", "JWT_EXPIRY", "RATE_LIMIT"} {
		os.Unsetenv(key)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DB.Host != "localhost" {
		t.Errorf("expected DB.Host=localhost, got %s", cfg.DB.Host)
	}
	if cfg.DB.Port != "5432" {
		t.Errorf("expected DB.Port=5432, got %s", cfg.DB.Port)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("expected Server.Port=8080, got %s", cfg.Server.Port)
	}
	if cfg.JWT.Expiry != 24*time.Hour {
		t.Errorf("expected JWT.Expiry=24h, got %v", cfg.JWT.Expiry)
	}
	if cfg.Rate.Limit != 100 {
		t.Errorf("expected Rate.Limit=100, got %d", cfg.Rate.Limit)
	}
	if cfg.CORS.Origins != "*" {
		t.Errorf("expected CORS.Origins=*, got %s", cfg.CORS.Origins)
	}
}

func TestLoadFromEnv(t *testing.T) {
	os.Setenv("DB_HOST", "db.example.com")
	os.Setenv("SERVER_PORT", "3000")
	os.Setenv("JWT_EXPIRY", "1h")
	os.Setenv("RATE_LIMIT", "50")
	defer func() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("JWT_EXPIRY")
		os.Unsetenv("RATE_LIMIT")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DB.Host != "db.example.com" {
		t.Errorf("expected DB.Host=db.example.com, got %s", cfg.DB.Host)
	}
	if cfg.Server.Port != "3000" {
		t.Errorf("expected Server.Port=3000, got %s", cfg.Server.Port)
	}
	if cfg.JWT.Expiry != time.Hour {
		t.Errorf("expected JWT.Expiry=1h, got %v", cfg.JWT.Expiry)
	}
	if cfg.Rate.Limit != 50 {
		t.Errorf("expected Rate.Limit=50, got %d", cfg.Rate.Limit)
	}
}

func TestLoadInvalidExpiry(t *testing.T) {
	os.Setenv("JWT_EXPIRY", "notaduration")
	defer os.Unsetenv("JWT_EXPIRY")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid JWT_EXPIRY")
	}
}

func TestDBConfigDSN(t *testing.T) {
	cfg := DBConfig{
		Host:    "localhost",
		Port:    "5432",
		User:    "restgo",
		Pass:    "secret",
		Name:    "restgo",
		SSLMode: "disable",
	}
	expected := "host=localhost port=5432 user=restgo password=secret dbname=restgo sslmode=disable"
	if cfg.DSN() != expected {
		t.Errorf("expected DSN=%s, got %s", expected, cfg.DSN())
	}
}
