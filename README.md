# restgo

[![CI](https://github.com/devaloi/restgo/actions/workflows/ci.yml/badge.svg)](https://github.com/devaloi/restgo/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A production-grade REST API in Go using only the standard library — user authentication, articles CRUD, pagination, and middleware stack. No Gin, no Chi, no Echo.

## Features

- **Standard library only** — `net/http` routing with Go 1.22+ method patterns
- **Hand-rolled JWT** — HMAC-SHA256 token generation and validation, no third-party auth libraries
- **Full middleware stack** — logging, CORS, rate limiting, recovery, request IDs
- **Repository pattern** — clean separation with PostgreSQL and in-memory implementations
- **Pagination & search** — cursor-free page-based pagination with full-text search
- **Request validation** — structured error responses with field-level detail
- **Graceful shutdown** — signal handling with connection draining
- **Docker ready** — single `docker compose up` for the full stack

## Quick Start

### With Docker (recommended)

```bash
git clone https://github.com/devaloi/restgo.git
cd restgo
docker compose up -d
# API available at http://localhost:8080
```

### Manual Build

```bash
# Prerequisites: Go 1.22+, PostgreSQL 16+

# Set up database
createdb restgo

# Configure (or copy .env.example to .env)
export DB_HOST=localhost DB_PORT=5432 DB_USER=restgo DB_PASS=restgo DB_NAME=restgo

# Build and run
make build
make run
```

### Without Database (demo mode)

If PostgreSQL is unavailable, the server starts with in-memory repositories:

```bash
go run ./cmd/restgo
# WARN: database unavailable, using in-memory repositories
# INFO: restgo server starting  port=8080
```

## API Reference

| Method   | Path                  | Auth | Description              |
|----------|-----------------------|------|--------------------------|
| `GET`    | `/health`             | No   | Health check             |
| `POST`   | `/api/auth/register`  | No   | Register a new user      |
| `POST`   | `/api/auth/login`     | No   | Login, receive JWT       |
| `GET`    | `/api/users/me`       | Yes  | Get current user profile |
| `GET`    | `/api/articles`       | No   | List articles (paginated)|
| `GET`    | `/api/articles/{id}`  | No   | Get article by ID        |
| `POST`   | `/api/articles`       | Yes  | Create article           |
| `PUT`    | `/api/articles/{id}`  | Yes  | Update article (owner)   |
| `DELETE` | `/api/articles/{id}`  | Yes  | Delete article (owner)   |

## Authentication

restgo uses hand-rolled JWT tokens with HMAC-SHA256 signing.

**Flow:**
1. Register or login to receive a JWT token
2. Include the token in subsequent requests via the `Authorization` header

```
Authorization: Bearer <token>
```

Tokens expire after 24 hours by default (configurable via `JWT_EXPIRY`).

Protected endpoints return `401 Unauthorized` without a valid token and `403 Forbidden` when acting on another user's resources.

## Pagination & Filtering

List endpoints support pagination and filtering via query parameters:

| Parameter   | Default | Description                    |
|-------------|---------|--------------------------------|
| `page`      | `1`     | Page number                    |
| `per_page`  | `20`    | Items per page                 |
| `search`    | —       | Search titles and bodies       |
| `author_id` | —       | Filter by author UUID          |
| `sort`      | `created_at` | Sort field (`created_at`, `updated_at`, `title`) |
| `dir`       | `desc`  | Sort direction (`asc`, `desc`) |

**Response format:**

```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42,
    "total_pages": 3
  }
}
```

## Architecture

```
cmd/restgo/main.go          Entry point, wiring, graceful shutdown
internal/
├── auth/                   JWT generation & validation, bcrypt
├── config/                 Environment-based configuration
├── database/               PostgreSQL connection & migration runner
├── domain/                 Core types, request/response models, errors
├── handler/                HTTP handlers, JSON response helpers
├── middleware/              Auth, CORS, logging, rate limit, recovery, request ID
├── repository/             Data access interfaces + PostgreSQL & mock implementations
├── router/                 Route registration, dependency wiring
└── service/                Business logic layer
migrations/                 Embedded SQL migration files
```

**Request flow:** `HTTP → middleware stack → router → handler → service → repository`

## Middleware Stack

| Middleware   | Description                                              |
|-------------|----------------------------------------------------------|
| **Recovery** | Catches panics, returns 500 JSON error                   |
| **RequestID**| Generates UUID, sets `X-Request-ID` header               |
| **Logging**  | Logs method, path, status, duration via `slog`           |
| **CORS**     | Configurable allowed origins, preflight handling         |
| **RateLimit**| Token bucket per client IP, configurable requests/min    |
| **Auth**     | JWT validation on protected routes, injects user context |

## Development

### Prerequisites

- Go 1.22+
- Docker & Docker Compose (for PostgreSQL)
- golangci-lint (optional, for linting)

### Commands

```bash
make build        # Build binary to bin/restgo
make run          # Build and run
make test         # Run all tests with -race
make lint         # Run golangci-lint
make cover        # Generate coverage report
make docker-up    # Start PostgreSQL + app via Docker Compose
make docker-down  # Stop and remove containers
make clean        # Remove build artifacts
```

## Example Requests

### Register

```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123","name":"Alice"}' | jq
```

```json
{
  "data": {
    "user": {
      "id": "a1b2c3d4-...",
      "email": "alice@example.com",
      "name": "Alice",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    },
    "token": "eyJhbGciOi..."
  }
}
```

### Login

```bash
curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"secret123"}' | jq
```

### Create Article

```bash
TOKEN="eyJhbGciOi..."  # from register or login response

curl -s -X POST http://localhost:8080/api/articles \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Hello World","body":"My first article"}' | jq
```

### List Articles with Pagination

```bash
curl -s "http://localhost:8080/api/articles?page=1&per_page=10&search=hello" | jq
```

### Update Article

```bash
curl -s -X PUT http://localhost:8080/api/articles/<id> \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Updated Title"}' | jq
```

### Delete Article

```bash
curl -s -X DELETE http://localhost:8080/api/articles/<id> \
  -H "Authorization: Bearer $TOKEN"
```

### Validation Error Response

```bash
curl -s -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"bad","password":"short","name":""}' | jq
```

```json
{
  "error": {
    "message": "validation failed",
    "details": [
      { "field": "email", "message": "invalid email format" },
      { "field": "password", "message": "password must be at least 8 characters" },
      { "field": "name", "message": "name is required" }
    ]
  }
}
```

## Configuration

All configuration is via environment variables (see [`.env.example`](.env.example)):

| Variable       | Default                    | Description                |
|----------------|----------------------------|----------------------------|
| `DB_HOST`      | `localhost`                | PostgreSQL host            |
| `DB_PORT`      | `5432`                     | PostgreSQL port            |
| `DB_USER`      | `restgo`                   | PostgreSQL user            |
| `DB_PASS`      | `restgo`                   | PostgreSQL password        |
| `DB_NAME`      | `restgo`                   | PostgreSQL database name   |
| `DB_SSLMODE`   | `disable`                  | PostgreSQL SSL mode        |
| `JWT_SECRET`   | `change-me-in-production`  | HMAC-SHA256 signing key    |
| `JWT_EXPIRY`   | `24h`                      | Token expiration duration  |
| `SERVER_PORT`  | `8080`                     | HTTP server port           |
| `CORS_ORIGINS` | `*`                        | Comma-separated origins    |
| `RATE_LIMIT`   | `100`                      | Requests per minute per IP |
| `LOG_LEVEL`    | `info`                     | Log level                  |

## Design Decisions

**Why stdlib-only?** Demonstrates deep understanding of Go's `net/http` package. Go 1.22 introduced method-based routing (`GET /path`), making external routers optional for most APIs.

**Why hand-rolled JWT?** JWT is three base64 segments with an HMAC signature — roughly 40 lines of code. Using a library for this adds a dependency with no meaningful benefit.

**Repository pattern.** Separating data access behind interfaces enables in-memory mocks for testing and PostgreSQL for production without changing business logic.

**Structured validation errors.** Field-level error details let clients build precise UI feedback without parsing error strings.

**Graceful shutdown.** In-flight requests complete before the server exits, preventing dropped connections during deployments.

## License

[MIT](LICENSE)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). PRs welcome — run `make all` before submitting.
