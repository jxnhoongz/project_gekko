# Dashboard Real Numbers — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the admin DashboardView's mock stats, upcoming-actions panel, and recent-activity feed with live data from one new backend endpoint. Repurpose "Upcoming" as a "Needs attention" list (stale waitlist signups + stale HOLD geckos) so the panel is useful before Phase 7 husbandry lands.

**Architecture:** Single endpoint `GET /api/admin/dashboard` runs three independent sqlc queries in parallel (stats aggregate, needs-attention union, recent-activity union), composes human-facing strings in Go, returns a typed JSON payload. Frontend replaces mock imports with a TanStack Query composable; dashboard sections are driven by that one query's data.

**Tech Stack:** Go 1.25, chi/v5, pgx/v5, sqlc@v1.27, goose; Vue 3.5, TypeScript, TanStack Vue Query, shadcn-vue, Vitest, vue-tsc.

**Spec:** `docs/superpowers/specs/2026-04-21-dashboard-real-numbers-design.md`

---

## File Structure

**Backend (new):**
- `backend/internal/queries/dashboard.sql` — three queries: `DashboardStats`, `DashboardNeedsAttention`, `DashboardRecentActivity`.
- `backend/internal/http/dashboard.go` — `MountDashboard` + `getDashboard` handler, DTOs, Go-side string composition.
- `backend/internal/http/dashboard_test.go` — integration tests against real DB (matches `auth_test.go` / `waitlist_test.go` pattern).
- `backend/internal/db/dashboard.sql.go` — **generated** by `sqlc generate`, do not hand-edit.

**Backend (modified):**
- `backend/cmd/gekko/main.go` — mount the new route.

**Frontend (new):**
- `apps/admin/src/composables/useDashboard.ts` — TanStack Query composable + typed response types.

**Frontend (modified):**
- `apps/admin/src/views/DashboardView.vue` — drop mock imports, consume `useDashboard()`, render live data and link-through rows.
- `apps/admin/src/mock/history.ts` — delete the `activity` and `upcoming` exports (now unused).
- `apps/admin/src/mock/index.ts` — trim re-exports.

---

## Task 1: Write dashboard.sql with the three queries

**Files:**
- Create: `backend/internal/queries/dashboard.sql`

- [ ] **Step 1: Create the query file**

Write `backend/internal/queries/dashboard.sql`:

```sql
-- name: DashboardStats :one
SELECT
  (SELECT COUNT(*) FROM geckos)                              AS total_geckos,
  (SELECT COUNT(*) FROM geckos WHERE status = 'BREEDING')    AS breeding,
  (SELECT COUNT(*) FROM geckos WHERE status = 'AVAILABLE')   AS available,
  (SELECT COUNT(*) FROM waitlist_entries)                    AS waitlist;

-- name: DashboardNeedsAttention :many
-- Waitlist entries that have been sitting uncontacted >7 days, and
-- geckos that have been on HOLD >7 days without any status change.
-- Top 6 merged by staleness (oldest first).
(
  SELECT 'waitlist_stale'::text AS kind,
         w.id            AS ref_id,
         'waitlist'::text AS ref_kind,
         w.email         AS subject,
         COALESCE(w.interested_in, '') AS detail_hint,
         w.created_at    AS due_at
  FROM waitlist_entries w
  WHERE w.contacted_at IS NULL
    AND w.created_at < NOW() - INTERVAL '7 days'
)
UNION ALL
(
  SELECT 'hold_stale'::text AS kind,
         g.id         AS ref_id,
         'gecko'::text AS ref_kind,
         COALESCE(g.name, g.code) AS subject,
         g.code       AS detail_hint,
         g.updated_at AS due_at
  FROM geckos g
  WHERE g.status = 'HOLD'
    AND g.updated_at < NOW() - INTERVAL '7 days'
)
ORDER BY due_at ASC
LIMIT 6;

-- name: DashboardRecentActivity :many
-- Top 15 most-recent events across three sources. Keep the shape
-- identical across branches so sqlc generates a single row type.
(
  SELECT 'gecko_created'::text AS kind,
         g.id                  AS ref_id,
         'gecko'::text         AS ref_kind,
         COALESCE(g.name, g.code) AS title,
         g.code                AS detail,
         g.created_at          AS at
  FROM geckos g
)
UNION ALL
(
  SELECT 'waitlist_created'::text AS kind,
         w.id                      AS ref_id,
         'waitlist'::text          AS ref_kind,
         'New waitlist signup'     AS title,
         w.email                   AS detail,
         w.created_at              AS at
  FROM waitlist_entries w
)
UNION ALL
(
  SELECT 'media_uploaded'::text AS kind,
         m.id                   AS ref_id,
         'gecko'::text          AS ref_kind,
         'Photo added to ' || COALESCE(g.name, g.code, '(unknown)') AS title,
         ''                     AS detail,
         m.uploaded_at          AS at
  FROM media m
  LEFT JOIN geckos g ON g.id = m.gecko_id
  WHERE m.gecko_id IS NOT NULL
)
ORDER BY at DESC
LIMIT 15;
```

- [ ] **Step 2: Generate Go code**

Run: `cd /home/zen/dev/project_gekko/backend && /home/zen/go/bin/sqlc generate`
Expected: silent success; a new `backend/internal/db/dashboard.sql.go` appears.

- [ ] **Step 3: Confirm the module still builds**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/queries/dashboard.sql backend/internal/db/dashboard.sql.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): sqlc queries for dashboard endpoint

Three queries: DashboardStats (1 row, 4 counts), DashboardNeedsAttention
(up to 6 stale items), DashboardRecentActivity (up to 15 recent events).
Union branches share a common column shape so sqlc generates one row
type per query.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Write the handler file with DTOs (no tests yet)

**Files:**
- Create: `backend/internal/http/dashboard.go`

- [ ] **Step 1: Create the handler file**

Write `backend/internal/http/dashboard.go`:

```go
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountDashboard registers GET /api/admin/dashboard.
func MountDashboard(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &dashboardDeps{q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/admin/dashboard", d.get)
	})
}

type dashboardDeps struct {
	q *db.Queries
}

// ---- DTOs (JSON shapes) ----

type dashStats struct {
	TotalGeckos int64 `json:"total_geckos"`
	Breeding    int64 `json:"breeding"`
	Available   int64 `json:"available"`
	Waitlist    int64 `json:"waitlist"`
}

type dashItem struct {
	Kind    string    `json:"kind"`
	Title   string    `json:"title"`
	Detail  string    `json:"detail"`
	At      time.Time `json:"at"`
	RefKind string    `json:"ref_kind"`
	RefID   int32     `json:"ref_id"`
}

// due_at is treated as the "at" timestamp too — same field name in JSON
// so the frontend can render the same list component for both panels.
type dashResponse struct {
	Stats           dashStats  `json:"stats"`
	NeedsAttention  []dashItem `json:"needs_attention"`
	RecentActivity  []dashItem `json:"recent_activity"`
}

// ---- handler ----

func (d *dashboardDeps) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		stats      db.DashboardStatsRow
		needsRows  []db.DashboardNeedsAttentionRow
		recentRows []db.DashboardRecentActivityRow
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		s, err := d.q.DashboardStats(gctx)
		if err != nil {
			return fmt.Errorf("stats: %w", err)
		}
		stats = s
		return nil
	})
	g.Go(func() error {
		rows, err := d.q.DashboardNeedsAttention(gctx)
		if err != nil {
			return fmt.Errorf("needs_attention: %w", err)
		}
		needsRows = rows
		return nil
	})
	g.Go(func() error {
		rows, err := d.q.DashboardRecentActivity(gctx)
		if err != nil {
			return fmt.Errorf("recent_activity: %w", err)
		}
		recentRows = rows
		return nil
	})

	if err := g.Wait(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := dashResponse{
		Stats: dashStats{
			TotalGeckos: stats.TotalGeckos,
			Breeding:    stats.Breeding,
			Available:   stats.Available,
			Waitlist:    stats.Waitlist,
		},
		NeedsAttention: composeNeedsAttention(needsRows),
		RecentActivity: composeRecentActivity(recentRows),
	}
	writeJSON(w, http.StatusOK, resp)
}

// composeNeedsAttention turns raw SQL rows into user-facing title/detail
// strings. Split out as its own function so the formatting is testable.
func composeNeedsAttention(rows []db.DashboardNeedsAttentionRow) []dashItem {
	out := make([]dashItem, 0, len(rows))
	for _, r := range rows {
		days := daysAgo(r.DueAt.Time)
		item := dashItem{
			Kind:    r.Kind,
			At:      r.DueAt.Time,
			RefKind: r.RefKind,
			RefID:   r.RefID,
		}
		switch r.Kind {
		case "waitlist_stale":
			item.Title = "Follow up with " + r.Subject
			if r.DetailHint != "" {
				item.Detail = fmt.Sprintf("Waitlist · %d days since signup · %s", days, r.DetailHint)
			} else {
				item.Detail = fmt.Sprintf("Waitlist · %d days since signup", days)
			}
		case "hold_stale":
			item.Title = r.Subject + " on HOLD"
			item.Detail = fmt.Sprintf("%s · %d days", r.DetailHint, days)
		default:
			item.Title = r.Subject
			item.Detail = r.DetailHint
		}
		out = append(out, item)
	}
	return out
}

// composeRecentActivity turns raw SQL rows into dashItems with already-
// formatted titles from SQL, keeping only minor Go-side tidying.
func composeRecentActivity(rows []db.DashboardRecentActivityRow) []dashItem {
	out := make([]dashItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dashItem{
			Kind:    r.Kind,
			Title:   r.Title,
			Detail:  r.Detail,
			At:      r.At.Time,
			RefKind: r.RefKind,
			RefID:   r.RefID,
		})
	}
	return out
}

// daysAgo returns whole days between t and now (never negative).
func daysAgo(t time.Time) int {
	d := int(time.Since(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}

// ---- compile-time guard: ensure context is always used ----
var _ = context.Background
```

- [ ] **Step 2: Add the errgroup dependency**

Run: `cd /home/zen/dev/project_gekko/backend && go get golang.org/x/sync/errgroup`
Expected: `go: added golang.org/x/sync v...` (or existing version).

- [ ] **Step 3: Confirm the module builds**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/dashboard.go backend/go.mod backend/go.sum
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): dashboard handler + DTOs

getDashboard runs the three sqlc queries in parallel via errgroup,
composes title/detail strings in Go (composeNeedsAttention,
composeRecentActivity), and returns a single JSON payload. Route
not mounted yet.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Mount the route + smoke-test it

**Files:**
- Modify: `backend/cmd/gekko/main.go`

- [ ] **Step 1: Add the mount call**

Read `backend/cmd/gekko/main.go` and find the block that mounts other routers. It currently has:

```go
apihttp.MountAuth(r, pool, signer)
apihttp.MountWaitlist(r, pool, signer)
apihttp.MountSchema(r, pool, signer)
apihttp.MountGeckos(r, pool, signer)
apihttp.MountMedia(r, pool, signer, cfg)
```

Append:

```go
apihttp.MountDashboard(r, pool, signer)
```

- [ ] **Step 2: Build and make sure nothing broke**

Run: `cd /home/zen/dev/project_gekko/backend && go build ./...`
Expected: silent exit 0.

- [ ] **Step 3: Smoke-test with curl**

Run:
```bash
TOKEN=$(curl -sS -X POST http://localhost:8420/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"zen@zeneticgekkos.com","password":"gekko-dev-2026"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")
curl -sS http://localhost:8420/api/admin/dashboard \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | head -40
```
Expected: JSON with `stats.total_geckos`, `stats.waitlist` as numbers, `needs_attention` and `recent_activity` as arrays (can be empty).

If `air` isn't running the server already, the response will fail with connection refused. If so, start it: `cd /home/zen/dev/project_gekko/backend && go run ./cmd/gekko &` and retry.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/cmd/gekko/main.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(backend): mount /api/admin/dashboard route

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Handler happy-path test

**Files:**
- Create: `backend/internal/http/dashboard_test.go`

- [ ] **Step 1: Write the test**

Write `backend/internal/http/dashboard_test.go`:

```go
package http

import (
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

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func dashSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool, int32) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// Admin so we get a token.
	email := "dash+" + time.Now().Format("150405.000000000") + "@example.com"
	hash, err := auth.HashPassword("test-password-123")
	require.NoError(t, err)
	q := db.New(pool)
	admin, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email: email, PasswordHash: hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM admin_users WHERE id = $1", admin.ID)
	})

	signer := auth.NewJWTSigner("test-secret", time.Hour)
	tok, err := signer.Issue(int64(admin.ID), admin.Email)
	require.NoError(t, err)

	r := chi.NewRouter()
	MountDashboard(r, pool, signer)
	return r, tok, pool, admin.ID
}

func TestDashboard_endpoint_returnsAll(t *testing.T) {
	router, tok, _, _ := dashSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var body struct {
		Stats          dashStats  `json:"stats"`
		NeedsAttention []dashItem `json:"needs_attention"`
		RecentActivity []dashItem `json:"recent_activity"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

	assert.GreaterOrEqual(t, body.Stats.TotalGeckos, int64(0))
	assert.GreaterOrEqual(t, body.Stats.Waitlist, int64(0))
	assert.NotNil(t, body.NeedsAttention, "needs_attention must always be an array, never null")
	assert.NotNil(t, body.RecentActivity, "recent_activity must always be an array, never null")
}

func TestDashboard_endpoint_requiresAuth(t *testing.T) {
	router, _, _, _ := dashSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestComposeNeedsAttention_waitlistStale(t *testing.T) {
	rows := []db.DashboardNeedsAttentionRow{
		{
			Kind:       "waitlist_stale",
			RefID:      42,
			RefKind:    "waitlist",
			Subject:    "maly@example.com",
			DetailHint: "Crested gecko",
			DueAt:      pgtype.Timestamp{Time: time.Now().Add(-11 * 24 * time.Hour), Valid: true},
		},
	}
	out := composeNeedsAttention(rows)
	require.Len(t, out, 1)
	assert.Equal(t, "waitlist_stale", out[0].Kind)
	assert.Equal(t, "Follow up with maly@example.com", out[0].Title)
	assert.Contains(t, out[0].Detail, "Crested gecko")
	assert.Contains(t, out[0].Detail, "11 days")
	assert.Equal(t, "waitlist", out[0].RefKind)
	assert.Equal(t, int32(42), out[0].RefID)
}

func TestComposeNeedsAttention_holdStale(t *testing.T) {
	rows := []db.DashboardNeedsAttentionRow{
		{
			Kind:       "hold_stale",
			RefID:      7,
			RefKind:    "gecko",
			Subject:    "Veasna",
			DetailHint: "ZGLP-2026-003",
			DueAt:      pgtype.Timestamp{Time: time.Now().Add(-9 * 24 * time.Hour), Valid: true},
		},
	}
	out := composeNeedsAttention(rows)
	require.Len(t, out, 1)
	assert.Equal(t, "Veasna on HOLD", out[0].Title)
	assert.Contains(t, out[0].Detail, "ZGLP-2026-003")
	assert.Contains(t, out[0].Detail, "9 days")
}
```

- [ ] **Step 2: Run the tests**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./internal/http/... -run Dashboard -v`
Expected: 4 tests PASS (endpoint_returnsAll, endpoint_requiresAuth, composeNeedsAttention_waitlistStale, composeNeedsAttention_holdStale).

- [ ] **Step 3: Run the full test suite to confirm nothing else broke**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./...`
Expected: all packages report `ok`.

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add backend/internal/http/dashboard_test.go
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "test(backend): dashboard endpoint happy-path + compose helpers

Covers auth requirement, JSON shape (arrays never null), and the
waitlist_stale / hold_stale string composition in Go.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Frontend composable + types

**Files:**
- Create: `apps/admin/src/composables/useDashboard.ts`

- [ ] **Step 1: Write the composable**

Write `apps/admin/src/composables/useDashboard.ts`:

```ts
import { useQuery } from '@tanstack/vue-query';
import { api } from '@/lib/api';

export interface DashboardStats {
  total_geckos: number;
  breeding: number;
  available: number;
  waitlist: number;
}

export type DashboardRefKind = 'gecko' | 'waitlist';

export type DashboardItemKind =
  | 'waitlist_stale'
  | 'hold_stale'
  | 'gecko_created'
  | 'waitlist_created'
  | 'media_uploaded';

export interface DashboardItem {
  kind: DashboardItemKind;
  title: string;
  detail: string;
  at: string;
  ref_kind: DashboardRefKind;
  ref_id: number;
}

export interface DashboardData {
  stats: DashboardStats;
  needs_attention: DashboardItem[];
  recent_activity: DashboardItem[];
}

export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: async () => {
      const { data } = await api.get<DashboardData>('/api/admin/dashboard');
      return data;
    },
    staleTime: 30_000,
  });
}
```

- [ ] **Step 2: Type-check**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds (no TS errors).

- [ ] **Step 3: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/composables/useDashboard.ts
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(admin): useDashboard composable + types

TanStack Query query for GET /api/admin/dashboard with a 30s
staleTime. Types match the backend DTO exactly.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Rewrite DashboardView to consume live data

**Files:**
- Modify: `apps/admin/src/views/DashboardView.vue`

- [ ] **Step 1: Replace the script block**

Open `apps/admin/src/views/DashboardView.vue` and replace the entire `<script setup lang="ts">...</script>` block with:

```vue
<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import {
  Turtle,
  Heart,
  Egg,
  ClipboardList,
  Drumstick,
  Scale,
  Sparkles,
  HeartPulse,
  ArrowRight,
  CircleAlert,
  Image as ImageIcon,
  Pause,
  AlertTriangle,
} from 'lucide-vue-next';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { Skeleton } from '@/components/ui/skeleton';
import PageHeader from '@/components/layout/PageHeader.vue';
import StatCard from '@/components/layout/StatCard.vue';
import { useAuthStore } from '@/stores/auth';
import { useDashboard, type DashboardItem, type DashboardItemKind } from '@/composables/useDashboard';
import { timeAgo } from '@/lib/format';

const auth = useAuthStore();
const router = useRouter();
const { data, isLoading, isError, error, refetch } = useDashboard();

const greetingName = computed(
  () => auth.admin?.name?.split(' ')[0] || auth.admin?.email?.split('@')[0] || 'there',
);

const greeting = computed(() => {
  const h = new Date().getHours();
  if (h < 5)  return 'Up late';
  if (h < 12) return 'Good morning';
  if (h < 17) return 'Good afternoon';
  if (h < 22) return 'Good evening';
  return 'Up late';
});

const needsAttention = computed<DashboardItem[]>(() => data.value?.needs_attention ?? []);
const recentActivity = computed<DashboardItem[]>(() => data.value?.recent_activity ?? []);
const stats = computed(() => data.value?.stats);

const attentionIcon: Record<DashboardItemKind, typeof Drumstick> = {
  waitlist_stale:   ClipboardList,
  hold_stale:       Pause,
  // unused in this panel but typed for completeness
  gecko_created:    Turtle,
  waitlist_created: ClipboardList,
  media_uploaded:   ImageIcon,
};

const activityIcon: Record<DashboardItemKind, typeof Drumstick> = {
  gecko_created:    Turtle,
  waitlist_created: ClipboardList,
  media_uploaded:   ImageIcon,
  // unused in this panel but typed for completeness
  waitlist_stale:   ClipboardList,
  hold_stale:       Pause,
};

function linkFor(item: DashboardItem) {
  if (item.ref_kind === 'gecko') {
    return { name: 'gecko-detail', params: { id: item.ref_id } };
  }
  return { name: 'waitlist' };
}
</script>
```

- [ ] **Step 2: Replace the template block**

Replace the entire `<template>...</template>` block with:

```vue
<template>
  <div class="flex flex-col gap-8">
    <PageHeader
      :eyebrow="greeting"
      :title="`Hello, ${greetingName}.`"
      subtitle="Everything you need to keep the colony moving today."
    >
      <template #actions>
        <Button variant="outline" size="sm" @click="router.push({ name: 'geckos' })">
          View geckos
          <ArrowRight class="size-4" />
        </Button>
      </template>
    </PageHeader>

    <!-- Error -->
    <Card
      v-if="isError"
      class="border-red-200 bg-red-50 p-4 flex items-start gap-3 !gap-3"
    >
      <AlertTriangle class="size-5 text-red-700 shrink-0 mt-0.5" />
      <div class="flex-1 min-w-0">
        <div class="text-sm font-semibold text-red-900">Couldn't load the dashboard.</div>
        <div class="text-xs text-red-800 break-all">{{ (error as Error)?.message }}</div>
      </div>
      <Button variant="outline" size="sm" @click="refetch()">Retry</Button>
    </Card>

    <!-- Loading -->
    <section
      v-else-if="isLoading"
      class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5"
    >
      <Skeleton v-for="n in 4" :key="n" class="h-28 rounded-xl" />
    </section>

    <!-- Stat grid -->
    <section
      v-else-if="stats"
      class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-5"
    >
      <StatCard
        label="Total geckos"
        :value="stats.total_geckos"
        :icon="Turtle"
        :delta="`${stats.breeding} breeding`"
        delta-tone="up"
        hint="Active in collection"
      />
      <StatCard
        label="Breeding"
        :value="stats.breeding"
        :icon="Heart"
        hint="Currently pairing or holdback"
      />
      <StatCard
        label="Available"
        :value="stats.available"
        :icon="Egg"
        hint="Listed for sale"
      />
      <StatCard
        label="Waitlist"
        :value="stats.waitlist"
        :icon="ClipboardList"
        hint="Interested buyers"
      />
    </section>

    <!-- Needs attention + Recent activity -->
    <section class="grid grid-cols-1 lg:grid-cols-5 gap-6" v-if="!isError">
      <!-- Needs attention -->
      <Card class="lg:col-span-3 border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
        <div class="flex items-center justify-between px-6 py-5">
          <div class="flex flex-col">
            <h2 class="font-serif text-xl text-brand-dark-950">Needs attention</h2>
            <p class="text-xs text-brand-dark-600 mt-0.5">
              Stale waitlist signups and long-held geckos
            </p>
          </div>
          <Badge variant="soft">{{ needsAttention.length }}</Badge>
        </div>
        <Separator />

        <div v-if="isLoading" class="p-6 flex flex-col gap-3">
          <Skeleton v-for="n in 3" :key="n" class="h-14 w-full" />
        </div>

        <ul v-else-if="needsAttention.length" class="divide-y divide-brand-cream-200">
          <li
            v-for="item in needsAttention"
            :key="`${item.kind}-${item.ref_id}`"
            class="flex items-center gap-4 px-6 py-4 hover:bg-brand-cream-100/60 transition-colors cursor-pointer"
            @click="router.push(linkFor(item))"
          >
            <div
              class="flex size-10 shrink-0 items-center justify-center rounded-lg bg-brand-gold-100 text-brand-gold-800"
            >
              <component :is="attentionIcon[item.kind]" class="size-5" stroke-width="1.75" />
            </div>
            <div class="flex flex-col min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="font-medium text-brand-dark-950 text-sm">{{ item.title }}</span>
                <Badge variant="warn" class="gap-1">
                  <CircleAlert class="size-3" /> Stale
                </Badge>
              </div>
              <span class="text-xs text-brand-dark-600">{{ item.detail }}</span>
            </div>
            <span class="text-xs font-medium shrink-0 text-brand-dark-600">{{ timeAgo(item.at) }}</span>
          </li>
        </ul>

        <div v-else class="px-6 py-10 text-sm text-brand-dark-500 text-center">
          Nothing needs your attention right now.
        </div>
      </Card>

      <!-- Recent activity -->
      <Card class="lg:col-span-2 border-brand-cream-300 bg-brand-cream-50 !p-0 !gap-0 overflow-hidden">
        <div class="flex items-center justify-between px-6 py-5">
          <div class="flex flex-col">
            <h2 class="font-serif text-xl text-brand-dark-950">Recent activity</h2>
            <p class="text-xs text-brand-dark-600 mt-0.5">Last few days across the colony</p>
          </div>
        </div>
        <Separator />

        <div v-if="isLoading" class="p-6 flex flex-col gap-3">
          <Skeleton v-for="n in 5" :key="n" class="h-8 w-full" />
        </div>

        <ol v-else-if="recentActivity.length" class="flex flex-col gap-5 p-6">
          <li
            v-for="a in recentActivity"
            :key="`${a.kind}-${a.ref_id}-${a.at}`"
            class="relative flex gap-3 pl-8 cursor-pointer hover:bg-brand-cream-100/40 -mx-2 px-2 rounded"
            @click="router.push(linkFor(a))"
          >
            <span
              class="absolute left-0 top-0.5 flex size-6 items-center justify-center rounded-full bg-brand-gold-100 text-brand-gold-800 border border-brand-gold-200"
            >
              <component :is="activityIcon[a.kind]" class="size-3.5" stroke-width="2" />
            </span>
            <div class="flex flex-col min-w-0 flex-1">
              <span class="text-sm font-medium text-brand-dark-950">{{ a.title }}</span>
              <span v-if="a.detail" class="text-xs text-brand-dark-600">{{ a.detail }}</span>
            </div>
            <span class="text-xs text-brand-dark-500 shrink-0">{{ timeAgo(a.at) }}</span>
          </li>
        </ol>

        <div v-else class="px-6 py-10 text-sm text-brand-dark-500 text-center">
          Nothing logged yet.
        </div>
      </Card>
    </section>
  </div>
</template>
```

- [ ] **Step 2: Type-check**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds, no TS errors.

- [ ] **Step 3: Run the existing vitest suite**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run test`
Expected: 7 tests PASS (no new tests yet, just confirming nothing regressed).

- [ ] **Step 4: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/views/DashboardView.vue
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "feat(admin): dashboard driven by /api/admin/dashboard

- 4 stat cards driven by real counts (total, breeding, available,
  waitlist). \"Active pairings\" and \"Eggs incubating\" replaced
  until Phase 8 lands those tables.
- \"Upcoming\" panel renamed \"Needs attention\" — lists stale
  waitlist signups and long-held geckos, each row routes to the
  relevant page on click.
- Recent activity feed now shows real gecko creations, waitlist
  signups, and photo uploads, each row routes on click.
- Loading shows skeletons; error shows a retry banner; empty
  sections show friendly one-liners.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Clean up the now-unused mock data

**Files:**
- Modify: `apps/admin/src/mock/history.ts`
- Modify: `apps/admin/src/mock/index.ts`

- [ ] **Step 1: Remove `activity` and `upcoming` from history.ts**

Open `apps/admin/src/mock/history.ts` and delete the two exported constants:

```ts
export const activity: ActivityItem[] = [ ... ];
export const upcoming: UpcomingAction[] = [ ... ];
```

Also remove their associated type imports `ActivityItem`, `UpcomingAction` from the top import line if no other symbol in the file uses them (it's OK to keep them if other exports still reference). Keep `feedings`, `weights`, `sheds`, `healthLogs` — they're cheap and will be useful as examples when Phase 7 lands.

- [ ] **Step 2: Trim the re-export in index.ts**

Open `apps/admin/src/mock/index.ts` and update the re-export line from:

```ts
export { feedings, weights, sheds, healthLogs, activity, upcoming } from './history';
```

to:

```ts
export { feedings, weights, sheds, healthLogs } from './history';
```

- [ ] **Step 3: Build to confirm nothing still imports `activity` or `upcoming`**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build`
Expected: build succeeds.

- [ ] **Step 4: Run tests**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run test`
Expected: 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/zen/dev/project_gekko
git add apps/admin/src/mock/history.ts apps/admin/src/mock/index.ts
git -c user.name="jxnhoongz" -c user.email="vatanahan09@gmail.com" commit -m "chore(admin): drop unused dashboard mock data

'activity' and 'upcoming' were only used by the old DashboardView;
that data now comes from the backend. Keep feedings/weights/sheds/
healthLogs — they're cheap and useful as examples when Phase 7 lands.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: End-to-end smoke + push

**Files:** None — verification + push.

- [ ] **Step 1: Full backend tests**

Run: `cd /home/zen/dev/project_gekko/backend && go test ./...`
Expected: all packages `ok` (including dashboard tests).

- [ ] **Step 2: Full admin build + tests**

Run: `cd /home/zen/dev/project_gekko/apps/admin && bun run build && bun run test`
Expected: build succeeds; 7 tests PASS.

- [ ] **Step 3: Manual live smoke via Vite proxy**

Run:
```bash
TOKEN=$(curl -sS -X POST http://localhost:5173/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"zen@zeneticgekkos.com","password":"gekko-dev-2026"}' \
  | python3 -c "import sys, json; print(json.load(sys.stdin)['token'])")
curl -sS http://localhost:5173/api/admin/dashboard \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | head -30
```
Expected: response shows real stats (total_geckos ≥ 6, waitlist ≥ 5), and `recent_activity` lists several gecko_created / waitlist_created / media_uploaded entries with real timestamps.

- [ ] **Step 4: Push**

Run:
```bash
cd /home/zen/dev/project_gekko
git push origin main
```
Expected: push succeeds.

- [ ] **Step 5: Confirmation ping**

Report to the user:

> Dashboard real-numbers landed. 6 commits pushed (spec + queries + handler + route mount + tests + composable + view + mock cleanup). Refresh the dashboard — stat cards, "Needs attention", and "Recent activity" are all driven live from the DB now. Click any row in either panel to jump to the relevant gecko or the waitlist.

---

## Self-Review

**1. Spec coverage:**
- Stats endpoint → Tasks 1, 2
- Needs-attention endpoint → Tasks 1, 2
- Recent-activity endpoint → Tasks 1, 2
- Go-side title/detail composition → Task 2 (`composeNeedsAttention`, `composeRecentActivity`) + Task 4 tests
- Route mount → Task 3
- Endpoint tests → Task 4
- `useDashboard()` composable + types → Task 5
- DashboardView rewrite (stats + panels + loading + error + empty) → Task 6
- Click-through navigation → Task 6 (`linkFor`)
- Mock cleanup → Task 7
- Rollout (commits grouped, push together) → Task 8

**2. Placeholder scan:** Every step shows the actual code, paths, and commands. No "TBD" or "add error handling". ✅

**3. Type consistency:**
- `DashboardItem.kind` is the union defined in Task 5; the two `Record<DashboardItemKind, ...>` maps in Task 6 cover every variant.
- Backend `dashItem.RefID` is `int32`, frontend `DashboardItem.ref_id` is `number` — Vue Router coerces when building params. ✅
- `composeNeedsAttention` input type `db.DashboardNeedsAttentionRow` and `composeRecentActivity` input `db.DashboardRecentActivityRow` are generated by sqlc in Task 1; field names (`Kind`, `Title`, `Subject`, `DetailHint`, `DueAt`, `At`, `RefID`, `RefKind`) match the query column aliases one-to-one.
- `dashStats.TotalGeckos int64` matches PG's COUNT return via `db.DashboardStatsRow.TotalGeckos int64`. ✅

**4. Ambiguity check:**
- "Which of `hatch_date`-style or `created_at`-style timestamps?" — explicit: spec + plan use `created_at` / `updated_at` / `uploaded_at` per table.
- "Does empty `needs_attention` return `[]` or `null`?" — Task 4 test asserts array never null; Go's `make([]dashItem, 0, ...)` in `composeNeedsAttention` guarantees `[]`. ✅
