# Build restgo — Clean REST API in Go

You are building a **portfolio project** for a Senior AI Engineer's public GitHub. It must be impressive, clean, and production-grade. Read these docs before writing any code:

1. **`G01-go-rest-api.md`** — Complete project spec: architecture, phases, domain model, middleware design, commit plan. This is your primary blueprint. Follow it phase by phase.
2. **`github-portfolio.md`** — Portfolio goals and Definition of Done (Level 1 + Level 2). Understand the quality bar.
3. **`github-portfolio-checklist.md`** — Pre-publish checklist. Every item must pass before you're done.

---

## Instructions

### Read first, build second
Read all three docs completely before writing a single line of code. Understand the clean architecture layers, the PostgreSQL repository pattern, and the JWT auth design.

### Follow the phases in order
The project spec has 5 phases. Do them in order:
1. **Foundation** — project setup, config, database connection, migration system
2. **Domain + Repository** — domain types, PostgreSQL repository with raw SQL (no ORM), repository interface
3. **Auth + Middleware** — JWT auth (hand-rolled, no library), middleware chain (logging, auth, CORS, recovery, request ID)
4. **Services + Handlers** — business logic layer, HTTP handlers, router setup
5. **Validation + Polish** — input validation, comprehensive tests, refactor, README

### Commit frequently
Follow the commit plan in the spec. Use **conventional commits**. Each commit should be a logical unit.

### Quality non-negotiables
- **stdlib HTTP only.** No Gin, no Chi, no Echo. Use `net/http` with Go 1.22+ method routing. This is the flagship Go project — it shows stdlib mastery.
- **Hand-rolled JWT.** No `golang-jwt` library. Implement HS256 signing and verification from `crypto/hmac` + `crypto/sha256`. This shows you understand the standard.
- **Raw SQL, no ORM.** Use `database/sql` with `lib/pq`. Write clean, parameterized SQL. This is deliberate — shows database fluency.
- **Clean architecture.** Handler → Service → Repository. Each layer has an interface. Tests mock at the boundary.
- **PostgreSQL with migrations.** Embedded SQL migration files, version tracking, up/down support.
- **Middleware from scratch.** Every middleware piece built by hand — no libraries.
- **Tests at every layer.** Repository tests with test database, service tests with mocked repo, handler tests with httptest.
- **No Docker for the app.** PostgreSQL can run in Docker for local dev, but the app itself is `go build` + `go run`. Include a note in README about running PostgreSQL locally.

### What NOT to do
- Don't use any HTTP framework. stdlib only.
- Don't use an ORM (GORM, sqlx, etc.). Raw `database/sql`.
- Don't use a JWT library. Hand-roll the HS256 implementation.
- Don't skip the migration system. No ad-hoc `CREATE TABLE` in code.
- Don't commit database credentials. Use environment variables + `.env.example`.
- Don't leave `// TODO` or `// FIXME` comments anywhere.

---

## GitHub Username

The GitHub username is **devaloi**. For Go module paths, use `github.com/devaloi/restgo`. All internal imports must use this module path.

## Start

Read the three docs. Then begin Phase 1 from `G01-go-rest-api.md`.
