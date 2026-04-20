# Phase 1 — Auth Foundation + Admin Shell Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the smallest vertical slice that proves the whole stack is wired — operator can log into the admin at `http://localhost:5173`, land on a protected dashboard, and log out. This is the foundation every later phase (waitlist, geckos, data visualizer, dashboard stats) depends on.

**Architecture:** The Go backend exposes `POST /api/auth/login` (bcrypt password check → issue JWT) and `GET /api/auth/me` (JWT-protected, returns the logged-in admin). The Vue admin uses shadcn-vue primitives themed with brand tokens, vue-router with `requiresAuth` guards, a Pinia auth store that persists the JWT in `localStorage`, and an axios client that attaches the bearer token to every outgoing request. A one-shot `gekko-seed` CLI bootstraps the first admin user so login can actually succeed.

**Tech Stack:**
- Backend: Go 1.25 · chi v5 · pgx v5 · sqlc · goose · zerolog · golang-jwt/v5 · bcrypt · testify
- Admin: Vue 3.5 · Vite 8 · TypeScript · Tailwind 4 · shadcn-vue · vue-router · Pinia · axios · @tanstack/vue-query · vue-sonner · vee-validate · zod

**Out of scope for Phase 1** (explicitly deferred to future phases):
- Waitlist (public submit + admin list + CSV) — Phase 2
- Geckos CRUD + photo upload — Phase 3
- Data visualizer — Phase 4
- Dashboard stats + Three.js `AnimatedBackground` on login — Phase 5
- Storefront scaffolding + prod deploy — Phase 6
- Password reset / password change flow (operator bootstraps via seed CLI)
- Refresh tokens (single short-lived JWT is fine for a solo operator)

**Definition of done** — all of:
1. `make db-up && make migrate` brings Postgres up and applies both existing migrations.
2. `go run ./cmd/gekko-seed -email x@y.z -password ****** -name "Zen"` creates an admin user.
3. `make dev-backend` serves `GET /health` → `{"status":"healthy"}`, and `POST /api/auth/login` with correct creds returns `{ "token": "...", "admin": { "id":…, "email":…, "name":… } }`.
4. `make dev-admin` serves the admin at `http://localhost:5173` with `DM Serif Display` + `Inter` fonts and brand colors.
5. From a fresh browser session: visiting `/` redirects to `/login`, entering correct creds lands on `/` with the admin's name in the top bar, clicking **Logout** redirects back to `/login` and clears the token.
6. Visiting `/` again after logout redirects back to `/login` (no zombie session).
7. All backend unit tests pass (`go test ./...`).
8. All frontend unit tests pass (`bun --cwd apps/admin run test`).

---

## Prerequisites (verify before Task 1)

- Postgres is reachable: `make db-up` starts container `gekko_db` on `localhost:5433`.
- Migrations apply clean: `make migrate` shows `OK 20260420000002_admin_users.sql`.
- Go 1.25, bun ≥ 1.1, Docker, sqlc, goose, air installed (see `docs/HANDOFF-2026-04-20.md` §0).

## File structure (what this plan creates or modifies)

### Backend (`backend/`)
```
backend/
├── cmd/
│   ├── gekko/main.go            ← MODIFY — wire config, routes, middleware
│   └── gekko-seed/main.go       ← NEW — CLI to seed the first admin
├── internal/
│   ├── config/
│   │   ├── config.go            ← NEW — typed env loader
│   │   └── config_test.go       ← NEW — tests
│   ├── auth/
│   │   ├── password.go          ← NEW — bcrypt hash + verify
│   │   ├── password_test.go     ← NEW
│   │   ├── jwt.go               ← NEW — issue + verify + claims struct
│   │   └── jwt_test.go          ← NEW
│   ├── http/
│   │   ├── auth.go              ← NEW — login + me handlers + router mount
│   │   ├── auth_test.go         ← NEW — integration tests via httptest
│   │   └── middleware.go        ← NEW — RequireAuth + context key helpers
│   └── queries/                 ← unchanged (admin_users.sql already exists)
├── .env.local                   ← NEW (gitignored)
└── .air.toml                    ← NEW (if missing)
```

### Admin (`apps/admin/`)
```
apps/admin/
├── components.json              ← NEW — shadcn-vue config
├── vite.config.ts               ← MODIFY — Tailscale-friendly host, aliases
├── index.html                   ← MODIFY — Google Fonts, title
├── src/
│   ├── style.css                ← REPLACE — brand theme (CSS custom props + @theme)
│   ├── main.ts                  ← REPLACE — router, pinia, vue-query, sonner
│   ├── App.vue                  ← REPLACE — <RouterView /> + <Toaster />
│   ├── router/
│   │   └── index.ts             ← NEW — routes + guards
│   ├── lib/
│   │   ├── api.ts               ← NEW — axios client + interceptors
│   │   └── utils.ts             ← NEW — shadcn-vue `cn()` helper (created by shadcn init)
│   ├── stores/
│   │   ├── auth.ts              ← NEW — Pinia auth store
│   │   └── auth.spec.ts         ← NEW — Vitest test
│   ├── views/
│   │   ├── LoginView.vue        ← NEW — email + password form
│   │   ├── DashboardView.vue    ← NEW — placeholder
│   │   └── NotFoundView.vue     ← NEW
│   ├── layouts/
│   │   └── AppShell.vue         ← NEW — sidebar + top bar wrapper
│   └── components/ui/…          ← NEW (generated by shadcn-vue add)
├── .env.local                   ← NEW (gitignored)
├── vitest.config.ts             ← NEW
└── package.json                 ← MODIFY — add `test` script
```

### Makefile (repo root)
Add a `seed` target and a `test` target (fan-out to backend + admin).

---

## Task 0: Environment bootstrap

**Files:**
- Create: `backend/.env.local`
- Create: `apps/admin/.env.local`
- Create: `backend/.air.toml` (if missing)
- Modify: `.gitignore`

- [ ] **Step 1: Verify Postgres + migrations are up**

```bash
make db-up
make migrate
make migrate-status
# Expected: both migrations show "Applied At"
```

- [ ] **Step 2: Generate a JWT secret**

```bash
openssl rand -hex 32
# Copy the 64-char hex string
```

- [ ] **Step 3: Create `backend/.env.local`**

Paste the JWT secret where indicated.

```
DB_URL=postgres://gekko:gekko@localhost:5433/gekko?sslmode=disable
PORT=8420
JWT_SECRET=PASTE_THE_HEX_FROM_STEP_2_HERE
JWT_TTL_HOURS=24
BASE_CURRENCY=USD
TIMEZONE=Asia/Phnom_Penh
LOG_LEVEL=debug
UPLOAD_DIR=./uploads
PUBLIC_UPLOAD_URL=http://localhost:8420/uploads
CORS_ORIGINS=http://localhost:5173,http://localhost:5174
```

- [ ] **Step 4: Create `apps/admin/.env.local`**

```
VITE_API_BASE_URL=http://localhost:8420
VITE_APP_NAME=Zenetic Gekkos Admin
```

- [ ] **Step 5: Confirm `.gitignore` excludes these**

Ensure the repo root `.gitignore` contains (append any that are missing):

```
backend/.env.local
backend/tmp/
backend/uploads/*
!backend/uploads/.gitkeep
apps/admin/.env.local
node_modules/
dist/
.DS_Store
```

- [ ] **Step 6: Create `backend/.air.toml`**

```toml
root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/gekko"
  delay = 500
  exclude_dir = ["tmp", "uploads", "internal/db"]
  exclude_regex = ["_test\\.go"]
  include_ext = ["go", "tpl", "tmpl", "html", "sql"]
  log = "build-errors.log"
  stop_on_error = true

[log]
  time = false

[misc]
  clean_on_exit = false
```

- [ ] **Step 7: Commit**

```bash
git add backend/.air.toml .gitignore
git commit -m "chore: add .air.toml and gitignore env/tmp/uploads"
```

---

## Task 1: Config loader (TDD)

**Files:**
- Create: `backend/internal/config/config.go`
- Test: `backend/internal/config/config_test.go`

- [ ] **Step 1: Add dev deps for testing**

```bash
cd backend
go get github.com/stretchr/testify@v1.8.4
go get github.com/golang-jwt/jwt/v5@v5.2.1
```

- [ ] **Step 2: Write the failing test**

Create `backend/internal/config/config_test.go`:

```go
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_happy(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@localhost:5433/d?sslmode=disable")
	t.Setenv("PORT", "8420")
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	t.Setenv("JWT_TTL_HOURS", "48")
	t.Setenv("CORS_ORIGINS", "http://a.test,http://b.test")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("UPLOAD_DIR", "/tmp/u")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres://u:p@localhost:5433/d?sslmode=disable", cfg.DBURL)
	assert.Equal(t, "8420", cfg.Port)
	assert.Len(t, cfg.JWTSecret, 64)
	assert.Equal(t, 48, cfg.JWTTTLHours)
	assert.Equal(t, []string{"http://a.test", "http://b.test"}, cfg.CORSOrigins)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "/tmp/u", cfg.UploadDir)
}

func TestLoad_defaults(t *testing.T) {
	t.Setenv("DB_URL", "postgres://u:p@h/d")
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("PORT", "")
	t.Setenv("JWT_TTL_HOURS", "")
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("UPLOAD_DIR", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Equal(t, "8420", cfg.Port)
	assert.Equal(t, 24, cfg.JWTTTLHours)
	assert.Equal(t, []string{}, cfg.CORSOrigins)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "./uploads", cfg.UploadDir)
}

func TestLoad_missing_required(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("JWT_SECRET", "")

	_, err := Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DB_URL")
}
```

- [ ] **Step 3: Run the test — expect compile failure**

```bash
cd backend
go test ./internal/config/...
# Expected: no Go files / package not found
```

- [ ] **Step 4: Write the implementation**

Create `backend/internal/config/config.go`:

```go
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBURL        string
	Port         string
	JWTSecret    string
	JWTTTLHours  int
	CORSOrigins  []string
	LogLevel     string
	UploadDir    string
}

func Load() (*Config, error) {
	cfg := &Config{
		DBURL:       os.Getenv("DB_URL"),
		Port:        envOr("PORT", "8420"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTLHours: envIntOr("JWT_TTL_HOURS", 24),
		CORSOrigins: splitCSV(os.Getenv("CORS_ORIGINS")),
		LogLevel:    envOr("LOG_LEVEL", "info"),
		UploadDir:   envOr("UPLOAD_DIR", "./uploads"),
	}

	if cfg.DBURL == "" {
		return nil, errors.New("DB_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(s string) []string {
	out := []string{}
	if s == "" {
		return out
	}
	for _, part := range strings.Split(s, ",") {
		p := strings.TrimSpace(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
```

- [ ] **Step 5: Run the test — expect pass**

```bash
go test ./internal/config/... -v
# Expected: PASS on all three tests
```

- [ ] **Step 6: Commit**

```bash
git add backend/internal/config backend/go.mod backend/go.sum
git commit -m "feat(backend): typed config loader with validation"
```

---

## Task 2: Password hashing helper (TDD)

**Files:**
- Create: `backend/internal/auth/password.go`
- Test: `backend/internal/auth/password_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/auth/password_test.go`:

```go
package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerify(t *testing.T) {
	plain := "hunter2-longer-than-8-chars"
	hash, err := HashPassword(plain)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, plain, hash)

	assert.True(t, VerifyPassword(hash, plain))
	assert.False(t, VerifyPassword(hash, "wrong-password"))
}

func TestHashPassword_tooShort(t *testing.T) {
	_, err := HashPassword("short")
	require.Error(t, err)
}

func TestVerifyPassword_malformedHash(t *testing.T) {
	assert.False(t, VerifyPassword("not-a-bcrypt-hash", "anything"))
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test ./internal/auth/...
# Expected: package has no files or undefined symbols
```

- [ ] **Step 3: Write the implementation**

Create `backend/internal/auth/password.go`:

```go
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLen = 8

func HashPassword(plain string) (string, error) {
	if len(plain) < minPasswordLen {
		return "", errors.New("password must be at least 8 characters")
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./internal/auth/... -v -run TestHash -run TestVerify
# Expected: PASS
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/password.go backend/internal/auth/password_test.go
git commit -m "feat(backend): bcrypt password hash + verify helpers"
```

---

## Task 3: JWT issue + verify (TDD)

**Files:**
- Create: `backend/internal/auth/jwt.go`
- Test: `backend/internal/auth/jwt_test.go`

- [ ] **Step 1: Write the failing test**

Create `backend/internal/auth/jwt_test.go`:

```go
package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIssueAndVerify(t *testing.T) {
	signer := NewJWTSigner("test-secret-xyz", time.Hour)

	tok, err := signer.Issue(42, "admin@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	claims, err := signer.Verify(tok)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.AdminID)
	assert.Equal(t, "admin@example.com", claims.Email)
}

func TestVerify_wrongSecret(t *testing.T) {
	a := NewJWTSigner("secret-a", time.Hour)
	b := NewJWTSigner("secret-b", time.Hour)

	tok, err := a.Issue(1, "x@y.z")
	require.NoError(t, err)

	_, err = b.Verify(tok)
	require.Error(t, err)
}

func TestVerify_expired(t *testing.T) {
	signer := NewJWTSigner("s", -time.Second) // negative TTL = already expired
	tok, err := signer.Issue(1, "x@y.z")
	require.NoError(t, err)

	_, err = signer.Verify(tok)
	require.Error(t, err)
}

func TestVerify_garbage(t *testing.T) {
	signer := NewJWTSigner("s", time.Hour)
	_, err := signer.Verify("not.a.jwt")
	require.Error(t, err)
}
```

- [ ] **Step 2: Run — expect fail**

```bash
go test ./internal/auth/... -run TestIssue -run TestVerify
# Expected: NewJWTSigner undefined etc.
```

- [ ] **Step 3: Write the implementation**

Create `backend/internal/auth/jwt.go`:

```go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	AdminID int64  `json:"sub_id"`
	Email   string `json:"email"`
	jwt.RegisteredClaims
}

type JWTSigner struct {
	secret []byte
	ttl    time.Duration
}

func NewJWTSigner(secret string, ttl time.Duration) *JWTSigner {
	return &JWTSigner{secret: []byte(secret), ttl: ttl}
}

func (s *JWTSigner) Issue(adminID int64, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		AdminID: adminID,
		Email:   email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "gekko-backend",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(s.secret)
}

func (s *JWTSigner) Verify(raw string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
```

- [ ] **Step 4: Run — expect pass**

```bash
go test ./internal/auth/... -v
# Expected: PASS on all password + jwt tests
```

- [ ] **Step 5: Commit**

```bash
git add backend/internal/auth/jwt.go backend/internal/auth/jwt_test.go backend/go.mod backend/go.sum
git commit -m "feat(backend): HS256 JWT issue + verify with typed claims"
```

---

## Task 4: `gekko-seed` CLI — bootstrap first admin

**Files:**
- Create: `backend/cmd/gekko-seed/main.go`
- Modify: `Makefile` (add `seed` target)

- [ ] **Step 1: Write the CLI**

Create `backend/cmd/gekko-seed/main.go`:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func main() {
	_ = godotenv.Load(".env.local")

	email := flag.String("email", "", "admin email (required)")
	password := flag.String("password", "", "admin password, ≥8 chars (required)")
	name := flag.String("name", "", "display name (optional)")
	flag.Parse()

	if *email == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "usage: gekko-seed -email <email> -password <pw> [-name <name>]")
		os.Exit(2)
	}

	hash, err := auth.HashPassword(*password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "password error: %v\n", err)
		os.Exit(1)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DB_URL is required (set in backend/.env.local)")
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	q := db.New(pool)

	var displayName *string
	if *name != "" {
		displayName = name
	}

	// Check if an admin with this email already exists.
	existing, err := q.GetAdminByEmail(ctx, *email)
	if err == nil {
		fmt.Printf("admin already exists: id=%d email=%s\n", existing.ID, existing.Email)
		os.Exit(0)
	}

	created, err := q.CreateAdmin(ctx, db.CreateAdminParams{
		Email:        *email,
		PasswordHash: hash,
		Name:         nullString(displayName),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create admin: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("created admin id=%d email=%s\n", created.ID, created.Email)
}
```

> **Note on `nullString`**: sqlc generated code uses `pgtype.Text` for nullable VARCHARs. Inspect the actual type in `backend/internal/db/models.go` (field `Name`). If it is `pgtype.Text`, add this helper at the bottom of `main.go`:
>
> ```go
> import "github.com/jackc/pgx/v5/pgtype"
>
> func nullString(s *string) pgtype.Text {
>   if s == nil {
>     return pgtype.Text{Valid: false}
>   }
>   return pgtype.Text{String: *s, Valid: true}
> }
> ```
>
> If sqlc generated `sql.NullString` instead, swap the helper to return that type (`return sql.NullString{String: *s, Valid: true}`).

- [ ] **Step 2: Compile-check the CLI**

```bash
cd backend
go build ./cmd/gekko-seed
# Expected: produces ./gekko-seed binary, no build errors
rm gekko-seed
```

- [ ] **Step 3: Add `seed` target to root Makefile**

Open `Makefile` at the repo root. Append (keeping the existing content intact):

```makefile
seed:  ## Seed the first admin (ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=Zen make seed)
	cd backend && go run ./cmd/gekko-seed \
	  -email "$(ADMIN_EMAIL)" -password "$(ADMIN_PASSWORD)" -name "$(ADMIN_NAME)"
```

Also update the `.PHONY` line at the top to include `seed`:

```makefile
.PHONY: help db-up db-down db-logs migrate migrate-status migrate-down sqlc dev-backend dev-admin dev-storefront seed
```

- [ ] **Step 4: Run the seed**

```bash
cd ~/dev/project_gekko
ADMIN_EMAIL=zen@zeneticgekkos.com ADMIN_PASSWORD=change-me-later ADMIN_NAME=Zen make seed
# Expected: "created admin id=1 email=zen@zeneticgekkos.com"

# Re-run — should be idempotent
ADMIN_EMAIL=zen@zeneticgekkos.com ADMIN_PASSWORD=change-me-later make seed
# Expected: "admin already exists: id=1 email=zen@zeneticgekkos.com"
```

- [ ] **Step 5: Verify in Postgres**

```bash
PGPASSWORD=gekko psql -h localhost -p 5433 -U gekko -d gekko -c 'SELECT id, email, name FROM admin_users;'
# Expected: one row with your email
```

- [ ] **Step 6: Commit**

```bash
git add backend/cmd/gekko-seed Makefile
git commit -m "feat(backend): gekko-seed CLI + make seed target"
```

---

## Task 5: Login + `/me` handlers (TDD, integration-style)

**Files:**
- Create: `backend/internal/http/auth.go`
- Create: `backend/internal/http/middleware.go`
- Test: `backend/internal/http/auth_test.go`

We test these together via `httptest` + a real Postgres pool (the test DB is the same dev DB — tests use a unique email so they don't collide with the seeded admin).

- [ ] **Step 1: Write the middleware skeleton first (empty types we'll flesh out in Task 6)**

Create `backend/internal/http/middleware.go`:

```go
package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
)

type ctxKey string

const ctxAdminKey ctxKey = "admin"

func RequireAuth(signer *auth.JWTSigner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			tok := strings.TrimPrefix(h, "Bearer ")
			claims, err := signer.Verify(tok)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxAdminKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func claimsFrom(r *http.Request) (*auth.Claims, bool) {
	c, ok := r.Context().Value(ctxAdminKey).(*auth.Claims)
	return c, ok
}
```

- [ ] **Step 2: Write the failing tests**

Create `backend/internal/http/auth_test.go`:

```go
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// testSetup returns a router, signer, and a freshly-created admin for the test.
// It writes to the dev DB under a test-only email and cleans up afterwards.
func testSetup(t *testing.T) (http.Handler, *auth.JWTSigner, string, string) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := "test+" + time.Now().Format("150405.000000") + "@example.com"
	plain := "test-password-123"
	hash, err := auth.HashPassword(plain)
	require.NoError(t, err)

	q := db.New(pool)
	created, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email:        email,
		PasswordHash: hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM admin_users WHERE id = $1", created.ID)
	})

	signer := auth.NewJWTSigner("test-secret", time.Hour)
	router := NewAuthRouter(pool, signer)
	return router, signer, email, plain
}

func TestLogin_happyPath(t *testing.T) {
	r, _, email, password := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.NotEmpty(t, got["token"])
	admin, _ := got["admin"].(map[string]any)
	assert.Equal(t, email, admin["email"])
}

func TestLogin_wrongPassword(t *testing.T) {
	r, _, email, _ := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": email, "password": "nope"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLogin_unknownEmail(t *testing.T) {
	r, _, _, _ := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": "nobody@example.com", "password": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	// Return 401 (not 404) to avoid email enumeration.
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestMe_withToken(t *testing.T) {
	r, signer, email, password := testSetup(t)

	// Log in first to get a real token.
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	r.ServeHTTP(loginRR, loginReq)
	require.Equal(t, http.StatusOK, loginRR.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &payload))
	tok := payload["token"].(string)
	_ = signer // keep available for direct-signer tests later

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+tok)
	meRR := httptest.NewRecorder()
	r.ServeHTTP(meRR, meReq)

	require.Equal(t, http.StatusOK, meRR.Code, "body=%s", meRR.Body.String())
	var me map[string]any
	require.NoError(t, json.Unmarshal(meRR.Body.Bytes(), &me))
	assert.Equal(t, email, me["email"])
}

func TestMe_missingToken(t *testing.T) {
	r, _, _, _ := testSetup(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
```

- [ ] **Step 3: Run — expect fail (NewAuthRouter undefined)**

```bash
cd backend
go test ./internal/http/...
# Expected: undefined: NewAuthRouter
```

- [ ] **Step 4: Implement `auth.go`**

Create `backend/internal/http/auth.go`:

```go
package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

type authDeps struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	signer *auth.JWTSigner
}

// NewAuthRouter mounts /api/auth/* (login + me).
func NewAuthRouter(pool *pgxpool.Pool, signer *auth.JWTSigner) http.Handler {
	d := &authDeps{pool: pool, q: db.New(pool), signer: signer}

	r := chi.NewRouter()
	r.Post("/api/auth/login", d.login)

	protected := chi.NewRouter()
	protected.Use(RequireAuth(signer))
	protected.Get("/api/auth/me", d.me)

	// Merge protected routes into the main tree.
	r.Mount("/", protected)
	return r
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminDTO struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type loginResp struct {
	Token string   `json:"token"`
	Admin adminDTO `json:"admin"`
}

func (d *authDeps) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	row, err := d.q.GetAdminByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	if !auth.VerifyPassword(row.PasswordHash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	tok, err := d.signer.Issue(int64(row.ID), row.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token issue failed"})
		return
	}

	writeJSON(w, http.StatusOK, loginResp{
		Token: tok,
		Admin: adminDTO{ID: int64(row.ID), Email: row.Email, Name: nullToString(row.Name)},
	})
}

func (d *authDeps) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}
	row, err := d.q.GetAdminByID(r.Context(), int32(claims.AdminID))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin gone"})
		return
	}
	writeJSON(w, http.StatusOK, adminDTO{ID: int64(row.ID), Email: row.Email, Name: nullToString(row.Name)})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
```

- [ ] **Step 5: Add the `nullToString` helper (matches the sqlc-generated nullable type)**

If sqlc generated `pgtype.Text` for the `name` column (check `backend/internal/db/models.go`), add this to `backend/internal/http/auth.go` below `writeJSON`:

```go
import "github.com/jackc/pgx/v5/pgtype"

func nullToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
```

If sqlc generated `sql.NullString` instead, swap the helper (and import `"database/sql"`):

```go
import "database/sql"

func nullToString(t sql.NullString) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
```

Use whichever one compiles. Do **not** keep both.

- [ ] **Step 6: Run the tests**

```bash
cd backend
go test ./internal/http/... -v
# Expected: PASS on TestLogin_happyPath, TestLogin_wrongPassword,
# TestLogin_unknownEmail, TestMe_withToken, TestMe_missingToken
```

> **If `CreateAdminParams.Name` complains about type mismatch**: inspect `db.CreateAdminParams` in the generated code. The `testSetup` helper above does not pass `Name` — if the generated param is a non-nullable type, either pass the zero value explicitly or regenerate sqlc so `name` is optional. The migration defines `name VARCHAR(120)` (nullable), so sqlc should already treat it as nullable.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/http/auth.go backend/internal/http/middleware.go backend/internal/http/auth_test.go
git commit -m "feat(backend): auth router — POST /login + GET /me with JWT guard"
```

---

## Task 6: Wire routes into `main.go`

**Files:**
- Modify: `backend/cmd/gekko/main.go`

- [ ] **Step 1: Replace `main.go` with the wired version**

Overwrite `backend/cmd/gekko/main.go` with:

```go
package main

import (
	"context"
	nethttp "net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/config"
	apihttp "github.com/jxnhoongz/project_gekko/backend/internal/http"
)

func main() {
	_ = godotenv.Load(".env.local")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config load failed")
	}

	if cfg.LogLevel == "debug" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.Kitchen})
	}
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("db connection failed")
	}
	defer pool.Close()

	signer := auth.NewJWTSigner(cfg.JWTSecret, time.Duration(cfg.JWTTTLHours)*time.Hour)

	r := chi.NewRouter()
	r.Use(chimw.RequestID, chimw.RealIP, chimw.Logger, chimw.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w nethttp.ResponseWriter, req *nethttp.Request) {
		if err := pool.Ping(req.Context()); err != nil {
			nethttp.Error(w, `{"status":"db down"}`, nethttp.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	// Static file server for v1 uploads.
	fs := nethttp.StripPrefix("/uploads/", nethttp.FileServer(nethttp.Dir(cfg.UploadDir)))
	r.Handle("/uploads/*", fs)

	// Auth routes (login public, /me protected).
	r.Mount("/", apihttp.NewAuthRouter(pool, signer))

	log.Info().Msgf("listening on :%s", cfg.Port)
	if err := nethttp.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal().Err(err).Send()
	}
}
```

- [ ] **Step 2: Build and smoke-test**

```bash
cd backend
go build ./cmd/gekko
go run ./cmd/gekko &
BACKEND_PID=$!
sleep 1

# Health
curl -sS http://localhost:8420/health
# Expected: {"status":"healthy"}

# Login (use the seeded credentials from Task 4)
curl -sS -X POST http://localhost:8420/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"zen@zeneticgekkos.com","password":"change-me-later"}'
# Expected: {"token":"eyJ...","admin":{"id":1,"email":"zen@zeneticgekkos.com","name":"Zen"}}

# /me with the returned token
TOKEN="<paste token from previous>"
curl -sS http://localhost:8420/api/auth/me -H "Authorization: Bearer $TOKEN"
# Expected: {"id":1,"email":"zen@zeneticgekkos.com","name":"Zen"}

kill $BACKEND_PID
```

- [ ] **Step 3: Run all backend tests**

```bash
go test ./...
# Expected: all PASS
```

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/gekko/main.go
git commit -m "feat(backend): wire config + auth routes into main"
```

---

## Task 7: Admin — initialize shadcn-vue with brand theme

**Files:**
- Create: `apps/admin/components.json`
- Create: `apps/admin/src/lib/utils.ts`
- Replace: `apps/admin/src/style.css`
- Modify: `apps/admin/vite.config.ts`
- Modify: `apps/admin/index.html`

The admin already has `vue@3.5`, `@tanstack/vue-query`, `pinia`, `vee-validate`, `zod`, `axios`, `vue-sonner`, `lucide-vue-next`, Tailwind 4 via `@tailwindcss/vite`. We add shadcn-vue primitives on top.

- [ ] **Step 1: Install shadcn-vue dependencies**

```bash
cd apps/admin
bun add reka-ui class-variance-authority clsx tailwind-merge
bun add -D @types/node tw-animate-css
```

- [ ] **Step 2: Create `components.json`**

Create `apps/admin/components.json`:

```json
{
  "$schema": "https://shadcn-vue.com/schema.json",
  "style": "new-york",
  "typescript": true,
  "tailwind": {
    "config": "",
    "css": "src/style.css",
    "baseColor": "neutral",
    "cssVariables": true
  },
  "aliases": {
    "components": "@/components",
    "composables": "@/composables",
    "utils": "@/lib/utils",
    "ui": "@/components/ui",
    "lib": "@/lib"
  },
  "iconLibrary": "lucide"
}
```

- [ ] **Step 3: Create the `cn` helper**

Create `apps/admin/src/lib/utils.ts`:

```ts
import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

- [ ] **Step 4: Replace `apps/admin/src/style.css` with the brand theme**

This uses Tailwind 4's `@theme` CSS-first config. Brand tokens come from the `gekko-design` skill.

```css
@import "tailwindcss";
@import "tw-animate-css";

@custom-variant dark (&:is(.dark *));

@theme {
  /* Brand cream (page backgrounds, surfaces) */
  --color-brand-cream-50: #faf8f3;
  --color-brand-cream-100: #f4f0e4;
  --color-brand-cream-200: #e9e1c9;
  --color-brand-cream-300: #d9cdac;
  --color-brand-cream-400: #c4b48e;
  --color-brand-cream-500: #ad9a6f;
  --color-brand-cream-600: #8f7e57;
  --color-brand-cream-700: #726447;
  --color-brand-cream-800: #564c37;
  --color-brand-cream-900: #3a3326;

  /* Brand gold (primary actions, accents) */
  --color-brand-gold-50: #fdf8ec;
  --color-brand-gold-100: #fbefd0;
  --color-brand-gold-200: #f6dc9a;
  --color-brand-gold-300: #efc262;
  --color-brand-gold-400: #e8a83a;
  --color-brand-gold-500: #d08b1f;
  --color-brand-gold-600: #b06c12;
  --color-brand-gold-700: #8b5211;
  --color-brand-gold-800: #6a3f10;
  --color-brand-gold-900: #4b2c0d;

  /* Brand dark (text, secondary actions) */
  --color-brand-dark-50: #f5f4f2;
  --color-brand-dark-100: #e8e5df;
  --color-brand-dark-200: #cfc9bf;
  --color-brand-dark-300: #ada498;
  --color-brand-dark-400: #847b6f;
  --color-brand-dark-500: #655c52;
  --color-brand-dark-600: #4e463f;
  --color-brand-dark-700: #3b342e;
  --color-brand-dark-800: #2a241f;
  --color-brand-dark-900: #1c1814;
  --color-brand-dark-950: #110e0b;

  --font-sans: "Inter", ui-sans-serif, system-ui, sans-serif;
  --font-serif: "DM Serif Display", ui-serif, serif;
}

/* shadcn-vue semantic tokens mapped to brand palette */
@layer base {
  :root {
    --background: var(--color-brand-cream-50);
    --foreground: var(--color-brand-dark-950);
    --card: var(--color-brand-cream-50);
    --card-foreground: var(--color-brand-dark-950);
    --popover: var(--color-brand-cream-50);
    --popover-foreground: var(--color-brand-dark-950);
    --primary: var(--color-brand-gold-600);
    --primary-foreground: #ffffff;
    --secondary: var(--color-brand-cream-100);
    --secondary-foreground: var(--color-brand-dark-950);
    --muted: var(--color-brand-cream-100);
    --muted-foreground: var(--color-brand-dark-600);
    --accent: var(--color-brand-gold-100);
    --accent-foreground: var(--color-brand-gold-800);
    --destructive: #c2410c;
    --destructive-foreground: #ffffff;
    --border: var(--color-brand-cream-300);
    --input: var(--color-brand-cream-300);
    --ring: var(--color-brand-gold-500);
    --radius: 0.625rem;
  }

  * {
    border-color: var(--border);
  }

  html,
  body,
  #app {
    height: 100%;
  }

  body {
    background: var(--color-brand-cream-50);
    color: var(--color-brand-dark-950);
    font-family: var(--font-sans);
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    margin: 0;
  }

  h1, h2, h3, h4 {
    font-family: var(--font-serif);
    color: var(--color-brand-dark-950);
    font-weight: 400;
    letter-spacing: -0.01em;
  }
}
```

- [ ] **Step 5: Update `apps/admin/vite.config.ts`**

```ts
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    port: 5173,
    host: '0.0.0.0', // reachable over Tailscale; see docs/tailscale-remote-access.md
  },
});
```

- [ ] **Step 6: Update `apps/admin/index.html`**

Replace the `<head>` contents and `<title>` so Google Fonts load:

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Zenetic Gekkos Admin</title>
    <link rel="preconnect" href="https://fonts.googleapis.com" />
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
    <link
      href="https://fonts.googleapis.com/css2?family=DM+Serif+Display&family=Inter:wght@400;500;600;700&display=swap"
      rel="stylesheet"
    />
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 7: Add shadcn-vue primitives needed for phase 1**

```bash
cd apps/admin
bunx shadcn-vue@latest add button card input label sonner
# If prompted about overwriting files, accept.
```

This creates `apps/admin/src/components/ui/{button,card,input,label,sonner}/` with Vue components.

- [ ] **Step 8: Sanity check the dev server boots**

```bash
cd apps/admin
bun run dev
# Open http://localhost:5173 in a browser.
# Expected: no console errors, default create-vite template still renders (we replace it next).
# Ctrl+C to stop.
```

- [ ] **Step 9: Commit**

```bash
git add apps/admin
git commit -m "feat(admin): shadcn-vue baseline + brand theme + fonts"
```

---

## Task 8: Admin — axios client + Pinia auth store (TDD the store)

**Files:**
- Create: `apps/admin/src/lib/api.ts`
- Create: `apps/admin/src/stores/auth.ts`
- Create: `apps/admin/src/stores/auth.spec.ts`
- Create: `apps/admin/vitest.config.ts`
- Modify: `apps/admin/package.json` (add `test` script)

- [ ] **Step 1: Create the axios client**

Create `apps/admin/src/lib/api.ts`:

```ts
import axios, { AxiosError, type InternalAxiosRequestConfig } from 'axios';

const TOKEN_STORAGE_KEY = 'gekko.admin.token';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8420',
  withCredentials: false,
});

api.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
  const tok = localStorage.getItem(TOKEN_STORAGE_KEY);
  if (tok) {
    cfg.headers.set('Authorization', `Bearer ${tok}`);
  }
  return cfg;
});

api.interceptors.response.use(
  (res) => res,
  (err: AxiosError) => {
    if (err.response?.status === 401) {
      // Surface globally so the auth store can react.
      window.dispatchEvent(new CustomEvent('gekko:unauthorized'));
    }
    return Promise.reject(err);
  },
);

export function storeToken(tok: string) {
  localStorage.setItem(TOKEN_STORAGE_KEY, tok);
}

export function clearToken() {
  localStorage.removeItem(TOKEN_STORAGE_KEY);
}

export function readToken(): string | null {
  return localStorage.getItem(TOKEN_STORAGE_KEY);
}
```

- [ ] **Step 2: Add Vitest + test deps**

```bash
cd apps/admin
bun add -D vitest @vitest/ui jsdom @pinia/testing
```

- [ ] **Step 3: Create `vitest.config.ts`**

```ts
import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import path from 'path';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  test: {
    environment: 'jsdom',
    globals: true,
  },
});
```

- [ ] **Step 4: Add `test` script to `apps/admin/package.json`**

In the `"scripts"` object, add:

```json
"test": "vitest run",
"test:watch": "vitest"
```

- [ ] **Step 5: Write the failing store test**

Create `apps/admin/src/stores/auth.spec.ts`:

```ts
import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { setActivePinia, createPinia } from 'pinia';
import { useAuthStore } from './auth';

// Mock the api module so the store's behavior is unit-testable.
vi.mock('@/lib/api', async () => {
  const actual: Record<string, any> = {};
  let token: string | null = null;
  return {
    api: {
      post: vi.fn(),
      get: vi.fn(),
    },
    storeToken: vi.fn((t: string) => {
      token = t;
    }),
    clearToken: vi.fn(() => {
      token = null;
    }),
    readToken: vi.fn(() => token),
    ...actual,
  };
});

import { api, storeToken, clearToken, readToken } from '@/lib/api';

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    (storeToken as any).mockClear();
    (clearToken as any).mockClear();
    (api.post as any).mockReset();
    (api.get as any).mockReset();
  });

  afterEach(() => {
    (readToken as any).mockReturnValue(null);
  });

  it('starts unauthenticated', () => {
    const store = useAuthStore();
    expect(store.isAuthenticated).toBe(false);
    expect(store.admin).toBeNull();
  });

  it('login success stores token and admin', async () => {
    (api.post as any).mockResolvedValue({
      data: { token: 'tok-123', admin: { id: 1, email: 'a@b.c', name: 'A' } },
    });

    const store = useAuthStore();
    const ok = await store.login('a@b.c', 'pw-long-enough');

    expect(ok).toBe(true);
    expect(store.isAuthenticated).toBe(true);
    expect(store.admin).toEqual({ id: 1, email: 'a@b.c', name: 'A' });
    expect(storeToken).toHaveBeenCalledWith('tok-123');
  });

  it('login failure returns false and leaves state unchanged', async () => {
    (api.post as any).mockRejectedValue({ response: { status: 401 } });
    const store = useAuthStore();

    const ok = await store.login('a@b.c', 'wrong');
    expect(ok).toBe(false);
    expect(store.isAuthenticated).toBe(false);
    expect(storeToken).not.toHaveBeenCalled();
  });

  it('logout clears token and admin', async () => {
    (api.post as any).mockResolvedValue({
      data: { token: 'tok', admin: { id: 1, email: 'a@b.c', name: 'A' } },
    });
    const store = useAuthStore();
    await store.login('a@b.c', 'pw-long-enough');

    store.logout();
    expect(store.isAuthenticated).toBe(false);
    expect(store.admin).toBeNull();
    expect(clearToken).toHaveBeenCalled();
  });

  it('restore() fetches /me when a token exists', async () => {
    (readToken as any).mockReturnValue('existing-tok');
    (api.get as any).mockResolvedValue({
      data: { id: 1, email: 'a@b.c', name: 'A' },
    });

    const store = useAuthStore();
    await store.restore();

    expect(api.get).toHaveBeenCalledWith('/api/auth/me');
    expect(store.isAuthenticated).toBe(true);
    expect(store.admin?.email).toBe('a@b.c');
  });

  it('restore() clears token if /me fails', async () => {
    (readToken as any).mockReturnValue('bad-tok');
    (api.get as any).mockRejectedValue({ response: { status: 401 } });

    const store = useAuthStore();
    await store.restore();

    expect(store.isAuthenticated).toBe(false);
    expect(clearToken).toHaveBeenCalled();
  });
});
```

- [ ] **Step 6: Run — expect fail (useAuthStore not found)**

```bash
cd apps/admin
bun run test
# Expected: cannot resolve ./auth
```

- [ ] **Step 7: Implement the store**

Create `apps/admin/src/stores/auth.ts`:

```ts
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { api, storeToken, clearToken, readToken } from '@/lib/api';

export interface Admin {
  id: number;
  email: string;
  name: string;
}

export const useAuthStore = defineStore('auth', () => {
  const admin = ref<Admin | null>(null);
  const loading = ref(false);

  const isAuthenticated = computed(() => admin.value !== null);

  async function login(email: string, password: string): Promise<boolean> {
    loading.value = true;
    try {
      const res = await api.post<{ token: string; admin: Admin }>('/api/auth/login', {
        email,
        password,
      });
      storeToken(res.data.token);
      admin.value = res.data.admin;
      return true;
    } catch {
      return false;
    } finally {
      loading.value = false;
    }
  }

  function logout() {
    clearToken();
    admin.value = null;
  }

  async function restore() {
    if (!readToken()) return;
    try {
      const res = await api.get<Admin>('/api/auth/me');
      admin.value = res.data;
    } catch {
      clearToken();
      admin.value = null;
    }
  }

  // React to global 401s (invalid/expired token) by logging out.
  if (typeof window !== 'undefined') {
    window.addEventListener('gekko:unauthorized', () => logout());
  }

  return { admin, loading, isAuthenticated, login, logout, restore };
});
```

- [ ] **Step 8: Run — expect pass**

```bash
bun run test
# Expected: 6 tests passing
```

- [ ] **Step 9: Commit**

```bash
git add apps/admin/src/lib apps/admin/src/stores apps/admin/vitest.config.ts apps/admin/package.json apps/admin/bun.lock
git commit -m "feat(admin): axios client + pinia auth store with tests"
```

---

## Task 9: Admin — router skeleton

**Files:**
- Create: `apps/admin/src/router/index.ts`
- Create: `apps/admin/src/views/LoginView.vue`
- Create: `apps/admin/src/views/DashboardView.vue`
- Create: `apps/admin/src/views/NotFoundView.vue`
- Create: `apps/admin/src/layouts/AppShell.vue`
- Replace: `apps/admin/src/main.ts`
- Replace: `apps/admin/src/App.vue`

- [ ] **Step 1: Placeholder `LoginView.vue`**

Create `apps/admin/src/views/LoginView.vue`:

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { Button } from '@/components/ui/button';
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from 'vue-sonner';
import { useAuthStore } from '@/stores/auth';

const email = ref('');
const password = ref('');
const auth = useAuthStore();
const router = useRouter();

async function onSubmit(e: Event) {
  e.preventDefault();
  const ok = await auth.login(email.value, password.value);
  if (ok) {
    router.push('/');
  } else {
    toast.error('Invalid email or password.');
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-brand-cream-50 px-4">
    <Card class="w-full max-w-md border-brand-cream-300">
      <CardHeader>
        <CardTitle class="text-3xl">Zenetic Gekkos</CardTitle>
        <CardDescription class="text-brand-dark-600">
          Admin sign-in
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form class="flex flex-col gap-4" @submit="onSubmit">
          <div class="flex flex-col gap-2">
            <Label for="email">Email</Label>
            <Input id="email" v-model="email" type="email" autocomplete="username" required />
          </div>
          <div class="flex flex-col gap-2">
            <Label for="password">Password</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              autocomplete="current-password"
              required
            />
          </div>
          <Button
            type="submit"
            :disabled="auth.loading"
            class="bg-brand-gold-600 hover:bg-brand-gold-700 text-white"
          >
            {{ auth.loading ? 'Signing in…' : 'Sign in' }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
```

- [ ] **Step 2: Placeholder `DashboardView.vue`**

Create `apps/admin/src/views/DashboardView.vue`:

```vue
<script setup lang="ts">
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card';
import { useAuthStore } from '@/stores/auth';

const auth = useAuthStore();
</script>

<template>
  <div class="flex flex-col gap-6">
    <div>
      <h1 class="text-4xl">Welcome back, {{ auth.admin?.name || auth.admin?.email }}.</h1>
      <p class="text-brand-dark-600 mt-2">
        Dashboard stats land here in phase 5. For now, this proves auth works.
      </p>
    </div>
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <Card v-for="stat in stats" :key="stat.label" class="border-brand-cream-300">
        <CardHeader>
          <CardTitle class="text-brand-dark-600 text-sm font-medium">{{ stat.label }}</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="text-3xl font-serif">{{ stat.value }}</div>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script lang="ts">
const stats = [
  { label: 'Total geckos', value: '—' },
  { label: 'Active pairings', value: '—' },
  { label: 'Eggs incubating', value: '—' },
  { label: 'Waitlist', value: '—' },
];
</script>
```

- [ ] **Step 3: `NotFoundView.vue`**

Create `apps/admin/src/views/NotFoundView.vue`:

```vue
<template>
  <div class="flex flex-col items-center justify-center min-h-screen gap-2">
    <h1 class="text-5xl">404</h1>
    <p class="text-brand-dark-600">That page isn't part of the admin (yet).</p>
    <RouterLink to="/" class="text-brand-gold-700 underline mt-4">Back to dashboard</RouterLink>
  </div>
</template>
```

- [ ] **Step 4: `AppShell.vue` — sidebar + top bar**

Create `apps/admin/src/layouts/AppShell.vue`:

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router';
import { Button } from '@/components/ui/button';
import { LogOut, LayoutDashboard } from 'lucide-vue-next';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const auth = useAuthStore();

function onLogout() {
  auth.logout();
  router.push('/login');
}
</script>

<template>
  <div class="min-h-screen flex bg-brand-cream-50 text-brand-dark-950">
    <aside class="w-56 border-r border-brand-cream-300 p-4 flex flex-col gap-2">
      <div class="font-serif text-xl mb-4">Zenetic</div>
      <RouterLink
        to="/"
        class="flex items-center gap-2 px-3 py-2 rounded-md hover:bg-brand-cream-100"
        active-class="bg-brand-gold-100 text-brand-gold-800"
      >
        <LayoutDashboard class="w-4 h-4" /> Dashboard
      </RouterLink>
      <!-- Nav for Geckos/Waitlist/Data added in later phases. -->
    </aside>
    <div class="flex-1 flex flex-col">
      <header class="border-b border-brand-cream-300 px-6 py-3 flex items-center justify-between">
        <div class="text-sm text-brand-dark-600">
          Signed in as {{ auth.admin?.email }}
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="text-brand-dark-950 hover:bg-brand-cream-100"
          @click="onLogout"
        >
          <LogOut class="w-4 h-4 mr-2" /> Log out
        </Button>
      </header>
      <main class="p-6 flex-1 overflow-auto">
        <RouterView />
      </main>
    </div>
  </div>
</template>
```

- [ ] **Step 5: Router with guards**

Create `apps/admin/src/router/index.ts`:

```ts
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '@/stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('@/layouts/AppShell.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/DashboardView.vue'),
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
  },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  const requiresAuth = to.matched.some((r) => r.meta.requiresAuth);

  if (requiresAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } };
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    return { name: 'dashboard' };
  }
  return true;
});
```

- [ ] **Step 6: Wire everything in `main.ts`**

Replace `apps/admin/src/main.ts`:

```ts
import { createApp } from 'vue';
import { createPinia } from 'pinia';
import { VueQueryPlugin } from '@tanstack/vue-query';
import App from './App.vue';
import { router } from './router';
import { useAuthStore } from '@/stores/auth';
import './style.css';

async function bootstrap() {
  const app = createApp(App);
  const pinia = createPinia();
  app.use(pinia);
  app.use(VueQueryPlugin);

  // Restore session before the first navigation so guards see the real state.
  const auth = useAuthStore();
  await auth.restore();

  app.use(router);
  app.mount('#app');
}

bootstrap();
```

- [ ] **Step 7: Replace `App.vue`**

```vue
<script setup lang="ts">
import { Toaster } from '@/components/ui/sonner';
</script>

<template>
  <RouterView />
  <Toaster rich-colors position="top-right" />
</template>
```

- [ ] **Step 8: Run the dev server and manually verify**

```bash
# Terminal 1: backend
cd backend && go run ./cmd/gekko

# Terminal 2: admin
bun --cwd apps/admin run dev
# Open http://localhost:5173
# Expected: redirect to /login, form renders with brand colors + DM Serif title.
# Log in with seeded creds. Expected: lands on /, shows "Welcome back, Zen." + top bar email.
# Click Log out. Expected: back at /login, localStorage.gekko.admin.token is gone.
# Manually visit http://localhost:5173/ again. Expected: redirect to /login.
```

- [ ] **Step 9: Commit**

```bash
git add apps/admin
git commit -m "feat(admin): router + login + app shell + dashboard placeholder"
```

---

## Task 10: Root Makefile — `test` target + small polish

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Append `test` + `test-backend` + `test-admin` targets**

```makefile
test: test-backend test-admin  ## Run all tests

test-backend:  ## Run Go tests
	cd backend && go test ./...

test-admin:  ## Run admin Vitest tests
	bun --cwd apps/admin run test
```

Update the `.PHONY` line:

```makefile
.PHONY: help db-up db-down db-logs migrate migrate-status migrate-down sqlc dev-backend dev-admin dev-storefront seed test test-backend test-admin
```

- [ ] **Step 2: Verify**

```bash
make help
make test
# Expected: Go tests PASS, admin Vitest tests PASS
```

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add make test / test-backend / test-admin targets"
```

---

## Task 11: End-to-end manual smoke test + checklist

**Files:** none new — this task only exercises what exists.

- [ ] **Step 1: Start from a clean slate**

```bash
# Stop anything running
pkill -f 'go run ./cmd/gekko' 2>/dev/null || true
pkill -f 'vite' 2>/dev/null || true

make db-down
make db-up
make migrate
# Expected: both migrations applied
```

- [ ] **Step 2: Seed the admin**

```bash
ADMIN_EMAIL=zen@zeneticgekkos.com ADMIN_PASSWORD=change-me-later ADMIN_NAME=Zen make seed
# Expected: "created admin id=1 ..."  (or "admin already exists ..." if re-running)
```

- [ ] **Step 3: Run the full test suite**

```bash
make test
# Expected: Go tests green, Vitest green
```

- [ ] **Step 4: Start backend + admin**

```bash
# Terminal 1
make dev-backend

# Terminal 2
make dev-admin
```

- [ ] **Step 5: Walk the checklist in a fresh browser window**

Use a fresh incognito window so `localStorage` is empty.

- Open `http://localhost:5173` → **redirects to `/login`**.
- Confirm fonts: the title "Zenetic Gekkos" renders in DM Serif Display (serif), labels/buttons in Inter.
- Confirm colors: background is warm cream (`#faf8f3`), button is gold (`#b06c12`). No raw grays or blue.
- Try wrong password → toast "Invalid email or password.", no redirect.
- Try correct creds → lands on `/`, top bar shows "Signed in as zen@zeneticgekkos.com", dashboard cards render (`—` placeholders).
- Open DevTools → Application → Local Storage → `gekko.admin.token` exists.
- Click **Log out** → redirects to `/login`, token is gone.
- Visit `/` directly in the URL bar → redirects to `/login`.
- Log in again, reload the page → dashboard still renders (session restored via `/me`).
- Edit `localStorage.gekko.admin.token` to a garbage string, reload → redirects to `/login` (401 handler fires).

- [ ] **Step 6: If anything fails, fix before declaring done**

Do **not** mark the plan complete if any bullet above fails. Fix the regression, re-run the relevant test or checklist step, and commit the fix.

- [ ] **Step 7: Final commit for checklist artifacts (if any)**

```bash
git status
# If any incidental fixes were made during the smoke test, commit them as one:
git commit -am "fix: phase 1 smoke test fixes"
```

---

## Self-review checklist (for the plan author)

- [ ] **Spec coverage**: Phase 1 covers admin v1 spec §5 (login), §6 (architecture wiring), §7 (admin_users table already done), §12 (JWT auth, requiresAuth guard). Phases 2–6 listed at top cover the remaining spec sections.
- [ ] **No placeholders**: every code step contains the actual code. No `// TODO: implement`.
- [ ] **Type consistency**: `Admin` type (id/email/name) is the same across backend `adminDTO`, frontend `Admin` interface, `loginResp`, and `/me` response.
- [ ] **Commands runnable as written**: every `bash` block assumes the repo root unless the step uses `cd`. All commands use the same DB URL, port, env conventions.
- [ ] **Bite-sized**: each step is one concrete action (2–5 min). Write / Run / Commit rhythm throughout.

---

## Execution

Plan complete and saved to `docs/superpowers/plans/2026-04-20-phase-1-auth-foundation.md`. Two execution options:

**1. Subagent-Driven (recommended)** — fresh subagent per task with review between tasks. Fastest honest iteration.

**2. Inline Execution** — execute tasks in this session with batch checkpoints for review.

Which approach?
