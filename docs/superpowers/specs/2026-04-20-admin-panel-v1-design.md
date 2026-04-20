# Admin Panel v1 — Design Spec

| | |
|---|---|
| **Date** | 2026-04-20 |
| **Status** | Draft — pending user review |
| **Primary repo** | `gekko-admin/` |
| **Touches** | `gekko_backend/` (new endpoints), `zenetic-gekko/` (waitlist form integration) |
| **Related skill** | [`gekko-design`](/.claude/skills/gekko-design/SKILL.md) |

---

## 1. Context

Zenetic Gekkos is a pre-breeding gecko business in Phnom Penh. Current state: **6 foundational geckos** (2♂/2♀ Leopard, 1♂/1♀ African Fat-tailed), no active breeding, no sales. Business strategy: **build audience via TikTok → direct traffic to storefront → capture waitlist → sell when hatchlings drop**.

The admin panel is the operator's daily workspace. Previously only the login screen existed; the dashboard was a placeholder. This spec defines **the minimum admin that lets the operator actually run the business day-to-day**, while deliberately avoiding features that only matter once breeding begins.

## 2. Goals

- Operator can add, edit, and photograph each gecko without writing SQL.
- Operator can see who signed up for the waitlist and contact them.
- Operator can see a simple "state of the business" dashboard at a glance.
- Operator can browse any table in the database (read-only) for quick lookups, debugging, and CSV export.
- All UI follows the [`gekko-design`](/.claude/skills/gekko-design/SKILL.md) skill — no drift.

## 3. Non-goals (explicit)

- No pairings / clutches / eggs UI (pre-breeding — premature).
- No standalone media library (per-gecko photos are sufficient).
- No supplies inventory, purchase orders, suppliers, or shipments UI (different business, out of focus).
- No admin-side editing of data via the visualizer (read-only only).
- No multi-user collaboration features (single operator for foreseeable future).
- No multilingual UI for admin v1 (admin is internal, operator speaks English).
- No mobile-optimized layouts (admin is desktop-first).

## 4. Scope summary

**Admin v1 = 5 routes** (4 new or polished + existing login):

| Route | Page | State |
|---|---|---|
| `/login` | Login | Existing, keep as-is |
| `/` | Dashboard | Upgrade placeholder with real counts |
| `/geckos` | Geckos list | **New** |
| `/geckos/new` + `/geckos/:id` | Gecko add / edit (incl. photos + genetics) | **New** |
| `/waitlist` | Waitlist viewer + CSV export | **New** |
| `/data` | Data visualizer (read-only, admin-only) | **New** |

Nav order (sidebar or top bar): **Dashboard · Geckos · Waitlist · Data**.

## 5. User & roles

Only one human uses admin v1: **the operator** (role = `ADMIN`). Backend already supports `ADMIN` / `STAFF` / `PUBLIC`. We respect the existing role system:

- `ADMIN` — can access everything including `/data`.
- `STAFF` — can access everything *except* `/data` (reserved for future hires).
- `PUBLIC` — cannot log into admin at all.

Authorization enforced both at the API layer (existing `authMiddleware` + a new `adminOnlyMiddleware`) and in the Vue router (via `meta.requiresAdmin`).

## 6. Architecture overview

```
[zenetic-gekko storefront]  --(POST /api/public/waitlist)-->  [gekko_backend]  <--(REST, JWT)--  [gekko-admin SPA]
                                                                   │
                                                                   ├── Postgres (existing schema + new waitlist_entries)
                                                                   └── filesystem /uploads/geckos/<id>/... (v1)
                                                                                 swap to Cloudflare R2 in v2
```

- **Frontend**: Existing Vue 3 + Vite + Tailwind + Pinia + Axios stack. No new libraries beyond what's already in `package.json`.
- **Backend**: Existing Hono + Drizzle + Postgres. Add multipart middleware for photo upload (one dependency: `@hono/multipart` or equivalent).
- **No GraphQL, no tRPC, no new state libraries.** Keep the stack boring.

## 7. Data model changes

### 7.1 New table: `waitlist_entries`

```sql
CREATE TABLE waitlist_entries (
  id            SERIAL PRIMARY KEY,
  email         VARCHAR(255) NOT NULL,
  telegram      VARCHAR(100),
  phone         VARCHAR(32),
  interested_in VARCHAR(100),     -- free-text tag, e.g. "leopard-2026-clutch"
  source        VARCHAR(50) DEFAULT 'website',
  notes         TEXT,
  contacted_at  TIMESTAMP,
  created_at    TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at    TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE UNIQUE INDEX waitlist_entries_email_idx ON waitlist_entries (LOWER(email));
CREATE INDEX waitlist_entries_created_idx ON waitlist_entries (created_at DESC);
```

- `email` is the canonical identifier — one entry per unique (lowercased) email.
- Re-submission from the storefront is idempotent (update existing record, preserve `created_at`).

### 7.2 No other schema changes in v1

The 28 existing tables cover everything else. `media` table already exists (URL-based) — admin will populate `url` with paths to our uploaded files.

## 8. API surface

### 8.1 New endpoints

| Method | Path | Auth | Purpose |
|---|---|---|---|
| `POST` | `/api/public/waitlist` | public, rate-limited | Storefront form submit |
| `GET`  | `/api/admin/waitlist` | staff+ | List entries, paginated, sortable |
| `POST` | `/api/admin/waitlist/:id/contacted` | staff+ | Mark entry as contacted (sets `contacted_at`) |
| `GET`  | `/api/admin/waitlist/export.csv` | staff+ | CSV export |
| `PUT`  | `/api/geckos/:id` | staff+ | Update gecko (currently missing) |
| `POST` | `/api/geckos/:id/photos` | staff+ | **Multipart** upload (one or more files) |
| `DELETE` | `/api/geckos/:id/photos/:mediaId` | staff+ | Delete a photo |
| `GET`  | `/api/admin/dashboard/stats` | staff+ | Dashboard counts |
| `GET`  | `/api/admin/data/:table` | **admin only** | Generic table browse (paginate, sort, filter) |
| `GET`  | `/api/admin/data/:table/export.csv` | **admin only** | CSV export for that table |

### 8.2 Existing endpoints used as-is

Geckos list, detail, delete; species; genetic_dictionary; geckos/:id/genes (GET, POST, DELETE); geckos/:id/media (GET).

### 8.3 Dashboard stats shape

```json
{
  "totalGeckos": 6,
  "byStatus": { "ACTIVE": 6, "SOLD": 0, "INACTIVE": 0 },
  "bySpecies": { "Leopard": 4, "African Fat-tailed": 2 },
  "bySex":     { "M": 3, "F": 3, "U": 0 },
  "waitlistCount": 0,
  "waitlistLast7Days": 0,
  "lastPhotoUploadAt": "2026-04-20T08:00:00Z"
}
```

Computed on every request (no caching — 6 geckos, trivial cost).

### 8.4 Data visualizer — safelist approach

We do **not** expose arbitrary SQL. Instead, a server-side **safelist** whitelists which tables are browsable and which columns are visible / searchable. Everything else returns 404.

```ts
// backend/src/admin/dataTables.ts  (new)
export const browsableTables = {
  geckos:           { columns: ['id', 'code', 'name', 'speciesId', 'sex', 'status', 'hatchDate', 'createdAt'],
                     searchable: ['code', 'name'], defaultSort: 'createdAt desc' },
  waitlist_entries: { columns: ['id', 'email', 'telegram', 'interestedIn', 'contactedAt', 'createdAt'],
                     searchable: ['email', 'telegram'], defaultSort: 'createdAt desc' },
  species:          { columns: ['id', 'code', 'commonName', 'scientificName'], searchable: ['commonName'] },
  genetic_dictionary: { columns: ['id', 'speciesId', 'code', 'name', 'category'], searchable: ['name', 'code'] },
  gecko_genes:      { columns: ['geckoId', 'traitId', 'zygosity'] },
  media:            { columns: ['id', 'entityType', 'entityId', 'url', 'uploadedAt'] },
  pairings:         { columns: ['id', 'sireId', 'damId', 'pairedAt', 'endedAt'] },
  clutches:         { columns: ['id', 'pairingId', 'laidAt', 'eggCount'] },
  eggs:             { columns: ['id', 'clutchId', 'status', 'laidAt', 'hatchedAt'] },
  feedings:         { columns: ['id', 'geckoId', 'feedingAt', 'food', 'amount'] },
  weights:          { columns: ['id', 'geckoId', 'measuredAt', 'grams'] },
} as const;
```

- Admin-only (middleware check).
- Pagination: `?limit=50&offset=0`. Max `limit=500`.
- Sort: `?sort=field desc|asc`, validated against `columns`.
- Filter: `?q=` runs an `ILIKE '%q%'` across `searchable`. No arbitrary column filters in v1.
- CSV export: streams the same result set with `limit=10000` cap.

**Sensitive tables NOT exposed**: `users`, `auth_sessions`, `translations` (pending), anything financial. If we need them later, add explicitly.

## 9. Page specs

### 9.1 Dashboard (`/`)

- Replaces the current placeholder `DashboardView.vue`.
- 4 stat cards (already designed): Total Geckos, Active Pairings (0 now), Eggs Incubating (0 now), Waitlist Count.
- Below: a 2-column section:
  - **Recent geckos** — last 5 geckos by `createdAt`, with thumbnail + link to detail.
  - **Recent waitlist signups** — last 5 entries, with email + date + quick "mark contacted" button.
- No charts in v1 (not enough data to be meaningful).

### 9.2 Geckos list (`/geckos`)

- Header: page title, primary "Add Gecko" button (gold).
- Filters row: species (dropdown), status (dropdown), search by code/name.
- Grid of cards (one per gecko, 3-up on desktop, 1-up on mobile):
  - Cover photo (first media entry, fallback to SVG low-poly gecko silhouette)
  - Code badge (e.g. "LP-004")
  - Name
  - Species + sex + status chips
  - Click → detail page
- Empty state: custom SVG low-poly illustration + "Add your first gecko" CTA.

### 9.3 Gecko add/edit (`/geckos/new`, `/geckos/:id`)

Single-page form with sections:

1. **Basics** — Code, Name, Species, Sex, Status, Hatch date, Acquired date, Sire (autocomplete from existing geckos), Dam (autocomplete), Notes.
2. **Genetics** — Add/remove traits. Each row: Trait (autocomplete from `genetic_dictionary` filtered by species) + Zygosity (HOM/HET/POSS_HET). Uses existing `POST /api/geckos/:id/genes`.
3. **Photos** — Drop zone + file picker (multi-select). Thumbnails with reorder (drag-and-drop if cheap, otherwise up/down arrows) and delete. First photo = cover.

Save behavior:
- `new`: create gecko first, then save genetics, then upload photos, then redirect to detail.
- `:id`: PATCH/PUT each section independently (so partial saves don't lose work). Unsaved-changes warning on route leave.

### 9.4 Waitlist (`/waitlist`)

- Table (not card grid): columns = Email, Telegram, Interested In, Signed Up, Contacted, Actions.
- Row actions: "Mark contacted" (if not yet), "Copy email", "Open in Telegram" (if handle present).
- Top-right: **"Export CSV"** button.
- Search bar: filter by email/telegram.
- Pagination: 50 per page.
- Empty state: SVG low-poly envelope + "No signups yet — add the form to the storefront to start collecting".

### 9.5 Data visualizer (`/data`) — admin only

- Left rail: list of browsable tables (from the safelist), grouped by domain (Geckos / Breeding / Media / Sales).
- Main area:
  - Table header with column names + sort indicators.
  - Rows paginated.
  - Top-right: search box + **"Export CSV"** button + row-count label.
- If a non-admin navigates here via URL, Vue router redirects to dashboard with a toast: "Admin access required."

## 10. Photo storage strategy

### 10.1 v1 — backend filesystem

- Path: `gekko_backend/uploads/geckos/<geckoId>/<timestamp>-<slug>.<ext>`.
- Hono serves `GET /uploads/*` as static (add middleware).
- Backend stores the relative URL (`/uploads/geckos/5/1714...star.jpg`) in `media.url`.
- Frontend prefixes with API base URL (`${VITE_API_BASE_URL}${url}`).
- Max file size: **5 MB** per photo. Reject larger.
- Accepted formats: `image/jpeg`, `image/png`, `image/webp`.
- Server-side resize (for thumbnails): deferred to v2. Originals served as-is in v1.

### 10.2 v2 — Cloudflare R2 (path forward, not built)

- Swap backend upload handler: write to R2 via S3 SDK, store full CDN URL in `media.url`.
- No schema change needed (`url` is already a string).
- Migration: keep filesystem URLs as-is for old records; new uploads go to R2. Optional batch-migrate script later.
- **Trigger for the swap**: first clutch photographed (= dozens of photos per week), or storage above 500 MB.

### 10.3 Why not Telegram

Rejected. Bot API URLs are not stable, subject to rate limits, not designed as a CDN, and violate ToS for hosting. See pillar in [`gekko-design`](/.claude/skills/gekko-design/SKILL.md).

## 11. Storefront → backend integration (minimum viable)

The waitlist form on the storefront is **out of scope for this spec's implementation** but must be reachable:

- Storefront adds a simple form (email + optional Telegram + optional "interested in" tag).
- Form POSTs to `${API_BASE}/api/public/waitlist`.
- API base URL configured via `VITE_API_BASE_URL` in storefront `.env`.
- CORS on backend allows the storefront's origin (Vercel preview + prod).
- Storefront form UI/copy is covered in a separate spec (audience-building milestone).

## 12. Auth & authorization

- Unchanged login flow. JWT issued by backend; stored in Pinia + `localStorage`; attached via axios interceptor.
- Vue router guards:
  - `/login` — no auth required.
  - Everything else — `requiresAuth: true`.
  - `/data/*` — `requiresAdmin: true` (in addition).
- Backend middleware:
  - `authMiddleware` — existing, enforces valid JWT.
  - `adminOnlyMiddleware` — **new**, checks `user.role === 'ADMIN'`.
- Logout clears token, redirects to `/login`.

## 13. Hosting

- **Admin**: Vercel or Cloudflare Pages (static Vue SPA build). Same domain pattern as storefront. Vercel likely — you already know it.
- **Backend**: **Railway** recommended. ~$5/mo includes Postgres + persistent filesystem (needed for v1 photo storage). Fly.io free tier is alternative but uploads on free-tier volumes can be wonky.
- **Postgres migration**: backup local Docker DB (`pg_dump`), restore on Railway (`psql < dump.sql`), update backend env.
- **Domains** (proposal, decide later): `admin.zeneticgekkos.com`, `api.zeneticgekkos.com`, `zeneticgekkos.com` (storefront).

## 14. Testing strategy

Pragmatic — this is a single-operator tool, not a SaaS:

- **Backend**: unit tests for the safelist / CSV export endpoints (high risk of data leakage). Happy-path integration tests for waitlist + gecko PUT + photo upload.
- **Frontend**: component tests for BaseButton/BaseCard already covered via usage. Add a Cypress or Playwright smoke test: login → add gecko → upload photo → see it in list. One critical-path E2E.
- No 80% coverage target for v1. Cover the risky parts (auth, uploads, data viz safelist), not every getter.

## 15. Deployment plan (rough sequence, detailed in impl plan)

1. Schema change: create `waitlist_entries` table via migration.
2. Backend: waitlist endpoints + CORS + rate limit.
3. Backend: gecko PUT + multipart photo upload.
4. Backend: dashboard stats + data visualizer safelist endpoints.
5. Admin UI: dashboard upgrade.
6. Admin UI: geckos list + detail + photo manager.
7. Admin UI: waitlist viewer.
8. Admin UI: data visualizer.
9. Storefront: waitlist form (separate spec).
10. Move backend to Railway; move admin to Vercel.

Phases 1–4 can be done in parallel with 5–8 (different repos). Storefront (9) is a follow-up spec.

## 16. Open questions for user

1. **Backend hosting** — Railway ($5/mo) acceptable, or prefer Fly.io free tier / stay local for now? *Impacts when we can point TikTok traffic at a real waitlist endpoint.*
2. **Fonts** — DM Serif Display + Inter as proposed in [`gekko-design`](/.claude/skills/gekko-design/SKILL.md), or different pairing?
3. **Telegram handle** — what's the actual Telegram / Messenger handle to link to from the storefront?
4. **Gecko codes** — existing format `LP-004`, `AF-001`? Auto-generate or manual entry?
5. **Who uploads to Vercel for storefront/admin** — you, or should we set up CI from GitHub? (Vercel GitHub integration is the standard path.)
6. **Source maps on admin build** — expose or hide? (Default: hide for prod.)

## 17. What makes v1 "done"

1. Operator can log in, see dashboard with real counts reflecting their 6 geckos.
2. Operator can add a new gecko with photos and genetics via the UI.
3. Operator can view the waitlist and export it to CSV.
4. Operator can browse any of the 11 safelisted tables read-only and export CSV.
5. The visual output matches the [`gekko-design`](/.claude/skills/gekko-design/SKILL.md) rules (no raw grays, no pure black/white, no shadcn drift, proper type scale).
6. Backend + admin deployed to a live URL the operator can reach from anywhere.

## 18. Out of v1 (explicit deferrals)

| Feature | When to revisit |
|---|---|
| Pairings / clutches / eggs UI | Month the first pairing is set up (3-6 months out) |
| Husbandry quick-log (feed/weight) | After v1 ships, if daily logging becomes a habit |
| Supplies / POs / suppliers / shipments | Only if supplies becomes a real revenue line |
| Media library (standalone) | Only if per-gecko photos prove insufficient |
| Staff role / multiple users | When first hire is made |
| R2 migration for photos | First clutch, or ~500 MB storage |
| Admin mobile layout | If operator actually logs in from phone regularly |
| i18n for admin (backend has translation data) | Never, probably — admin is English-only |
| Storefront waitlist form UI | Separate spec (audience-building milestone) |
