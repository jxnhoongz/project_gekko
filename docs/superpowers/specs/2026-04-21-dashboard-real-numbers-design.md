# Dashboard — real numbers

> **Goal:** Replace the admin DashboardView's mock stats, upcoming-actions panel, and recent-activity feed with live data sourced from the existing database. Repurpose the Upcoming Actions panel as a "Needs attention" list so it's useful before Phase 7 (husbandry) ships.

## Scope

**In scope**

- A single backend endpoint `GET /api/admin/dashboard` that returns stats, needs-attention items, and recent activity in one round trip.
- Replace mocked `stats`, `upcoming`, and `activity` consumption in `apps/admin/src/views/DashboardView.vue` with live data fetched via TanStack Query.
- Click-through navigation from needs-attention and activity rows to the relevant gecko or waitlist page.

**Out of scope**

- Geckos list, detail, schema, waitlist views — untouched.
- Active pairings, eggs incubating, husbandry-driven "upcoming feedings/weigh-ins" — stay deferred to Phase 7 / Phase 8 as the underlying tables don't exist.
- Configurable thresholds (the 7-day cutoffs are hardcoded for this pass; a knob can come later if operator wants different numbers).
- Retention / pagination on activity — first 15 rows, period.

## Backend

### New endpoint

```
GET /api/admin/dashboard         (admin-auth)
```

Response shape:

```json
{
  "stats": {
    "total_geckos": 6,
    "breeding": 2,
    "available": 1,
    "waitlist": 5
  },
  "needs_attention": [
    {
      "kind": "waitlist_stale",
      "title": "Follow up with Kosal Ly",
      "detail": "Waitlist · 11 days since signup · Crested gecko",
      "due_at": "2026-04-10T14:06:33Z",
      "ref_kind": "waitlist",
      "ref_id": 5
    },
    {
      "kind": "hold_stale",
      "title": "Veasna on HOLD",
      "detail": "ZGLP-2026-003 · 9 days",
      "due_at": "2026-04-12T00:00:00Z",
      "ref_kind": "gecko",
      "ref_id": 6
    }
  ],
  "recent_activity": [
    {
      "kind": "gecko_created",
      "title": "Apsara",
      "detail": "ZGLP-2026-001",
      "at": "2026-04-21T01:33:49Z",
      "ref_kind": "gecko",
      "ref_id": 1
    },
    {
      "kind": "waitlist_created",
      "title": "New waitlist signup",
      "detail": "kosal.ly@example.com",
      "at": "2026-04-20T14:06:33Z",
      "ref_kind": "waitlist",
      "ref_id": 5
    },
    {
      "kind": "media_uploaded",
      "title": "Photo added to Apsara",
      "detail": "",
      "at": "2026-04-21T04:35:17Z",
      "ref_kind": "gecko",
      "ref_id": 1
    }
  ]
}
```

### Queries (sqlc)

Three `:one`/`:many` queries backing the endpoint, all in `backend/internal/queries/dashboard.sql`.

**Stats — one query with subselects:**
```sql
-- name: DashboardStats :one
SELECT
  (SELECT COUNT(*) FROM geckos)                                   AS total_geckos,
  (SELECT COUNT(*) FROM geckos WHERE status = 'BREEDING')         AS breeding,
  (SELECT COUNT(*) FROM geckos WHERE status = 'AVAILABLE')        AS available,
  (SELECT COUNT(*) FROM waitlist_entries)                         AS waitlist;
```

**Needs attention — UNION ALL merge, sorted by staleness:**
```sql
-- name: DashboardNeedsAttention :many
(
  SELECT 'waitlist_stale' AS kind,
         w.id            AS ref_id,
         'waitlist'      AS ref_kind,
         w.email         AS subject,
         w.interested_in AS detail_hint,
         w.created_at    AS due_at
  FROM waitlist_entries w
  WHERE w.contacted_at IS NULL
    AND w.created_at < NOW() - INTERVAL '7 days'
)
UNION ALL
(
  SELECT 'hold_stale' AS kind,
         g.id         AS ref_id,
         'gecko'      AS ref_kind,
         COALESCE(g.name, g.code) AS subject,
         g.code       AS detail_hint,
         g.updated_at AS due_at
  FROM geckos g
  WHERE g.status = 'HOLD'
    AND g.updated_at < NOW() - INTERVAL '7 days'
)
ORDER BY due_at ASC
LIMIT 6;
```

**Recent activity — UNION ALL across three sources:**
```sql
-- name: DashboardRecentActivity :many
(
  SELECT 'gecko_created' AS kind,
         g.id            AS ref_id,
         'gecko'         AS ref_kind,
         COALESCE(g.name, g.code) AS title,
         g.code          AS detail,
         g.created_at    AS at
  FROM geckos g
)
UNION ALL
(
  SELECT 'waitlist_created' AS kind,
         w.id              AS ref_id,
         'waitlist'        AS ref_kind,
         'New waitlist signup' AS title,
         w.email           AS detail,
         w.created_at      AS at
  FROM waitlist_entries w
)
UNION ALL
(
  SELECT 'media_uploaded' AS kind,
         m.id             AS ref_id,
         'gecko'          AS ref_kind,
         'Photo added to ' || COALESCE(g.name, g.code, '(unknown)') AS title,
         ''               AS detail,
         m.uploaded_at    AS at
  FROM media m
  LEFT JOIN geckos g ON g.id = m.gecko_id
  WHERE m.gecko_id IS NOT NULL
)
ORDER BY at DESC
LIMIT 15;
```

### Handler

`backend/internal/http/dashboard.go` mounts at `MountDashboard(r, pool, signer)`:

- Single handler `getDashboard` runs the three queries in parallel via goroutines + errgroup (stats / needs_attention / recent_activity are independent).
- Maps rows to DTOs.
- Composes the response JSON.
- On any sub-query error: return 500 with the offending error surface, don't try to partial-render.

The SQL returns raw columns (`subject`, `detail_hint`, `due_at`). Human-facing strings — `title` (e.g. "Follow up with Kosal Ly", "Veasna on HOLD"), and `detail` (e.g. "Waitlist · 11 days since signup · Crested gecko", "ZGLP-2026-003 · 9 days") — are composed in the Go handler, not SQL. Keeps the SQL portable and the formatting testable.

### Testing

One integration test per DB-backed query plus one handler happy-path test:

- `TestDashboardStats_happyPath` — seeds a mix of statuses, asserts counts
- `TestDashboardNeedsAttention` — creates a stale waitlist + stale hold, asserts both appear
- `TestDashboard_endpoint` — full route, asserts JSON shape + auth required

Fixtures reset per test (matches existing `auth_test.go` pattern).

## Frontend

### Composable

`apps/admin/src/composables/useDashboard.ts`:

```ts
export function useDashboard() {
  return useQuery({
    queryKey: ['dashboard'],
    queryFn: async () => (await api.get('/api/admin/dashboard')).data,
    staleTime: 30_000,
  });
}
```

### `DashboardView.vue` changes

Remove all mock imports (`upcoming`, `activity`, `geckos` from `@/mock`). Replace with `useDashboard()` data.

**Stat cards (4 slots):**
- Total geckos → `stats.total_geckos`, hint: `${breeding} breeding`
- Active pairings → **replaced** by Breeding (`stats.breeding`), hint "Breeding-status geckos"
- Eggs incubating → **replaced** by Available (`stats.available`), hint "Listed for sale"
- Waitlist → `stats.waitlist`, hint "Interested buyers"

**Needs attention panel:**
- Header stays "Needs attention" (eyebrow change from "Upcoming today")
- Each row: icon chip (waitlist = `ClipboardList`, hold = `Pause`), title, detail, `timeAgo(due_at)`
- Row is a RouterLink to `/waitlist` (anchor to entry if possible) or `/geckos/:id`
- Empty state: "Nothing needs your attention right now."

**Recent activity:**
- Each row renders with a kind-specific icon (`gecko_created` → `Turtle`, `waitlist_created` → `ClipboardList`, `media_uploaded` → `Image`)
- Clickable → routes per `ref_kind`/`ref_id`
- Empty state: "Nothing logged yet."

**Loading / error:**
- Loading: 4 stat-card skeletons + two card-shaped skeletons for the panels
- Error: single red banner at the top of the dashboard with a retry button; stat grid hidden

### Mock cleanup

After the rewrite, `activity` and `upcoming` in `apps/admin/src/mock/history.ts` are unused. Remove them and trim the re-exports in `mock/index.ts`. `feedings`, `weights`, `sheds`, `healthLogs` exports stay — they're still referenced by nothing (GeckoDetailView moved to empty states in Phase 3), but they're cheap to keep and will be useful as seed examples when Phase 7 ships. Types live in `src/types/index.ts` and aren't affected.

## Data flow

```
┌────────────────────┐     GET /api/admin/dashboard     ┌──────────────────┐
│  DashboardView.vue │ ───────────────────────────────▶ │  dashboard.go    │
│   useDashboard()   │                                  │  getDashboard    │
└────────────────────┘                                  └─────┬────────────┘
                                                              │ (parallel)
                             ┌────────────────────────────────┼────────────────────────────┐
                             ▼                                ▼                            ▼
                  ┌──────────────────────┐      ┌─────────────────────────┐    ┌─────────────────────┐
                  │  DashboardStats      │      │  DashboardNeedsAttention│    │  DashboardRecentActi │
                  │  (1 row, 4 counts)   │      │  (up to 6)              │    │  vity (up to 15)     │
                  └──────────────────────┘      └─────────────────────────┘    └─────────────────────┘
```

## Error handling

- **Sub-query error:** 500 with the first error message exposed (matches existing pattern in `listGeckos`). No partial responses — partials mask bugs.
- **Empty sources:** each section has a friendly empty state in the UI, no errors.
- **Auth missing/invalid:** `RequireAuth` middleware returns 401 before any DB work; handler unreachable without token.

## Testing

**Backend (go test)**
- `TestDashboardStats_happyPath`
- `TestDashboardNeedsAttention_mixed`
- `TestDashboard_endpoint_returnsAll`
- `TestDashboard_endpoint_requiresAuth`

**Admin (vitest)**
- Snapshot render test: dashboard with fake query state → matches golden markup (minimal, mostly a smoke test)

## Rollout

- One backend commit (queries + handler + tests).
- One admin commit (composable + DashboardView + mock cleanup).
- Pushed together so the frontend never runs against a stale backend.
