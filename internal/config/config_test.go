package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars that might interfere
	for _, key := range []string{"DB_HOST", "DB_PORT", "SERVER_PORT", "JWT_EXPIRY", "RATE_LIMIT",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME"} {
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
	if cfg.DB.MaxOpenConns != 25 {
		t.Errorf("expected DB.MaxOpenConns=25, got %d", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 10 {
		t.Errorf("expected DB.MaxIdleConns=10, got %d", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected DB.ConnMaxLifetime=5m, got %v", cfg.DB.ConnMaxLifetime)
	}
	if cfg.DB.ConnMaxIdleTime != 1*time.Minute {
		t.Errorf("expected DB.ConnMaxIdleTime=1m, got %v", cfg.DB.ConnMaxIdleTime)
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

func TestLoadCustomDBPoolSettings(t *testing.T) {
	os.Setenv("DB_MAX_OPEN_CONNS", "50")
	os.Setenv("DB_MAX_IDLE_CONNS", "20")
	os.Setenv("DB_CONN_MAX_LIFETIME", "10m")
	os.Setenv("DB_CONN_MAX_IDLE_TIME", "3m")
	defer func() {
		os.Unsetenv("DB_MAX_OPEN_CONNS")
		os.Unsetenv("DB_MAX_IDLE_CONNS")
		os.Unsetenv("DB_CONN_MAX_LIFETIME")
		os.Unsetenv("DB_CONN_MAX_IDLE_TIME")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DB.MaxOpenConns != 50 {
		t.Errorf("expected DB.MaxOpenConns=50, got %d", cfg.DB.MaxOpenConns)
	}
	if cfg.DB.MaxIdleConns != 20 {
		t.Errorf("expected DB.MaxIdleConns=20, got %d", cfg.DB.MaxIdleConns)
	}
	if cfg.DB.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("expected DB.ConnMaxLifetime=10m, got %v", cfg.DB.ConnMaxLifetime)
	}
	if cfg.DB.ConnMaxIdleTime != 3*time.Minute {
		t.Errorf("expected DB.ConnMaxIdleTime=3m, got %v", cfg.DB.ConnMaxIdleTime)
	}
}
