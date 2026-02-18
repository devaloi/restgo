# G01: restgo — Clean REST API in Go

**Catalog ID:** G01 | **Size:** M | **Language:** Go
**Repo name:** `restgo`
**One-liner:** A production-grade REST API in Go using only the standard library — clean architecture, PostgreSQL, JWT auth, middleware chain, and thorough tests.

---

## Why This Stands Out

- **stdlib-only HTTP** — no Gin, no Chi, no Echo. Shows you can build it from scratch with Go 1.22 routing
- **Repository pattern** — handler → service → repository with interfaces, easily testable
- **JWT authentication** — from-scratch token generation + middleware validation
- **Pagination, filtering, sorting** — real-world query patterns, not toy CRUD
- **Database migrations** — versioned SQL files applied on startup
- **Request validation** — struct tags + custom validator, clear error messages
- **Comprehensive middleware** — logging, auth, CORS, rate limit, request ID, recovery
- **Docker compose** — app + PostgreSQL + migrations, one-command startup

---

## Architecture

```
restgo/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: wire deps, run migrations, start server
├── internal/
│   ├── config/
│   │   └── config.go            # Env-based config with validation
│   ├── domain/
│   │   ├── user.go              # User entity + DTOs
│   │   ├── article.go           # Article entity + DTOs
│   │   └── errors.go            # Domain error types (NotFound, Conflict, Validation)
│   ├── handler/
│   │   ├── handler.go           # HTTP handler struct, route registration
│   │   ├── user.go              # User endpoints (register, login, profile)
│   │   ├── user_test.go
│   │   ├── article.go           # Article CRUD + list with pagination
│   │   ├── article_test.go
│   │   ├── response.go          # JSON response helpers (success, error, paginated)
│   │   └── middleware.go         # Auth middleware extracts user from JWT
│   ├── service/
│   │   ├── user.go              # User business logic (register, authenticate)
│   │   ├── user_test.go
│   │   ├── article.go           # Article business logic
│   │   └── article_test.go
│   ├── repository/
│   │   ├── user.go              # User repository interface
│   │   ├── article.go           # Article repository interface
│   │   ├── postgres/
│   │   │   ├── user.go          # PostgreSQL user repo
│   │   │   ├── user_test.go
│   │   │   ├── article.go       # PostgreSQL article repo
│   │   │   └── article_test.go
│   │   └── migrations/
│   │       ├── migrator.go      # Run .sql files in order
│   │       ├── 001_users.sql
│   │       └── 002_articles.sql
│   ├── auth/
│   │   ├── jwt.go               # JWT generation + validation (HS256)
│   │   └── jwt_test.go
│   ├── middleware/
│   │   ├── chain.go             # Middleware chaining helper
│   │   ├── logging.go           # Structured request logging
│   │   ├── cors.go              # CORS headers
│   │   ├── ratelimit.go         # Token bucket per IP
│   │   ├── recovery.go          # Panic recovery
│   │   ├── requestid.go         # X-Request-ID generation
│   │   └── middleware_test.go
│   └── validator/
│       ├── validator.go         # Request body validation
│       └── validator_test.go
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile
├── .env.example
├── .gitignore
├── .golangci.yml
├── LICENSE
└── README.md
```

---

## API Endpoints

### Auth
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/register` | No | Create account |
| POST | `/api/v1/login` | No | Get JWT token |
| GET | `/api/v1/me` | Yes | Current user profile |

### Articles
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/articles` | No | List with pagination, filter, sort |
| GET | `/api/v1/articles/:id` | No | Get single article |
| POST | `/api/v1/articles` | Yes | Create article |
| PUT | `/api/v1/articles/:id` | Yes | Update (owner only) |
| DELETE | `/api/v1/articles/:id` | Yes | Delete (owner only) |

### Query Parameters (list)
`?page=1&per_page=20&sort=created_at&order=desc&search=golang&author_id=5`

---

## Tech Stack

| Component | Choice |
|-----------|--------|
| Language | Go 1.22+ |
| HTTP | stdlib net/http (1.22 routing) |
| Database | PostgreSQL 15+ |
| Auth | JWT HS256 (hand-rolled, no lib) |
| Migrations | Embedded SQL files |
| Validation | Custom struct validator |
| Testing | stdlib + testcontainers-go for DB tests |
| Linting | golangci-lint |
| Container | Docker + docker-compose |

---

## Phased Build Plan

### Phase 1: Foundation

**1.1 — Project setup**
- `go mod init github.com/devaloi/restgo`
- Directory structure, Makefile, Dockerfile, docker-compose.yml, .env.example
- Makefile: build, test, lint, run, docker-up, docker-down, migrate

**1.2 — Config + database connection**
- Env-based config: DB_URL, JWT_SECRET, PORT, LOG_LEVEL
- PostgreSQL connection pool with ping check
- Graceful shutdown with context

**1.3 — Migration system**
- Embedded .sql files via `embed` package
- Track applied migrations in `schema_migrations` table
- Auto-run on startup
- Tests: apply, skip already applied, rollback on error

### Phase 2: Domain + Repository

**2.1 — Domain types**
- `User`: ID, email, password_hash, name, created_at
- `Article`: ID, title, body, author_id, created_at, updated_at
- Request/Response DTOs separate from entities
- Domain error types: `ErrNotFound`, `ErrConflict`, `ErrValidation`

**2.2 — User repository**
- Interface: `Create`, `GetByID`, `GetByEmail`, `Exists`
- PostgreSQL implementation
- Password hashing with bcrypt
- Tests with testcontainers-go (real PostgreSQL)

**2.3 — Article repository**
- Interface: `Create`, `GetByID`, `Update`, `Delete`, `List`
- `List` supports: pagination (LIMIT/OFFSET), sort field + direction, search (ILIKE), author filter
- Tests: CRUD, pagination, filtering, sorting

### Phase 3: Auth + Middleware

**3.1 — JWT auth**
- Generate: claims (user_id, email, exp, iat), HS256 signing
- Validate: parse token, verify signature, check expiry
- No external JWT library — shows algorithm understanding
- Tests: generate/validate round-trip, expired token, tampered token

**3.2 — Middleware chain**
- `Chain(handler, ...middleware)` helper
- Logging: method, path, status, duration, request_id
- CORS: configurable origins, methods, headers
- Rate limit: token bucket per client IP
- Recovery: catch panics, log stack, return 500
- Request ID: generate UUID, set header, add to context
- Auth: extract JWT from Authorization header, set user in context
- Tests for each middleware

### Phase 4: Services + Handlers

**4.1 — User service + handlers**
- `Register`: validate input, check duplicate email, hash password, create user, return JWT
- `Login`: find by email, verify password, return JWT
- `GetProfile`: return user from context
- Request validation: email format, password min length, required fields
- Tests: success paths, duplicate email, wrong password, validation errors

**4.2 — Article service + handlers**
- `Create`: validate, associate with auth user, insert
- `GetByID`: return or 404
- `Update`: verify ownership, update fields
- `Delete`: verify ownership, delete
- `List`: parse query params, call repo with filters, return paginated response
- Paginated response: `{ data: [], meta: { page, per_page, total, total_pages } }`
- Tests: full CRUD, ownership checks, pagination, filtering

### Phase 5: Validation + Polish

**5.1 — Request validator**
- Validate struct fields: required, min/max length, email format, enum values
- Return structured error: `{ errors: [{ field, message }] }`
- Tests: all validation rules

**5.2 — Route registration + server wiring**
- Register all routes with appropriate middleware
- Public routes: no auth middleware
- Protected routes: auth middleware applied
- Health check endpoint: `/health`

**5.3 — Integration tests**
- Full HTTP tests: register → login → create article → list → update → delete
- Test auth protection: 401 without token, 403 on wrong owner
- Test pagination and filtering end-to-end

**5.4 — README + documentation**
- Badges, install, quick start with Docker
- API reference table
- Architecture diagram
- Example curl commands
- Environment variables reference

---

## Commit Plan

1. `chore: scaffold project with Docker and config`
2. `feat: add migration system with embedded SQL`
3. `feat: add domain types and repository interfaces`
4. `feat: add PostgreSQL user repository`
5. `feat: add PostgreSQL article repository with pagination`
6. `feat: add JWT auth (hand-rolled HS256)`
7. `feat: add middleware chain (logging, CORS, rate limit, recovery)`
8. `feat: add user service and handlers (register, login, profile)`
9. `feat: add article service and handlers (CRUD + list)`
10. `feat: add request validation`
11. `test: add end-to-end integration tests`
12. `docs: add README with API reference and Docker setup`
