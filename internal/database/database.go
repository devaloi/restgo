package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/devaloi/restgo/internal/config"
	"github.com/devaloi/restgo/internal/domain"
)

func Connect(cfg config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	db.SetMaxOpenConns(domain.DBMaxOpenConns)
	db.SetMaxIdleConns(domain.DBMaxIdleConns)
	db.SetConnMaxLifetime(domain.DBConnMaxLifetime)
	db.SetConnMaxIdleTime(domain.DBConnMaxIdleTime)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return db, nil
}
