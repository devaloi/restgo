package domain

import "time"

// Pagination defaults and limits.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Validation limits.
const (
	MaxTitleLength = 255
)

// Rate limiting defaults.
const (
	DefaultRateLimit    = 100
	SecondsPerMinute    = 60.0
)

// Database connection pool settings.
const (
	DBMaxOpenConns    = 25
	DBMaxIdleConns    = 10
	DBConnMaxLifetime = 5 * time.Minute
	DBConnMaxIdleTime = 1 * time.Minute
)

// CORS settings.
const (
	CORSMaxAge = "86400"
)
