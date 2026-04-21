# Storefront MVP — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up `apps/storefront` — a public Vue 3 SPA with home, gecko list, gecko detail, and waitlist-signup pages — plus three new backend public endpoints (list/detail available geckos, waitlist signup) behind a light in-memory rate limiter.

**Architecture:** Second bun-workspace Vue app on port 5174, same stack and brand tokens as admin. Backend adds `MountPublic` with no JWT dependency — uses sanitized DTOs that strictly exclude `notes`, `sire_id`, `dam_id`, `status`, `acquired_date`, `updated_at` from public responses. Morph composition mirrors admin's `morphFromTraits` but runs server-side.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, sqlc@v1.27; Vue 3.5, Vite, TypeScript, Tailwind 4, shadcn-vue, TanStack Vue Query, Vitest.

**Spec:** `docs/superpowers/specs/2026-04-21-storefront-mvp-design.md`

---

## File Structure

**Backend (new):**
- `backend/internal/queries/public.sql` — 4 public queries (`ListAvailableGeckos`, `GetAvailableGeckoByCode`, `ListGenesForGeckoByCode`, `ListMediaForGeckoByCode`).
- `backend/internal/db/public.sql.go` — **generated**, do not hand-edit.
- `backend/internal/http/ratelimit.go` — in-memory per-IP rate-limit middleware with the inline TODO.
- `backend/internal/http/public.go` — `MountPublic`, 3 handlers, sanitized DTOs, Go-side morph composition.
- `backend/internal/http/public_test.go` — 6 integration tests.

**Backend (modified):**
- `backend/cmd/gekko/main.go` — append `apihttp.MountPublic(r, pool)`.

**Storefront (new app, all new files):**
```
apps/storefront/
├── index.html
├── package.json
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── vite.config.ts
├── vitest.config.ts
├── components.json
├── public/logo/                  # copied from apps/admin/public/logo/
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── style.css                 # copied from apps/admin/src/style.css
│   ├── router/index.ts
│   ├── lib/
│   │   ├── api.ts                # NO token interceptor (public app)
│   │   └── format.ts             # copied from apps/admin/src/lib/format.ts
│   ├── types/gecko.ts            # PUBLIC subset only
│   ├── composables/
│   │   ├── usePublicGeckos.ts
│   │   └── useWaitlistSignup.ts
│   ├── components/
│   │   ├── BrandLogo.vue         # copied from apps/admin
│   │   ├── SiteHeader.vue
│   │   ├── SiteFooter.vue
│   │   ├── PublicGeckoCard.vue
│   │   ├── art/
│   │   │   ├── LowPolyGecko.vue  # copied from apps/admin
│   │   │   └── LowPolyAccent.vue # copied from apps/admin
│   │   └── ui/                   # shadcn primitives copied from apps/admin
│   │       ├── button/
│   │       ├── card/
│   │       ├── input/
│   │       ├── label/
│   │       ├── badge/
│   │       ├── skeleton/
│   │       └── sonner/
│   └── views/
│       ├── HomeView.vue
│       ├── GeckosView.vue
│       ├── GeckoDetailView.vue
│       └── WaitlistView.vue
└── tests/App.smoke.spec.ts
```

---

## Task 1: Backend — public sqlc queries

**Files:**
- Create: `backend/internal/queries/public.sql`

- [ ] **Step 1: Create the queries file**

Write `backend/internal/queries/public.sql`:

```sql
-- name: ListAvailableGeckos :many
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.list_price_usd,
  g.created_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
WHERE g.status = 'AVAILABLE'
ORDER BY g.created_at DESC;

-- name: GetAvailableGeckoByCode :one
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.list_price_usd,
  g.created_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
WHERE g.status = 'AVAILABLE' AND g.code = $1
LIMIT 1;

-- name: ListPublicGenesByGeckoIDs :many
-- Used to compose morphs for the list endpoint in one round trip.
SELECT
  gg.gecko_id,
  gd.trait_name,
  gd.trait_code,
  gg.zygosity
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
WHERE gg.gecko_id = ANY($1::int[])
ORDER BY gg.gecko_id, gd.trait_name;

-- name: ListPublicMediaByGeckoIDs :many
-- Used to pre-load cover photos (first by display_order) for the list view.
SELECT DISTINCT ON (gecko_id) gecko_id, url, caption, display_order, uploaded_at
FROM media
WHERE gecko_id = ANY($1::int[])
ORDER BY gecko_id, display_order, uploaded_at;
```

- [ ] **Step 2: Generate Go code**

Run: `cd /home/zen/dev/project_gekko/backend && /home/zen/go/bin/sqlc generate`
Expected: silent exit 0; `backend/internal/db/public.sql.go` created.

- [ ] **Step 3: Verify build**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/queries/public.sql backend/internal/db/public.sql.go backend/internal/db/querier.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): sqlc queries for public storefront endpoints

ListAvailableGeckos (list with species join), GetAvailableGeckoByCode
(single, 404 on non-AVAILABLE), plus two :many helpers keyed on
gecko_id arrays so the list endpoint preloads traits and cover photos
in one round trip each — no N+1.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Backend — in-memory rate-limit middleware

**Files:**
- Create: `backend/internal/http/ratelimit.go`

- [ ] **Step 1: Create the middleware file**

Write `backend/internal/http/ratelimit.go`:

```go
package http

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TODO: swap to Postgres- or Redis-backed limiter when we run more than
// one backend instance, or when a second rate-limited endpoint appears.

// IPRateLimiter is a simple per-IP sliding-window limiter. Safe for concurrent
// use. Not cluster-safe. Intended for low-traffic single-box MVP deployments.
type IPRateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	window time.Duration
	max    int
	now    func() time.Time
}

// NewIPRateLimiter returns a limiter that allows `max` requests per `window`
// per client IP. For MVP use 5 per hour on POST /api/public/waitlist.
func NewIPRateLimiter(max int, window time.Duration) *IPRateLimiter {
	return &IPRateLimiter{
		hits:   map[string][]time.Time{},
		window: window,
		max:    max,
		now:    time.Now,
	}
}

// Middleware wraps a handler so requests beyond the limit receive 429.
func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		if !l.allow(ip) {
			w.Header().Set("Retry-After", fmt.Sprintf("%.0f", l.window.Seconds()))
			writeJSON(w, http.StatusTooManyRequests, map[string]string{
				"error": "too many requests; try again later",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// allow records a hit for the given ip and returns false if that would exceed
// the limit.
func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.window)

	fresh := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cutoff) {
			fresh = append(fresh, t)
		}
	}
	if len(fresh) >= l.max {
		l.hits[ip] = fresh
		return false
	}
	fresh = append(fresh, now)
	l.hits[ip] = fresh
	return true
}

// clientIP extracts the client IP from X-Forwarded-For (first entry) if the
// header is present, falling back to RemoteAddr. Vite proxy in dev mode sets
// X-Forwarded-For correctly; production reverse proxies should also.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	addr := r.RemoteAddr
	if i := strings.LastIndexByte(addr, ':'); i >= 0 {
		return addr[:i]
	}
	return addr
}
```

- [ ] **Step 2: Verify build**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 3: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/ratelimit.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): in-memory per-IP rate-limit middleware

Simple sliding-window limiter for the upcoming POST /api/public/waitlist
endpoint. 5 requests per hour per client IP is enough to deter form
spam without Redis. Explicit TODO at the top of the file points to the
scaling path (Postgres- or Redis-backed limiter) when we outgrow it.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Backend — public handlers + waitlist create

**Files:**
- Create: `backend/internal/http/public.go`
- Modify: `backend/cmd/gekko/main.go`

- [ ] **Step 1: Create `backend/internal/http/public.go`**

```go
package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountPublic mounts the storefront-facing endpoints (no auth). One endpoint
// is wrapped in a per-IP rate limiter (waitlist signup). The list + detail
// endpoints are left open — they're trivially cacheable and low-risk.
func MountPublic(r chi.Router, pool *pgxpool.Pool) {
	d := &publicDeps{pool: pool, q: db.New(pool)}
	waitlistLimiter := NewIPRateLimiter(5, time.Hour)

	r.Get("/api/public/geckos", d.listAvailable)
	r.Get("/api/public/geckos/{code}", d.getByCode)
	r.With(waitlistLimiter.Middleware).Post("/api/public/waitlist", d.createWaitlist)
}

type publicDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// ---- public DTOs (strict subset; no notes/sire/dam/status/acquired_date) ----

type publicGeckoDTO struct {
	Code           string  `json:"code"`
	Name           string  `json:"name"`
	SpeciesCode    string  `json:"species_code"`
	SpeciesName    string  `json:"species_name"`
	Morph          string  `json:"morph"`
	Sex            string  `json:"sex"`
	HatchDate      *string `json:"hatch_date"`
	ListPriceUsd   *string `json:"list_price_usd"`
	CoverPhotoUrl  *string `json:"cover_photo_url"`
}

type publicGeckoListResp struct {
	Geckos []publicGeckoDTO `json:"geckos"`
	Total  int              `json:"total"`
}

type publicMediaDTO struct {
	Url          string `json:"url"`
	Caption      string `json:"caption"`
	DisplayOrder int32  `json:"display_order"`
}

type publicGeckoDetailDTO struct {
	publicGeckoDTO
	Photos []publicMediaDTO `json:"photos"`
}

// ---- list ----

func (d *publicDeps) listAvailable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := d.q.ListAvailableGeckos(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}

	ids := make([]int32, 0, len(rows))
	for _, g := range rows {
		ids = append(ids, g.ID)
	}

	// Preload traits + covers in one round trip each.
	genes, err := d.q.ListPublicGenesByGeckoIDs(ctx, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list genes failed"})
		return
	}
	genesByGecko := map[int32][]db.ListPublicGenesByGeckoIDsRow{}
	for _, g := range genes {
		genesByGecko[g.GeckoID] = append(genesByGecko[g.GeckoID], g)
	}

	covers, err := d.q.ListPublicMediaByGeckoIDs(ctx, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list covers failed"})
		return
	}
	coverByGecko := map[int32]string{}
	for _, c := range covers {
		coverByGecko[c.GeckoID.Int32] = c.Url
	}

	out := make([]publicGeckoDTO, 0, len(rows))
	for _, g := range rows {
		out = append(out, publicGeckoDTO{
			Code:          g.Code,
			Name:          textOrEmpty(g.Name),
			SpeciesCode:   string(g.SpeciesCode),
			SpeciesName:   g.SpeciesCommonName,
			Morph:         composePublicMorph(genesByGecko[g.ID]),
			Sex:           string(g.Sex),
			HatchDate:     dateOrNil(g.HatchDate),
			ListPriceUsd:  numericOrNil(g.ListPriceUsd),
			CoverPhotoUrl: coverPtr(coverByGecko, g.ID),
		})
	}

	writeJSON(w, http.StatusOK, publicGeckoListResp{Geckos: out, Total: len(out)})
}

// ---- detail ----

func (d *publicDeps) getByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}

	ctx := r.Context()
	row, err := d.q.GetAvailableGeckoByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}

	genes, err := d.q.ListPublicGenesByGeckoIDs(ctx, []int32{row.ID})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list genes failed"})
		return
	}

	photos, err := d.q.ListMediaForGecko(ctx, pgtype.Int4{Int32: row.ID, Valid: true})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list media failed"})
		return
	}
	photosOut := make([]publicMediaDTO, 0, len(photos))
	for _, p := range photos {
		photosOut = append(photosOut, publicMediaDTO{
			Url:          p.Url,
			Caption:      textOrEmpty(p.Caption),
			DisplayOrder: p.DisplayOrder,
		})
	}

	var cover *string
	if len(photosOut) > 0 {
		cover = &photosOut[0].Url
	}

	writeJSON(w, http.StatusOK, publicGeckoDetailDTO{
		publicGeckoDTO: publicGeckoDTO{
			Code:          row.Code,
			Name:          textOrEmpty(row.Name),
			SpeciesCode:   string(row.SpeciesCode),
			SpeciesName:   row.SpeciesCommonName,
			Morph:         composePublicMorph(genes),
			Sex:           string(row.Sex),
			HatchDate:     dateOrNil(row.HatchDate),
			ListPriceUsd:  numericOrNil(row.ListPriceUsd),
			CoverPhotoUrl: cover,
		},
		Photos: photosOut,
	})
}

// ---- waitlist ----

type publicWaitlistReq struct {
	Email        string `json:"email"`
	Telegram     string `json:"telegram"`
	Phone        string `json:"phone"`
	InterestedIn string `json:"interested_in"`
	Notes        string `json:"notes"`
}

type publicWaitlistResp struct {
	ID           *int32 `json:"id,omitempty"`
	Deduplicated bool   `json:"deduplicated,omitempty"`
}

var emailRE = regexp.MustCompile(`^.+@.+\..+$`)

func (d *publicDeps) createWaitlist(w http.ResponseWriter, r *http.Request) {
	var req publicWaitlistReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Telegram = strings.TrimSpace(req.Telegram)
	req.Phone = strings.TrimSpace(req.Phone)
	req.InterestedIn = strings.TrimSpace(req.InterestedIn)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required"})
		return
	}
	if !emailRE.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return
	}
	if len(req.Email) > 255 || len(req.Telegram) > 100 || len(req.Phone) > 32 ||
		len(req.InterestedIn) > 100 || len(req.Notes) > 2000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field too long"})
		return
	}

	row, err := d.q.CreateWaitlistEntry(r.Context(), db.CreateWaitlistEntryParams{
		Email:        req.Email,
		Telegram:     pgText(req.Telegram),
		Phone:        pgText(req.Phone),
		InterestedIn: pgText(req.InterestedIn),
		Column5:      "storefront",
		Notes:        pgText(req.Notes),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			writeJSON(w, http.StatusOK, publicWaitlistResp{Deduplicated: true})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}

	id := row.ID
	writeJSON(w, http.StatusCreated, publicWaitlistResp{ID: &id})
}

// ---- helpers ----

type geneRow struct {
	TraitName string
	Zygosity  db.Zygosity
}

// composePublicMorph mirrors admin's morphFromTraits in Go so the public
// storefront stays free of genetics business logic.
func composePublicMorph(rows []db.ListPublicGenesByGeckoIDsRow) string {
	if len(rows) == 0 {
		return "Normal"
	}
	var hom, het, poss []string
	for _, r := range rows {
		switch r.Zygosity {
		case db.ZygosityHOM:
			hom = append(hom, r.TraitName)
		case db.ZygosityHET:
			het = append(het, r.TraitName)
		case db.ZygosityPOSSHET:
			poss = append(poss, r.TraitName)
		}
	}
	parts := []string{}
	if len(hom) > 0 {
		parts = append(parts, strings.Join(hom, " "))
	}
	if len(het) > 0 {
		prefixed := make([]string, len(het))
		for i, n := range het {
			prefixed[i] = "het " + n
		}
		parts = append(parts, strings.Join(prefixed, " "))
	}
	if len(poss) > 0 {
		prefixed := make([]string, len(poss))
		for i, n := range poss {
			prefixed[i] = "poss. het " + n
		}
		parts = append(parts, strings.Join(prefixed, " "))
	}
	if len(parts) == 0 {
		return "Normal"
	}
	return strings.Join(parts, ", ")
}

func coverPtr(m map[int32]string, id int32) *string {
	if u, ok := m[id]; ok {
		return &u
	}
	return nil
}
```

- [ ] **Step 2: Add pgerrcode dependency**

Run: `cd /home/zen/dev/project_gekko/backend && go get github.com/jackc/pgerrcode`
Expected: `go: added github.com/jackc/pgerrcode ...` or already-present.

- [ ] **Step 3: Mount in main.go**

Open `backend/cmd/gekko/main.go`. Find the existing `apihttp.Mount*` block. Append:

```go
apihttp.MountPublic(r, pool)
```

Final block should read:

```go
apihttp.MountAuth(r, pool, signer)
apihttp.MountWaitlist(r, pool, signer)
apihttp.MountSchema(r, pool, signer)
apihttp.MountGeckos(r, pool, signer)
apihttp.MountMedia(r, pool, signer, cfg)
apihttp.MountDashboard(r, pool, signer)
apihttp.MountPublic(r, pool)
```

- [ ] **Step 4: Verify build**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 5: Smoke-test with curl (backend on :8420)**

```bash
echo "--- public list ---"
curl -sS http://localhost:8420/api/public/geckos | python3 -m json.tool | head -20

echo "--- public waitlist ---"
curl -sS -X POST http://localhost:8420/api/public/waitlist \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke+test@example.com","interested_in":"smoke"}' \
  -w "\nstatus=%{http_code}\n"

echo "--- duplicate waitlist ---"
curl -sS -X POST http://localhost:8420/api/public/waitlist \
  -H "Content-Type: application/json" \
  -d '{"email":"smoke+test@example.com"}' -w "\nstatus=%{http_code}\n"
```

Expected: list returns `{"geckos":[...],"total":N}`; first waitlist returns `201` with an id; duplicate returns `200` with `{"deduplicated":true}`. Clean up: `docker exec -i gekko_db psql -U gekko -d gekko -c "DELETE FROM waitlist_entries WHERE email='smoke+test@example.com'"`.

- [ ] **Step 6: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/public.go backend/cmd/gekko/main.go backend/go.mod backend/go.sum
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): public storefront endpoints (list/detail/waitlist)

MountPublic with three routes, no JWT. Sanitized DTOs — never leak
notes/sire/dam/status/acquired_date/updated_at. List handler preloads
genes + cover photos in one round trip each. Detail returns a 404 for
non-AVAILABLE codes so we don't advertise private stock. Waitlist POST
is wrapped in the per-IP rate limiter (5/hour) and returns
{ deduplicated: true } on unique-violation so we don't reveal who's
already on the list.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Backend — public endpoint tests

**Files:**
- Create: `backend/internal/http/public_test.go`

- [ ] **Step 1: Write the test file**

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

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func publicSetup(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	r := chi.NewRouter()
	MountPublic(r, pool)
	return r, pool
}

func makePublicGecko(t *testing.T, pool *pgxpool.Pool, code string, status db.GeckoStatus) int32 {
	t.Helper()
	q := db.New(pool)
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT id FROM species ORDER BY id LIMIT 1").Scan(&speciesID))
	g, err := q.CreateGecko(context.Background(), db.CreateGeckoParams{
		Code:      code,
		SpeciesID: speciesID,
		Sex:       db.SexM,
		Column7:   db.NullGeckoStatus{GeckoStatus: status, Valid: true},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM geckos WHERE id = $1", g.ID)
	})
	return g.ID
}

func TestPublicListGeckos_onlyAvailable(t *testing.T) {
	router, pool := publicSetup(t)
	stamp := time.Now().Format("150405000")
	makePublicGecko(t, pool, "PA-"+stamp, db.GeckoStatusAVAILABLE)
	makePublicGecko(t, pool, "PH-"+stamp, db.GeckoStatusHOLD)
	makePublicGecko(t, pool, "PB-"+stamp, db.GeckoStatusBREEDING)

	req := httptest.NewRequest(http.MethodGet, "/api/public/geckos", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var body struct {
		Geckos []publicGeckoDTO `json:"geckos"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

	var codes []string
	for _, g := range body.Geckos {
		codes = append(codes, g.Code)
	}
	assert.Contains(t, codes, "PA-"+stamp)
	assert.NotContains(t, codes, "PH-"+stamp)
	assert.NotContains(t, codes, "PB-"+stamp)
}

func TestPublicGetGecko_byCode_available(t *testing.T) {
	router, pool := publicSetup(t)
	stamp := time.Now().Format("150405000")
	code := "PG-" + stamp
	makePublicGecko(t, pool, code, db.GeckoStatusAVAILABLE)

	req := httptest.NewRequest(http.MethodGet, "/api/public/geckos/"+code, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	raw := rr.Body.String()
	assert.Contains(t, raw, `"code":"`+code+`"`)
	assert.NotContains(t, raw, `"notes"`)
	assert.NotContains(t, raw, `"sire_id"`)
	assert.NotContains(t, raw, `"dam_id"`)
	assert.NotContains(t, raw, `"status"`)
}

func TestPublicGetGecko_byCode_notAvailable(t *testing.T) {
	router, pool := publicSetup(t)
	stamp := time.Now().Format("150405000")
	code := "PN-" + stamp
	makePublicGecko(t, pool, code, db.GeckoStatusHOLD)

	req := httptest.NewRequest(http.MethodGet, "/api/public/geckos/"+code, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPublicWaitlist_create(t *testing.T) {
	router, pool := publicSetup(t)
	email := "t+" + time.Now().Format("150405.000000000") + "@example.com"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM waitlist_entries WHERE email = $1", email)
	})

	body := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())
	var got publicWaitlistResp
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.NotNil(t, got.ID)
}

func TestPublicWaitlist_duplicate(t *testing.T) {
	router, pool := publicSetup(t)
	email := "dup+" + time.Now().Format("150405.000000000") + "@example.com"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM waitlist_entries WHERE email = $1", email)
	})

	body1 := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
	req1 := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body1)
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code)

	body2 := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
	req2 := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body2)
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code, "body=%s", rr2.Body.String())

	var got publicWaitlistResp
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &got))
	assert.True(t, got.Deduplicated)
	assert.Nil(t, got.ID)
}

func TestPublicWaitlist_rateLimit(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// Build a router with a fresh limiter so we're not affected by other tests.
	r := chi.NewRouter()
	d := &publicDeps{pool: pool, q: db.New(pool)}
	limiter := NewIPRateLimiter(5, time.Hour)
	r.With(limiter.Middleware).Post("/api/public/waitlist", d.createWaitlist)

	emails := []string{}
	t.Cleanup(func() {
		for _, e := range emails {
			_, _ = pool.Exec(context.Background(), "DELETE FROM waitlist_entries WHERE email = $1", e)
		}
	})

	okCount, throttled := 0, 0
	for i := 0; i < 6; i++ {
		email := "rl+" + time.Now().Format("150405.000000000") + "@example.com"
		emails = append(emails, email)
		body := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
		req := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.9.8.7:12345" // force the same IP for every request
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		switch rr.Code {
		case http.StatusCreated:
			okCount++
		case http.StatusTooManyRequests:
			throttled++
		}
	}
	assert.Equal(t, 5, okCount, "first 5 should be accepted")
	assert.Equal(t, 1, throttled, "6th should be throttled")

	_ = pgtype.Int4{} // keep pgtype import alive for other tests referencing it in the package
}
```

- [ ] **Step 2: Run the tests**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./internal/http/... -run "Public" -v`
Expected: 6 tests PASS.

- [ ] **Step 3: Run the full backend suite**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./...`
Expected: all packages `ok`.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/public_test.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "test(backend): public endpoint integration tests

Six tests: list returns only AVAILABLE, detail by code returns sanitized
shape, detail 404s on non-AVAILABLE, waitlist create 201, duplicate
returns { deduplicated: true }, rate-limiter throttles the 6th request
from the same IP in an hour.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Storefront — app scaffold

**Files:**
- Create: all files listed below.

- [ ] **Step 1: Create the directory structure**

```bash
cd /home/zen/dev/project_gekko
mkdir -p apps/storefront/src/{router,lib,types,composables,components/art,components/ui,views}
mkdir -p apps/storefront/public/logo
mkdir -p apps/storefront/tests
```

- [ ] **Step 2: Write `apps/storefront/package.json`**

```json
{
  "name": "storefront",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc -b && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest"
  },
  "dependencies": {
    "@tanstack/vue-query": "^5.99.2",
    "@vueuse/core": "^14.2.1",
    "axios": "^1.15.1",
    "class-variance-authority": "^0.7.1",
    "clsx": "^2.1.1",
    "lucide-vue-next": "^1.0.0",
    "pinia": "^3.0.4",
    "reka-ui": "^2.9.6",
    "tailwind-merge": "^3.5.0",
    "vue": "^3.5.32",
    "vue-router": "4",
    "vue-sonner": "^2.0.9"
  },
  "devDependencies": {
    "@tailwindcss/vite": "^4.2.2",
    "@types/node": "^24.12.2",
    "@vitejs/plugin-vue": "^6.0.6",
    "@vitest/ui": "^4.1.4",
    "@vue/tsconfig": "^0.9.1",
    "jsdom": "^29.0.2",
    "tailwindcss": "^4.2.2",
    "tw-animate-css": "^1.4.0",
    "typescript": "~6.0.2",
    "vite": "^8.0.9",
    "vitest": "^4.1.4",
    "vue-tsc": "^3.2.7"
  }
}
```

- [ ] **Step 3: Write `apps/storefront/tsconfig.json`**

```json
{
  "files": [],
  "references": [
    { "path": "./tsconfig.app.json" },
    { "path": "./tsconfig.node.json" }
  ]
}
```

- [ ] **Step 4: Write `apps/storefront/tsconfig.app.json`**

```json
{
  "extends": "@vue/tsconfig/tsconfig.dom.json",
  "include": ["src/**/*", "src/**/*.vue"],
  "exclude": ["src/**/__tests__/*", "tests/**/*"],
  "compilerOptions": {
    "composite": true,
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.app.tsbuildinfo",
    "baseUrl": ".",
    "paths": { "@/*": ["./src/*"] }
  }
}
```

- [ ] **Step 5: Write `apps/storefront/tsconfig.node.json`**

```json
{
  "extends": "@tsconfig/node22/tsconfig.json",
  "include": ["vite.config.*", "vitest.config.*"],
  "compilerOptions": {
    "composite": true,
    "tsBuildInfoFile": "./node_modules/.tmp/tsconfig.node.tsbuildinfo",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "types": ["node"]
  }
}
```

- [ ] **Step 6: Write `apps/storefront/vite.config.ts`**

```ts
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';
import path from 'path';

const BACKEND = process.env.VITE_BACKEND_URL ?? 'http://localhost:8420';

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    port: 5174,
    host: '0.0.0.0',
    allowedHosts: ['.ts.net', 'gekko-dev', 'localhost'],
    proxy: {
      '/api':     { target: BACKEND, changeOrigin: true },
      '/uploads': { target: BACKEND, changeOrigin: true },
      '/health':  { target: BACKEND, changeOrigin: true },
    },
  },
});
```

- [ ] **Step 7: Write `apps/storefront/vitest.config.ts`**

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

- [ ] **Step 8: Write `apps/storefront/index.html`**

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/logo/logo_no_text.svg" />
    <link rel="apple-touch-icon" href="/logo/logo_no_text.png" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Zenetic Gekkos</title>
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

- [ ] **Step 9: Copy brand assets from admin**

```bash
cd /home/zen/dev/project_gekko
cp apps/admin/public/logo/* apps/storefront/public/logo/
cp apps/admin/src/style.css apps/storefront/src/style.css
cp apps/admin/src/components/BrandLogo.vue apps/storefront/src/components/BrandLogo.vue
cp apps/admin/src/components/art/LowPolyGecko.vue apps/storefront/src/components/art/LowPolyGecko.vue
cp apps/admin/src/components/art/LowPolyAccent.vue apps/storefront/src/components/art/LowPolyAccent.vue
cp apps/admin/src/lib/format.ts apps/storefront/src/lib/format.ts
```

- [ ] **Step 10: Copy shadcn UI primitives used by the storefront**

```bash
cd /home/zen/dev/project_gekko
for dir in button card input label badge skeleton sonner; do
  mkdir -p apps/storefront/src/components/ui/$dir
  cp -r apps/admin/src/components/ui/$dir/* apps/storefront/src/components/ui/$dir/
done
cp apps/admin/src/lib/utils.ts apps/storefront/src/lib/utils.ts
cp apps/admin/components.json apps/storefront/components.json
```

- [ ] **Step 11: Write `apps/storefront/src/lib/api.ts`**

Note this is DIFFERENT from admin — no token interceptor. Pure axios.

```ts
import axios from 'axios';

export const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL ?? '',
  withCredentials: false,
});
```

- [ ] **Step 12: Write `apps/storefront/src/types/gecko.ts`**

```ts
export interface PublicGecko {
  code: string;
  name: string;
  species_code: string;
  species_name: string;
  morph: string;
  sex: 'M' | 'F' | 'U';
  hatch_date: string | null;
  list_price_usd: string | null;
  cover_photo_url: string | null;
}

export interface PublicGeckoDetail extends PublicGecko {
  photos: { url: string; caption: string; display_order: number }[];
}

export interface PublicGeckoListResponse {
  geckos: PublicGecko[];
  total: number;
}
```

- [ ] **Step 13: Write `apps/storefront/src/main.ts`**

```ts
import { createApp } from 'vue';
import { createPinia } from 'pinia';
import { VueQueryPlugin } from '@tanstack/vue-query';
import App from './App.vue';
import { router } from './router';
import './style.css';

const app = createApp(App);
app.use(createPinia());
app.use(VueQueryPlugin);
app.use(router);
app.mount('#app');
```

- [ ] **Step 14: Write `apps/storefront/src/App.vue`**

```vue
<script setup lang="ts">
import { Toaster } from '@/components/ui/sonner';
</script>

<template>
  <RouterView />
  <Toaster rich-colors position="top-right" />
</template>
```

- [ ] **Step 15: Write `apps/storefront/src/router/index.ts`** (placeholder — views come in later tasks)

```ts
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';

const routes: RouteRecordRaw[] = [
  { path: '/',              name: 'home',         component: () => import('@/views/HomeView.vue') },
  { path: '/geckos',        name: 'geckos',       component: () => import('@/views/GeckosView.vue') },
  { path: '/geckos/:code',  name: 'gecko-detail', component: () => import('@/views/GeckoDetailView.vue'), props: true },
  { path: '/waitlist',      name: 'waitlist',     component: () => import('@/views/WaitlistView.vue') },
  { path: '/:pathMatch(.*)*', redirect: { name: 'home' } },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
});
```

- [ ] **Step 16: Write stub views so the build compiles** (they get replaced in later tasks — keep a minimal valid component each)

Create `apps/storefront/src/views/HomeView.vue`:
```vue
<template><div>Home — coming in Task 8</div></template>
```
Create `apps/storefront/src/views/GeckosView.vue`:
```vue
<template><div>Geckos — coming in Task 8</div></template>
```
Create `apps/storefront/src/views/GeckoDetailView.vue`:
```vue
<script setup lang="ts">defineProps<{ code: string }>();</script>
<template><div>Gecko detail — coming in Task 9</div></template>
```
Create `apps/storefront/src/views/WaitlistView.vue`:
```vue
<template><div>Waitlist — coming in Task 9</div></template>
```

- [ ] **Step 17: Install dependencies**

```bash
cd /home/zen/dev/project_gekko/apps/storefront
bun install
```
Expected: lockfile updated; all packages install successfully.

- [ ] **Step 18: Verify build + that vite dev server starts**

```bash
cd /home/zen/dev/project_gekko/apps/storefront
bun run build
```
Expected: `vue-tsc -b && vite build` both succeed, `dist/` produced.

Quick runtime check: `bun run dev &` then `curl -sI http://localhost:5174/ | head -3` — expect `HTTP/1.1 200 OK`. Kill the dev server: `kill %1`.

- [ ] **Step 19: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/storefront
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(storefront): scaffold Vue 3 app on port 5174

Second bun-workspace app, same Vite/TS/Tailwind/shadcn-vue stack as
admin. Brand assets (style.css, BrandLogo, LowPolyGecko, LowPolyAccent,
logo/) copied verbatim from admin; shared UI primitives (button/card/
input/label/badge/skeleton/sonner) copied too. api.ts is slim (no
token interceptor). Router wires four public routes; views are stubs
that get replaced in subsequent tasks so the build stays green.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Storefront — composables

**Files:**
- Create: `apps/storefront/src/composables/usePublicGeckos.ts`
- Create: `apps/storefront/src/composables/useWaitlistSignup.ts`

- [ ] **Step 1: Write `usePublicGeckos.ts`**

```ts
import { useQuery } from '@tanstack/vue-query';
import { api } from '@/lib/api';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';
import type { PublicGecko, PublicGeckoDetail, PublicGeckoListResponse } from '@/types/gecko';

export function usePublicGeckos() {
  return useQuery({
    queryKey: ['public', 'geckos'],
    queryFn: async () => {
      const { data } = await api.get<PublicGeckoListResponse>('/api/public/geckos');
      return data;
    },
    staleTime: 60_000,
  });
}

export function usePublicGecko(code: MaybeRef<string | null>) {
  return useQuery({
    queryKey: ['public', 'geckos', code],
    queryFn: async () => {
      const c = unref(code);
      if (!c) throw new Error('no code');
      const { data } = await api.get<PublicGeckoDetail>(`/api/public/geckos/${c}`);
      return data;
    },
    enabled: () => !!unref(code),
    staleTime: 60_000,
    retry: (failureCount, error: any) => {
      // Don't retry on 404 — gecko isn't available.
      if (error?.response?.status === 404) return false;
      return failureCount < 2;
    },
  });
}

// Convenience: latest N available geckos for the home teaser.
export function useAvailableTeaser(n = 3) {
  return useQuery({
    queryKey: ['public', 'geckos', 'teaser', n],
    queryFn: async () => {
      const { data } = await api.get<PublicGeckoListResponse>('/api/public/geckos');
      return data.geckos.slice(0, n);
    },
    staleTime: 60_000,
  });
}
```

- [ ] **Step 2: Write `useWaitlistSignup.ts`**

```ts
import { useMutation } from '@tanstack/vue-query';
import { api } from '@/lib/api';

export interface WaitlistPayload {
  email: string;
  telegram?: string;
  phone?: string;
  interested_in?: string;
  notes?: string;
}

export interface WaitlistResult {
  id?: number;
  deduplicated?: boolean;
}

export function useWaitlistSignup() {
  return useMutation({
    mutationFn: async (payload: WaitlistPayload) => {
      const { data } = await api.post<WaitlistResult>('/api/public/waitlist', payload);
      return data;
    },
  });
}
```

- [ ] **Step 3: Type-check**

Run: `cd /home/zen/dev/project_gekko/apps/storefront && bun run build`
Expected: success.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/storefront/src/composables
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(storefront): public geckos + waitlist composables

TanStack Query wrappers for GET /api/public/geckos (list, single by
code, and a teaser-of-N helper for the home page) plus a mutation for
POST /api/public/waitlist. Detail query skips retries on 404 so a
missing/non-available code fails fast in the UI.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Storefront — SiteHeader + SiteFooter + PublicGeckoCard

**Files:**
- Create: `apps/storefront/src/components/SiteHeader.vue`
- Create: `apps/storefront/src/components/SiteFooter.vue`
- Create: `apps/storefront/src/components/PublicGeckoCard.vue`

- [ ] **Step 1: Write `SiteHeader.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { Button } from '@/components/ui/button';
import { Menu, X } from 'lucide-vue-next';
import BrandLogo from '@/components/BrandLogo.vue';

const mobileOpen = ref(false);
</script>

<template>
  <header class="sticky top-0 z-40 bg-brand-cream-50/95 backdrop-blur-sm border-b border-brand-cream-300">
    <div class="mx-auto max-w-6xl px-4 sm:px-6 h-16 flex items-center justify-between">
      <RouterLink :to="{ name: 'home' }" class="flex items-center">
        <BrandLogo :size="32" />
      </RouterLink>
      <!-- Desktop nav -->
      <nav class="hidden sm:flex items-center gap-6 text-sm">
        <RouterLink
          :to="{ name: 'geckos' }"
          class="text-brand-dark-700 hover:text-brand-dark-950"
          active-class="!text-brand-gold-700"
        >Geckos</RouterLink>
        <RouterLink
          :to="{ name: 'waitlist' }"
          active-class="!text-brand-gold-700"
        >
          <Button variant="default" size="sm">Join waitlist</Button>
        </RouterLink>
      </nav>
      <!-- Mobile toggle -->
      <Button
        class="sm:hidden"
        variant="ghost"
        size="icon"
        aria-label="Toggle menu"
        @click="mobileOpen = !mobileOpen"
      >
        <X v-if="mobileOpen" class="size-5" />
        <Menu v-else class="size-5" />
      </Button>
    </div>
    <!-- Mobile menu -->
    <div
      v-if="mobileOpen"
      class="sm:hidden border-t border-brand-cream-300 bg-brand-cream-50"
    >
      <div class="px-4 py-3 flex flex-col gap-2">
        <RouterLink
          :to="{ name: 'geckos' }"
          class="px-3 py-2 rounded-md hover:bg-brand-cream-100"
          @click="mobileOpen = false"
        >Geckos</RouterLink>
        <RouterLink
          :to="{ name: 'waitlist' }"
          class="px-3 py-2 rounded-md bg-brand-gold-600 text-white text-center hover:bg-brand-gold-700"
          @click="mobileOpen = false"
        >Join waitlist</RouterLink>
      </div>
    </div>
  </header>
</template>
```

- [ ] **Step 2: Write `SiteFooter.vue`**

```vue
<template>
  <footer class="mt-16 border-t border-brand-cream-300 bg-brand-cream-50">
    <div class="mx-auto max-w-6xl px-4 sm:px-6 py-8 text-sm text-brand-dark-600 flex flex-col sm:flex-row gap-4 sm:items-center sm:justify-between">
      <div>© {{ new Date().getFullYear() }} Zenetic Gekkos · Phnom Penh</div>
      <div class="flex items-center gap-4">
        <RouterLink :to="{ name: 'geckos' }" class="hover:text-brand-dark-950">Geckos</RouterLink>
        <RouterLink :to="{ name: 'waitlist' }" class="hover:text-brand-dark-950">Waitlist</RouterLink>
      </div>
    </div>
  </footer>
</template>
```

- [ ] **Step 3: Write `PublicGeckoCard.vue`**

```vue
<script setup lang="ts">
import { Card } from '@/components/ui/card';
import { Mars, Venus, HelpCircle } from 'lucide-vue-next';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import type { PublicGecko } from '@/types/gecko';
import { ageFromBirth } from '@/lib/format';
import { computed } from 'vue';

const props = defineProps<{ gecko: PublicGecko }>();

const sexIcon = computed(() => {
  const m = { M: Mars, F: Venus, U: HelpCircle } as const;
  return m[props.gecko.sex];
});

const sexColor = computed(() => {
  const m = {
    M: 'text-sky-700 bg-sky-100',
    F: 'text-rose-700 bg-rose-100',
    U: 'text-brand-dark-600 bg-brand-cream-200',
  } as const;
  return m[props.gecko.sex];
});
</script>

<template>
  <RouterLink
    :to="{ name: 'gecko-detail', params: { code: gecko.code } }"
    class="block"
  >
    <Card
      class="group overflow-hidden border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 cursor-pointer transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg"
    >
      <div
        class="relative h-48 bg-gradient-to-br from-brand-cream-200 via-brand-gold-100 to-brand-cream-100 flex items-center justify-center overflow-hidden"
      >
        <img
          v-if="gecko.cover_photo_url"
          :src="gecko.cover_photo_url"
          :alt="gecko.name"
          class="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
        />
        <LowPolyGecko v-else :size="160" />
        <div
          class="absolute top-3 right-3 flex size-7 items-center justify-center rounded-full shadow-sm"
          :class="sexColor"
        >
          <component :is="sexIcon" class="size-4" stroke-width="2" />
        </div>
      </div>
      <div class="p-5 flex flex-col gap-2">
        <div class="flex items-start justify-between gap-3">
          <div class="flex flex-col">
            <span class="text-[11px] uppercase tracking-wider text-brand-dark-600 font-semibold">
              {{ gecko.code }}
            </span>
            <h3 class="font-serif text-xl text-brand-dark-950 leading-tight">
              {{ gecko.name || 'Unnamed' }}
            </h3>
          </div>
          <div v-if="gecko.list_price_usd" class="text-right shrink-0">
            <div class="font-semibold text-brand-gold-700">${{ gecko.list_price_usd }}</div>
          </div>
        </div>
        <div class="text-xs text-brand-dark-600">
          <span>{{ gecko.species_name }}</span>
          <span class="mx-2 size-1 inline-block rounded-full bg-brand-cream-400 align-middle" />
          <span class="text-brand-dark-700 font-medium">{{ gecko.morph }}</span>
        </div>
        <div v-if="gecko.hatch_date" class="text-xs text-brand-dark-600">
          {{ ageFromBirth(gecko.hatch_date) }} old
        </div>
      </div>
    </Card>
  </RouterLink>
</template>
```

- [ ] **Step 4: Build**

Run: `cd /home/zen/dev/project_gekko/apps/storefront && bun run build`
Expected: success.

- [ ] **Step 5: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/storefront/src/components/SiteHeader.vue apps/storefront/src/components/SiteFooter.vue apps/storefront/src/components/PublicGeckoCard.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(storefront): site chrome + public gecko card

SiteHeader: sticky top bar with the brand logo, Geckos link, prominent
'Join waitlist' CTA, and a mobile hamburger that expands a vertical nav.
SiteFooter: copyright + two text links. PublicGeckoCard mirrors the
admin card aesthetic but drops private fields — just photo/cover + code,
name, species, morph, sex badge, price, age.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Storefront — HomeView + GeckosView

**Files:**
- Modify: `apps/storefront/src/views/HomeView.vue`
- Modify: `apps/storefront/src/views/GeckosView.vue`

- [ ] **Step 1: Replace `HomeView.vue` with the landing page**

```vue
<script setup lang="ts">
import { useRouter } from 'vue-router';
import { ArrowRight } from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import LowPolyAccent from '@/components/art/LowPolyAccent.vue';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import PublicGeckoCard from '@/components/PublicGeckoCard.vue';
import { useAvailableTeaser } from '@/composables/usePublicGeckos';

const router = useRouter();
const { data: teaser, isLoading: teaserLoading } = useAvailableTeaser(3);
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <!-- Hero -->
    <section class="relative overflow-hidden">
      <div class="mx-auto max-w-6xl px-4 sm:px-6 py-16 sm:py-24 grid grid-cols-1 md:grid-cols-2 gap-10 items-center">
        <div class="flex flex-col gap-6">
          <span class="inline-flex items-center gap-2 text-xs font-semibold tracking-[0.16em] uppercase text-brand-gold-700">
            <LowPolyAccent :size="18" /> Zenetic Gekkos
          </span>
          <h1 class="font-serif text-5xl sm:text-6xl text-brand-dark-950 leading-tight">
            Small-batch gecko breedery in Phnom Penh.
          </h1>
          <p class="text-lg text-brand-dark-700 max-w-md">
            Hand-raised leopard, crested, and African fat-tail geckos — health-first, paired for pattern and temperament.
          </p>
          <div class="flex flex-col sm:flex-row gap-3">
            <Button size="lg" @click="router.push({ name: 'waitlist' })">
              Join the waitlist
              <ArrowRight class="size-4" />
            </Button>
            <Button variant="outline" size="lg" @click="router.push({ name: 'geckos' })">
              Browse available
            </Button>
          </div>
        </div>
        <div class="flex items-center justify-center">
          <LowPolyGecko :size="320" class="animate-float" />
        </div>
      </div>
    </section>

    <!-- Available teaser -->
    <section class="mx-auto max-w-6xl px-4 sm:px-6 py-12">
      <div class="flex items-end justify-between mb-6">
        <div>
          <span class="text-xs uppercase tracking-wider text-brand-gold-700 font-semibold">Currently available</span>
          <h2 class="font-serif text-3xl mt-1">Ready for new homes</h2>
        </div>
        <RouterLink :to="{ name: 'geckos' }" class="text-sm text-brand-gold-700 hover:underline hidden sm:inline">
          See all →
        </RouterLink>
      </div>
      <div v-if="teaserLoading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <Skeleton v-for="n in 3" :key="n" class="h-80 rounded-xl" />
      </div>
      <div v-else-if="teaser && teaser.length" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <PublicGeckoCard v-for="g in teaser" :key="g.code" :gecko="g" />
      </div>
      <div
        v-else
        class="rounded-xl border border-dashed border-brand-cream-400 bg-brand-cream-50 p-10 text-center"
      >
        <p class="text-brand-dark-700">
          Nothing available right now — we'll announce new hatchlings to the waitlist first.
        </p>
        <Button variant="default" size="sm" class="mt-4" @click="router.push({ name: 'waitlist' })">
          Join the waitlist
        </Button>
      </div>
    </section>

    <!-- About -->
    <section class="mx-auto max-w-3xl px-4 sm:px-6 py-12 text-center">
      <span class="text-xs uppercase tracking-wider text-brand-gold-700 font-semibold">About</span>
      <h2 class="font-serif text-3xl mt-1">Bred for health, priced for keepers.</h2>
      <p class="text-brand-dark-700 mt-4">
        Zenetic is a small, transparent operation. Every gecko we sell is a holdback we chose to raise ourselves — proven eaters, well-sheds, and genetics we're proud of. Ask us anything.
      </p>
    </section>

    <!-- CTA band -->
    <section class="bg-brand-gold-100 border-y border-brand-cream-300">
      <div class="mx-auto max-w-4xl px-4 sm:px-6 py-10 text-center flex flex-col items-center gap-4">
        <h2 class="font-serif text-2xl sm:text-3xl text-brand-dark-950">
          Want first dibs on our next clutch?
        </h2>
        <Button size="lg" @click="router.push({ name: 'waitlist' })">
          Join the waitlist
          <ArrowRight class="size-4" />
        </Button>
      </div>
    </section>

    <SiteFooter />
  </div>
</template>
```

- [ ] **Step 2: Replace `GeckosView.vue` with the full list**

```vue
<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton';
import { Button } from '@/components/ui/button';
import { useRouter } from 'vue-router';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import PublicGeckoCard from '@/components/PublicGeckoCard.vue';
import { usePublicGeckos } from '@/composables/usePublicGeckos';

const router = useRouter();
const { data, isLoading, isError } = usePublicGeckos();
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <main class="mx-auto max-w-6xl w-full px-4 sm:px-6 py-10 flex-1">
      <div class="flex flex-col gap-2 mb-8">
        <span class="text-xs uppercase tracking-wider text-brand-gold-700 font-semibold">Collection</span>
        <h1 class="font-serif text-4xl">Currently available</h1>
        <p class="text-brand-dark-700">Every gecko shown here is ready to ship (locally) or pickup in Phnom Penh.</p>
      </div>

      <div v-if="isLoading" class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <Skeleton v-for="n in 6" :key="n" class="h-80 rounded-xl" />
      </div>

      <div
        v-else-if="isError"
        class="rounded-xl border border-red-200 bg-red-50 p-6 text-center text-red-900"
      >
        Couldn't load the collection. Please refresh.
      </div>

      <div
        v-else-if="data && data.geckos.length"
        class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-5"
      >
        <PublicGeckoCard v-for="g in data.geckos" :key="g.code" :gecko="g" />
      </div>

      <div
        v-else
        class="rounded-xl border border-dashed border-brand-cream-400 bg-brand-cream-50 p-10 text-center flex flex-col items-center gap-4"
      >
        <p class="text-brand-dark-700">
          Nothing available right now — we'll announce new hatchlings to the waitlist first.
        </p>
        <Button @click="router.push({ name: 'waitlist' })">Join the waitlist</Button>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>
```

- [ ] **Step 3: Build**

Run: `cd /home/zen/dev/project_gekko/apps/storefront && bun run build`
Expected: success.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/storefront/src/views/HomeView.vue apps/storefront/src/views/GeckosView.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(storefront): home + geckos list views

Home is a four-section landing page — hero, available-now teaser
(useAvailableTeaser(3)), about, CTA band. The teaser has a dedicated
empty state that funnels straight to the waitlist when no geckos are
AVAILABLE. /geckos lists the full available grid with loading
skeletons, error retry, and the same funnel empty state.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Storefront — GeckoDetailView + WaitlistView

**Files:**
- Modify: `apps/storefront/src/views/GeckoDetailView.vue`
- Modify: `apps/storefront/src/views/WaitlistView.vue`

- [ ] **Step 1: Replace `GeckoDetailView.vue`**

```vue
<script setup lang="ts">
import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import { ArrowLeft, ArrowRight, Mars, Venus, HelpCircle } from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import LowPolyGecko from '@/components/art/LowPolyGecko.vue';
import { usePublicGecko } from '@/composables/usePublicGeckos';
import { ageFromBirth, formatDate } from '@/lib/format';

const props = defineProps<{ code: string }>();
const router = useRouter();
const { data: gecko, isLoading, isError, error } = usePublicGecko(computed(() => props.code));

const selectedIdx = ref(0);
const mainPhoto = computed(() => {
  if (!gecko.value || !gecko.value.photos.length) return null;
  return gecko.value.photos[selectedIdx.value]?.url ?? gecko.value.photos[0].url;
});

const sexIcon = computed(() => {
  if (!gecko.value) return HelpCircle;
  const m = { M: Mars, F: Venus, U: HelpCircle } as const;
  return m[gecko.value.sex];
});

const sexLabel = computed(() => {
  if (!gecko.value) return '';
  const m = { M: 'Male', F: 'Female', U: 'Unsexed' } as const;
  return m[gecko.value.sex];
});

function onJoinWaitlist() {
  router.push({ name: 'waitlist', query: { interested_in: props.code } });
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <main class="mx-auto max-w-6xl w-full px-4 sm:px-6 py-10 flex-1">
      <!-- Loading -->
      <div v-if="isLoading" class="flex flex-col gap-4">
        <Skeleton class="h-6 w-28" />
        <Skeleton class="h-80 w-full rounded-xl" />
        <Skeleton class="h-10 w-1/2" />
      </div>

      <!-- 404 / error -->
      <div
        v-else-if="isError"
        class="rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-10 text-center flex flex-col items-center gap-4"
      >
        <LowPolyGecko :size="160" class="animate-float opacity-70" />
        <h1 class="font-serif text-3xl">Not available</h1>
        <p class="text-brand-dark-700">
          This gecko isn't in the current collection. It may have been reserved or sold.
        </p>
        <Button @click="router.push({ name: 'geckos' })">
          <ArrowLeft class="size-4" /> Back to available
        </Button>
      </div>

      <!-- Success -->
      <div v-else-if="gecko" class="flex flex-col gap-10">
        <!-- Breadcrumb -->
        <nav class="text-sm text-brand-dark-600 flex items-center gap-2">
          <RouterLink :to="{ name: 'geckos' }" class="hover:text-brand-gold-700">Geckos</RouterLink>
          <span>/</span>
          <span class="text-brand-dark-800">{{ gecko.code }}</span>
        </nav>

        <!-- Hero -->
        <section class="grid grid-cols-1 md:grid-cols-[1fr_360px] gap-8">
          <div class="flex flex-col gap-3">
            <div class="relative aspect-[4/3] rounded-xl overflow-hidden border border-brand-cream-300 bg-brand-cream-100">
              <img
                v-if="mainPhoto"
                :src="mainPhoto"
                :alt="gecko.name"
                class="w-full h-full object-cover"
              />
              <div v-else class="w-full h-full flex items-center justify-center">
                <LowPolyGecko :size="260" />
              </div>
            </div>
            <div v-if="gecko.photos.length > 1" class="flex gap-2 overflow-x-auto">
              <button
                v-for="(p, i) in gecko.photos"
                :key="i"
                type="button"
                class="size-20 rounded-md overflow-hidden border-2 transition-colors shrink-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-gold-500"
                :class="selectedIdx === i ? 'border-brand-gold-600' : 'border-brand-cream-300 hover:border-brand-cream-400'"
                @click="selectedIdx = i"
              >
                <img :src="p.url" :alt="p.caption" class="w-full h-full object-cover" />
              </button>
            </div>
          </div>

          <!-- Facts -->
          <aside class="flex flex-col gap-4">
            <div>
              <span class="text-xs uppercase tracking-wider text-brand-dark-600 font-semibold">{{ gecko.code }}</span>
              <h1 class="font-serif text-4xl mt-1">{{ gecko.name || 'Unnamed' }}</h1>
              <p class="text-brand-dark-700 mt-2">
                {{ gecko.species_name }} · <span class="text-brand-dark-950 font-medium">{{ gecko.morph }}</span>
              </p>
            </div>

            <dl class="grid grid-cols-2 gap-3 py-4 border-y border-brand-cream-200">
              <div>
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Sex</dt>
                <dd class="font-serif text-lg flex items-center gap-1.5">
                  <component :is="sexIcon" class="size-4" stroke-width="2" />
                  {{ sexLabel }}
                </dd>
              </div>
              <div v-if="gecko.hatch_date">
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Age</dt>
                <dd class="font-serif text-lg">{{ ageFromBirth(gecko.hatch_date) }}</dd>
              </div>
              <div v-if="gecko.hatch_date">
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Hatched</dt>
                <dd class="font-serif text-lg">{{ formatDate(gecko.hatch_date) }}</dd>
              </div>
              <div v-if="gecko.list_price_usd">
                <dt class="text-[11px] uppercase tracking-wider text-brand-dark-600">Price</dt>
                <dd class="font-serif text-2xl text-brand-gold-700">${{ gecko.list_price_usd }}</dd>
              </div>
            </dl>

            <Button size="lg" class="w-full" @click="onJoinWaitlist">
              Interested? Join the waitlist
              <ArrowRight class="size-4" />
            </Button>
          </aside>
        </section>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>
```

- [ ] **Step 2: Replace `WaitlistView.vue`**

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRoute } from 'vue-router';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { CheckCircle2, ArrowRight } from 'lucide-vue-next';
import SiteHeader from '@/components/SiteHeader.vue';
import SiteFooter from '@/components/SiteFooter.vue';
import LowPolyAccent from '@/components/art/LowPolyAccent.vue';
import { useWaitlistSignup } from '@/composables/useWaitlistSignup';
import { toast } from 'vue-sonner';

const route = useRoute();
const email = ref('');
const telegram = ref('');
const phone = ref('');
const interestedIn = ref('');
const notes = ref('');

const success = ref<'created' | 'deduplicated' | null>(null);
const mutation = useWaitlistSignup();

onMounted(() => {
  const q = route.query.interested_in;
  if (typeof q === 'string' && q) {
    interestedIn.value = q;
  }
});

async function onSubmit(e: Event) {
  e.preventDefault();
  if (!email.value.trim()) {
    toast.error('Email is required.');
    return;
  }
  try {
    const res = await mutation.mutateAsync({
      email: email.value.trim(),
      telegram: telegram.value.trim() || undefined,
      phone: phone.value.trim() || undefined,
      interested_in: interestedIn.value.trim() || undefined,
      notes: notes.value.trim() || undefined,
    });
    success.value = res.deduplicated ? 'deduplicated' : 'created';
  } catch (e: unknown) {
    const msg = (e as any)?.response?.data?.error ?? (e as Error).message ?? 'Something went wrong';
    toast.error(String(msg));
  }
}
</script>

<template>
  <div class="min-h-screen flex flex-col bg-brand-cream-50 text-brand-dark-950">
    <SiteHeader />

    <main class="mx-auto max-w-2xl w-full px-4 sm:px-6 py-12 flex-1">
      <!-- Success state -->
      <section
        v-if="success"
        class="rounded-xl border border-brand-cream-300 bg-brand-cream-50 p-10 text-center flex flex-col items-center gap-4"
      >
        <CheckCircle2 class="size-12 text-brand-gold-700" />
        <h1 class="font-serif text-3xl">
          {{ success === 'created' ? 'You\'re on the list.' : 'Already there.' }}
        </h1>
        <p class="text-brand-dark-700 max-w-md">
          {{
            success === 'created'
              ? 'Thanks — we\'ll be in touch when new geckos are ready.'
              : 'It looks like you\'re already on the waitlist. We\'ll be in touch when new geckos are ready.'
          }}
        </p>
        <Button variant="outline" @click="$router.push({ name: 'home' })">
          <ArrowRight class="size-4" /> Back to home
        </Button>
      </section>

      <!-- Form state -->
      <section v-else>
        <div class="mb-6">
          <span class="inline-flex items-center gap-2 text-xs font-semibold tracking-[0.16em] uppercase text-brand-gold-700">
            <LowPolyAccent :size="16" /> Waitlist
          </span>
          <h1 class="font-serif text-4xl mt-2">Tell us who to call.</h1>
          <p class="text-brand-dark-700 mt-2">
            We'll email you when geckos matching your interest become available. No spam — one note per drop.
          </p>
        </div>

        <form class="flex flex-col gap-5" @submit="onSubmit">
          <div class="flex flex-col gap-2">
            <Label for="wl-email">Email <span class="text-destructive">*</span></Label>
            <Input id="wl-email" v-model="email" type="email" required autocomplete="email" class="bg-white" />
          </div>
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="flex flex-col gap-2">
              <Label for="wl-telegram">Telegram</Label>
              <Input id="wl-telegram" v-model="telegram" placeholder="@you" class="bg-white" />
            </div>
            <div class="flex flex-col gap-2">
              <Label for="wl-phone">Phone</Label>
              <Input id="wl-phone" v-model="phone" placeholder="+855…" class="bg-white" />
            </div>
          </div>
          <div class="flex flex-col gap-2">
            <Label for="wl-interest">Interested in (optional)</Label>
            <Input
              id="wl-interest"
              v-model="interestedIn"
              placeholder="Tangerine leopard, lilly white crested, etc."
              class="bg-white"
            />
          </div>
          <div class="flex flex-col gap-2">
            <Label for="wl-notes">Notes (optional)</Label>
            <textarea
              id="wl-notes"
              v-model="notes"
              rows="4"
              class="rounded-md border border-brand-cream-300 bg-white px-3 py-2 text-sm resize-y"
              placeholder="First-time keeper? Experienced? Preferred drop window?"
            />
          </div>
          <Button type="submit" size="lg" :disabled="mutation.isPending.value">
            {{ mutation.isPending.value ? 'Submitting…' : 'Join the waitlist' }}
          </Button>
        </form>
      </section>
    </main>

    <SiteFooter />
  </div>
</template>
```

- [ ] **Step 3: Build + smoke test**

Run: `cd /home/zen/dev/project_gekko/apps/storefront && bun run build`
Expected: success.

Quickly boot the dev server and curl through the Vite proxy:
```bash
cd /home/zen/dev/project_gekko/apps/storefront
bun run dev &
sleep 3
curl -sI http://localhost:5174/ | head -3            # expect 200
curl -sS http://localhost:5174/api/public/geckos | head -3   # expect JSON or empty array
kill %1
```

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/storefront/src/views/GeckoDetailView.vue apps/storefront/src/views/WaitlistView.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(storefront): gecko detail + waitlist signup views

Detail page: cover image swapper, facts sidebar with sex/age/hatch/price,
breadcrumb, CTA that pushes the user to /waitlist?interested_in=<code>
so the form pre-fills. 404 path shows a friendly 'not available' card.
Waitlist view: form with email/telegram/phone/interested_in/notes,
client-side required check on email, server deduplicate state distinct
from fresh-signup state, toast on error. Uses useRoute query to
pre-fill interested_in.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: End-to-end smoke + push

**Files:** None — verification + push.

- [ ] **Step 1: Full backend tests**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./...`
Expected: all packages `ok`.

- [ ] **Step 2: Admin build + tests (regression check)**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build && bun run test`
Expected: build succeeds; 7 tests PASS.

- [ ] **Step 3: Storefront build + dev server boot**

Run: `cd /home/zen/dev/project_gekko/apps/storefront && bun run build`
Expected: success.

Then boot the dev server in the background and hit the 4 routes:
```bash
cd /home/zen/dev/project_gekko/apps/storefront
nohup bun run dev > /tmp/storefront.log 2>&1 &
disown
sleep 5
for path in / /geckos /waitlist; do
  printf "%-12s → " "$path"
  curl -s -o /dev/null -w "%{http_code}\n" "http://localhost:5174$path"
done
echo "--- public API through storefront proxy ---"
curl -sS http://localhost:5174/api/public/geckos | python3 -m json.tool | head -10
```
Expected: 200s from the three routes; the API call returns the JSON shape. (Leave the dev server running — terminal-keeper also uses port 5174.)

- [ ] **Step 4: Push**

```bash
cd /home/zen/dev/project_gekko
git push origin main
```
Expected: push succeeds.

- [ ] **Step 5: Ping Zen**

Send a Telegram reply summarizing:
- All 4 storefront routes live (/, /geckos, /geckos/:code, /waitlist)
- Three new public endpoints on the backend
- Available teaser and full list connect to real /api/public/geckos
- Waitlist form posts to /api/public/waitlist with deduplication + rate-limit
- `apps/storefront` on port 5174
- Terminal-keeper tile already ready
- Commits pushed

---

## Self-Review

**1. Spec coverage:**
- Public list endpoint → Task 1, 3
- Public detail endpoint → Task 1, 3
- Public waitlist POST → Task 1, 3
- Rate-limit middleware w/ inline TODO → Task 2 (TODO included verbatim)
- Sanitized DTOs (no notes/sire/dam/status/acquired_date) → Task 3 (asserted in Task 4 tests)
- Morph composition server-side → Task 3 (`composePublicMorph`)
- Backend tests (6 per spec) → Task 4
- apps/storefront scaffold → Task 5
- Router with 4 routes → Task 5 (wired early so stubs stay valid)
- Composables (usePublicGeckos, useWaitlistSignup, useAvailableTeaser) → Task 6
- SiteHeader + SiteFooter + PublicGeckoCard → Task 7
- HomeView (hero + teaser + about + CTA band) → Task 8
- GeckosView (grid + loading/error/empty) → Task 8
- GeckoDetailView (hero + gallery + facts + CTA) → Task 9
- WaitlistView (form + success + pre-fill) → Task 9
- Vite proxy config → Task 5, exercised in Task 10 smoke
- `.ts.net` wildcard for Tailscale → Task 5
- End-to-end smoke + push → Task 10

**2. Placeholder scan:** No "TBD", "implement later", etc. Stub views in Task 5 are explicit placeholders that get REPLACED in Tasks 8-9 (so the build stays green between tasks). That's a deliberate staging, not a hole.

**3. Type consistency:**
- Backend `publicGeckoDTO.Morph` is `string`; frontend `PublicGecko.morph` is `string`. ✅
- Backend `publicGeckoDTO.ListPriceUsd` is `*string` (JSON `string | null`); frontend matches as `string | null`. ✅
- Backend `publicWaitlistResp.ID` is `*int32` → JSON `number | undefined`; frontend `WaitlistResult.id` is `number | undefined`. ✅
- Backend `CreateWaitlistEntryParams.Column5` is the sqlc-generated `any` param for the default-string column; passing `"storefront"` at the call site matches existing usage in `backend/internal/http/waitlist.go`. ✅
- Route names in the frontend (`home`, `geckos`, `gecko-detail`, `waitlist`) are referenced by exact name in every RouterLink and `router.push({ name: '...' })` call across Tasks 7, 8, 9. ✅
- Public endpoint paths (`/api/public/geckos`, `/api/public/geckos/{code}`, `/api/public/waitlist`) are referenced the same way from backend mount and frontend composables. ✅

**4. Ambiguity check:**
- "Which sex icon for Unsexed?" — `HelpCircle`, consistent with admin (Task 7, Task 9). ✅
- "Where does the Join-waitlist CTA send `interested_in`?" — query string `?interested_in=<code>`; WaitlistView reads via `useRoute().query.interested_in` (Task 9, Step 2). ✅
- "What happens on 404 from detail endpoint?" — `retry: false` in `usePublicGecko`, so `isError` fires quickly; the view shows the friendly "not available" card (Task 9, Step 1). ✅
- "Empty DB: does list show error or empty state?" — backend returns `{geckos:[], total:0}`, not an error; the view falls through to the empty-state card with a waitlist funnel (Task 8, Step 2). ✅
