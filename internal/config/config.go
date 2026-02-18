package config

import (
	"fmt"
	"os"
	"time"

	"github.com/devaloi/restgo/internal/domain"
)

type Config struct {
	DB     DBConfig
	JWT    JWTConfig
	Server ServerConfig
	CORS   CORSConfig
	Rate   RateConfig
	Log    LogConfig
}

type DBConfig struct {
	Host    string
	Port    string
	User    string
	Pass    string
	Name    string
	SSLMode string
}

func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Pass, c.Name, c.SSLMode,
	)
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type ServerConfig struct {
	Port string
}

type CORSConfig struct {
	Origins string
}

type RateConfig struct {
	Limit int
}

type LogConfig struct {
	Level string
}

func Load() (*Config, error) {
	expiry, err := time.ParseDuration(envOrDefault("JWT_EXPIRY", "24h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY: %w", err)
	}

	limit := domain.DefaultRateLimit
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		n := 0
		for _, ch := range v {
			if ch < '0' || ch > '9' {
				return nil, fmt.Errorf("invalid RATE_LIMIT: %s", v)
			}
			n = n*10 + int(ch-'0')
		}
		limit = n
	}

	cfg := &Config{
		DB: DBConfig{
			Host:    envOrDefault("DB_HOST", "localhost"),
			Port:    envOrDefault("DB_PORT", "5432"),
			User:    envOrDefault("DB_USER", "restgo"),
			Pass:    envOrDefault("DB_PASS", "restgo"),
			Name:    envOrDefault("DB_NAME", "restgo"),
			SSLMode: envOrDefault("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			Secret: envOrDefault("JWT_SECRET", "change-me-in-production"),
			Expiry: expiry,
		},
		Server: ServerConfig{
			Port: envOrDefault("SERVER_PORT", "8080"),
		},
		CORS: CORSConfig{
			Origins: envOrDefault("CORS_ORIGINS", "*"),
		},
		Rate: RateConfig{
			Limit: limit,
		},
		Log: LogConfig{
			Level: envOrDefault("LOG_LEVEL", "info"),
		},
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
