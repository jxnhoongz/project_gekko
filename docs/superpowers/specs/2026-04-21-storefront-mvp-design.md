# Storefront MVP

> **Goal:** Stand up `apps/storefront` as a second bun-workspace Vue app — a public-facing site that lets interested buyers see what's available, read about the breedery, join the waitlist, and deep-link to individual geckos. No customer accounts, no checkout, no i18n yet.

## Scope

**In scope**

- New workspace `apps/storefront` (Vue 3.5 / Vite / TypeScript / Tailwind 4 / shadcn-vue / TanStack Query), Vite dev server on port 5174.
- Brand parity with admin via shared gekko-design tokens (cream + gold + low-poly). `BrandLogo`, `LowPolyGecko`, `LowPolyAccent` SVG components are copied from admin (not yet extracted to a shared package — defer).
- Backend: three new public endpoints (no JWT):
  - `GET /api/public/geckos` — list geckos where `status = 'AVAILABLE'`, sanitized DTO.
  - `GET /api/public/geckos/{code}` — single gecko by code, 404 if missing or not AVAILABLE.
  - `POST /api/public/waitlist` — create waitlist entry.
- Four storefront routes:
  - `/` — landing page (hero + available teaser + waitlist CTA + about section + footer).
  - `/geckos` — full available grid with a single public-safe card type.
  - `/geckos/:code` — gecko detail page (photos, morph, species, sex, age, price, waitlist CTA).
  - `/waitlist` — dedicated signup form with success state.
- Vite dev proxy (`/api`, `/uploads`) to the backend on `:8420`, mirroring the admin setup.
- Light in-memory per-IP rate limit on `POST /api/public/waitlist` (5 per hour) — enough to deter form spam without a proper rate-limit infrastructure.

**Out of scope (deferred)**

- SEO metadata / sitemap / og tags / per-gecko share previews.
- i18n (EN / KM / ZH via legacy `translations` table).
- Analytics hooks.
- Cart, checkout, Stripe, customer accounts.
- Password reset, login, session cookies.
- Shared `packages/ui-brand/` — revisit when a third app or serious divergence appears.

## Architecture

```
                                                 browser
                                                    │
                             ┌──────────────────────┼──────────────────────┐
                             ▼                      ▼                      ▼
                      apps/admin/           apps/storefront/          direct backend
                      (admin UI, auth)      (public, no auth)         (dev tools only)
                             │                      │
                             │ /api/*               │ /api/public/*
                             │ /uploads/*           │ /uploads/*
                             ▼                      ▼
                    ┌───────────────────────────────────────────┐
                    │   backend (chi router, :8420)             │
                    │   ── admin: RequireAuth middleware        │
                    │   ── public: no auth, rate-limited        │
                    └───────────────────────────────────────────┘
                                       │
                                 ┌─────┴─────┐
                                 ▼           ▼
                            Postgres     uploads/
```

Two Vue apps share the same backend but hit different endpoint families. Admin endpoints live under `/api/*` (behind `RequireAuth`); the new public endpoints live under `/api/public/*` with no auth middleware. Public DTOs are deliberately a strict subset of admin DTOs — no `notes`, `sire_id`, `dam_id`, `status`, `acquired_date`, or `updated_at`.

## Backend

### New endpoints

All mounted via a new `MountPublic(r, pool)` helper in `backend/internal/http/public.go`. No JWT signer dependency — no authenticated user context.

#### `GET /api/public/geckos`

Query all AVAILABLE geckos, with cover photo pre-loaded.

**Response:**
```json
{
  "geckos": [
    {
      "code": "ZGLP-2026-004",
      "name": "Suri",
      "species_code": "CR",
      "species_name": "Crested Gecko",
      "morph": "Dalmatian",
      "sex": "F",
      "hatch_date": "2024-03-08",
      "list_price_usd": "180",
      "cover_photo_url": "/uploads/geckos/4/…png"
    }
  ],
  "total": 1
}
```

Fields deliberately omitted: internal `id`, `notes`, `sire_id`, `dam_id`, `status` (always AVAILABLE), `acquired_date`, `created_at`, `updated_at`, raw `traits` array (we compose a single `morph` string via the existing `morphFromTraits` logic, reimplemented server-side).

#### `GET /api/public/geckos/{code}`

Single gecko by code. Returns 404 if not found or not AVAILABLE. Includes all photos in `photos` array (same public shape as list DTO, plus `photos: [{url, caption, display_order}]`).

#### `POST /api/public/waitlist`

**Request:**
```json
{
  "email": "required",
  "telegram": "optional",
  "phone": "optional",
  "interested_in": "optional",
  "notes": "optional"
}
```

Validation:
- `email` required, basic regex `.+@.+\..+` (not RFC-compliant, but enough).
- String length caps: email ≤ 255, telegram ≤ 100, phone ≤ 32, interested_in ≤ 100, notes ≤ 2000.
- On duplicate email (unique index on `LOWER(email)`), return 200 with `{ "deduplicated": true }` — don't leak whether someone already signed up; spam-friendly behavior.

**Response:** 201 with `{ "id": 42 }` on first insert, 200 with `{ "deduplicated": true }` on duplicate.

**Rate limiting:** simple in-memory map `ip -> [timestamps]`, prune older than 1 hour, reject with 429 when count > 5 in the last hour. Not cluster-safe, but fine for single-box MVP.

Add this comment verbatim at the top of the rate-limit middleware so the scaling path is obvious when we hit it:

```go
// TODO: swap to Postgres- or Redis-backed limiter when we run more than
// one backend instance, or when a second rate-limited endpoint appears.
```

### New queries (sqlc)

In `backend/internal/queries/public.sql`:

- `ListAvailableGeckos :many` — joins `geckos` + `species` + first-media cover per gecko, filtered to `status = 'AVAILABLE'`, ordered by `created_at DESC`.
- `GetAvailableGeckoByCode :one` — single gecko by code + species, 404 if not available.
- `ListGenesForGeckoByCode :many` — traits for a gecko looked up by code (for `morph` composition).
- `ListMediaForGeckoByCode :many` — photos ordered by display_order.

(`CreateWaitlistEntry` already exists from Phase 2.)

### Morph composition

Reuse the JavaScript logic from `apps/admin/src/types/gecko.ts:morphFromTraits` in Go:

```go
func composeMorph(traits []trait) string {
	hom := []string{}
	het := []string{}
	poss := []string{}
	for _, t := range traits {
		switch t.Zygosity {
		case db.ZygosityHOM: hom = append(hom, t.TraitName)
		case db.ZygosityHET: het = append(het, t.TraitName)
		case db.ZygosityPOSSHET: poss = append(poss, t.TraitName)
		}
	}
	parts := []string{}
	if len(hom) > 0 { parts = append(parts, strings.Join(hom, " ")) }
	if len(het) > 0 {
		prefixed := make([]string, len(het))
		for i, n := range het { prefixed[i] = "het " + n }
		parts = append(parts, strings.Join(prefixed, " "))
	}
	if len(poss) > 0 {
		prefixed := make([]string, len(poss))
		for i, n := range poss { prefixed[i] = "poss. het " + n }
		parts = append(parts, strings.Join(prefixed, " "))
	}
	if len(parts) == 0 { return "Normal" }
	return strings.Join(parts, ", ")
}
```

## Storefront app

### Scaffolding

```
apps/storefront/
├── index.html
├── package.json               # private workspace, no "version"
├── tsconfig.json
├── tsconfig.app.json
├── tsconfig.node.json
├── vite.config.ts             # port 5174, proxy /api + /uploads + /health
├── components.json            # shadcn-vue config
├── public/
│   └── logo/                  # copied from admin
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── style.css              # copied from admin (brand tokens)
│   ├── router/index.ts
│   ├── lib/api.ts             # axios instance (no token interceptor)
│   ├── lib/format.ts          # copied from admin
│   ├── types/gecko.ts         # public-safe subset
│   ├── composables/
│   │   ├── usePublicGeckos.ts
│   │   └── useWaitlistSignup.ts
│   ├── components/
│   │   ├── BrandLogo.vue
│   │   ├── art/
│   │   │   ├── LowPolyGecko.vue
│   │   │   └── LowPolyAccent.vue
│   │   ├── SiteHeader.vue
│   │   ├── SiteFooter.vue
│   │   ├── PublicGeckoCard.vue
│   │   └── ui/                # shadcn primitives (button, card, input, label, sonner)
│   └── views/
│       ├── HomeView.vue
│       ├── GeckosView.vue
│       ├── GeckoDetailView.vue
│       └── WaitlistView.vue
├── .env.local                 # optional; empty by default (uses vite proxy)
└── vitest.config.ts           # minimal
```

### Routes

```ts
{
  routes: [
    { path: '/',              name: 'home',          component: HomeView },
    { path: '/geckos',        name: 'geckos',        component: GeckosView },
    { path: '/geckos/:code',  name: 'gecko-detail',  component: GeckoDetailView, props: true },
    { path: '/waitlist',      name: 'waitlist',      component: WaitlistView },
    { path: '/:pathMatch(.*)*', name: 'not-found',   component: HomeView }, // soft 404 → home
  ],
}
```

### Shared primitives

`SiteHeader.vue`:
- Sticky top bar with `BrandLogo` on the left (routes to `/`) and nav (`Geckos`, `Waitlist`) on the right.
- On mobile: hamburger collapses nav.

`SiteFooter.vue`:
- `© 2026 Zenetic Gekkos · Phnom Penh` + optional Telegram / Instagram links (Zen can fill in later; leave empty for MVP).

`PublicGeckoCard.vue`:
- Simplified GeckoCard — cover photo (or LowPolyGecko fallback), code (small), name (serif large), morph (single line), species + sex, price badge. No status badge (all cards are AVAILABLE by definition).
- Links to `/geckos/:code`.

### Pages

#### `HomeView.vue`

Landing, roughly 4 vertical sections:
1. **Hero** — full-bleed cream/gold gradient panel, `LowPolyGecko` (floating animation), display-font headline ("Small-batch gecko breedery in Phnom Penh"), one-sentence pitch, primary CTA "Join the waitlist" (→ `/waitlist`) + secondary CTA "Browse available" (→ `/geckos`).
2. **Available teaser** — grid of up to 3 `PublicGeckoCard`s (latest AVAILABLE). Empty state: "Nothing available right now — we'll announce new hatchlings to the waitlist first." Link to full list.
3. **About Zenetic** — short paragraph + low-poly accent. Static copy; no CMS.
4. **CTA banner** — "Want first dibs on our next clutch? Join the waitlist." Button links to `/waitlist`.

#### `GeckosView.vue`

Full grid of available geckos. Above the grid: short header "Currently available". Empty state reused from Home. No filters for MVP (collection is small).

#### `GeckoDetailView.vue`

- Hero photo (cover) + gallery strip of remaining photos (click to swap cover image inline — no lightbox for MVP, just a simple image swap).
- Facts block: code, name, species, morph, sex, age, price.
- "Interested? Join the waitlist" CTA at the bottom (passes `interested_in=<code>` as pre-filled form state).
- Breadcrumb "Geckos / ZGLP-2026-004".

#### `WaitlistView.vue`

Form fields per spec. Client-side validation: email format, required. On success: swap form with a soft "Thanks — we'll be in touch." state. On error (network / rate limit): toast with message.

Pre-fill from query param: if `?interested_in=ZGLP-2026-004`, populate that field.

## Data flow

```
Browser ─── GET /api/public/geckos ──────▶ backend/internal/http/public.go
                                               │
                                               ▼
                             sqlc: ListAvailableGeckos, ListGenesForGecko…
                                               │
                                               ▼
                                          Postgres
                                               │
                                   (map → publicGeckoDTO)
                                               ▲
                                               │
                                     morph composed server-side

Browser ─── POST /api/public/waitlist ────▶ rate-limit middleware
                                               │
                                               ▼
                                          validate + CreateWaitlistEntry
                                               │
                                    unique violation? → { deduplicated: true }
                                    otherwise → { id }
```

## Error handling

- **Rate-limit exceeded:** 429 with `Retry-After` header.
- **Duplicate email:** catch `23505` unique violation on `waitlist_entries.email`; return 200 deduplicated.
- **Unknown gecko code:** 404.
- **Status-not-AVAILABLE gecko by code:** 404 (don't leak existence of private geckos).
- **Malformed body:** 400.
- **DB error:** 500 with a generic "something went wrong" — no internal detail leaked to the public.
- **Storefront fetch failures:** TanStack Query's `isError` path → user-facing "Couldn't load — try again" panel with a retry button.

## Testing

**Backend (go test)**
- `TestPublicListGeckos_onlyAvailable` — seed 1 AVAILABLE + 1 HOLD + 1 BREEDING, assert list returns only the AVAILABLE entry.
- `TestPublicGetGecko_byCode_available` — fetch an AVAILABLE gecko by code, assert sanitized DTO shape (no `notes`, no `sire_id`).
- `TestPublicGetGecko_byCode_notAvailable` — fetch a HOLD gecko, expect 404.
- `TestPublicWaitlist_create` — POST valid body, assert 201 + entry in DB.
- `TestPublicWaitlist_duplicate` — POST same email twice, second returns 200 with `deduplicated: true`.
- `TestPublicWaitlist_rateLimit` — 6 POSTs from same IP, 6th gets 429.

**Storefront (vitest)**
- `App.smoke.spec.ts` — mount the app with a mocked router and assert the root renders without throwing.
- Per-view smoke tests are not in scope for MVP.

## Rollout

Commits grouped:

1. Backend public endpoints + queries + tests.
2. Storefront scaffold (package.json, vite.config, tsconfig, router, shared primitives copied from admin).
3. Home + Gecko list + Gecko detail + Waitlist views.
4. Final smoke: run dev server, hit the 4 routes, push.

## Open questions for later

- When the collection grows, add filters (species, sex, price range) to `/geckos`.
- Photo lightbox — defer until a gecko with many photos actually needs it.
- Hosting: storefront is static-ish (SPA) — can deploy to Cloudflare Pages / Netlify later, or co-locate with backend.
- Domain: `zeneticgekkos.com` or similar. Out of scope for MVP.
