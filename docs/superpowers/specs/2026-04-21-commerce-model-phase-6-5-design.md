# Phase 6.5 — Commerce Model (Admin MVP)

> **Goal:** Replace `geckos.list_price_usd` with a proper commerce layer (`listings` + `listing_geckos` + `listing_components`) so gecko rows stay pure biology and supplies/packages become first-class sellable items. Admin-only for this pass; storefront keeps working via a thin public-API compatibility shim that joins against listings.

## Scope

**In scope (tonight, admin MVP)**

- Three new tables: `listings`, `listing_geckos`, `listing_components` + two enums (`listing_type`, `listing_status`).
- Data migration: every gecko with a `list_price_usd IS NOT NULL` becomes a `LISTED` GECKO listing + a `listing_geckos` row. The column is then dropped from `geckos`.
- sqlc queries for listing CRUD + junction management.
- Backend: `MountListings(r, pool, signer)` with `GET/POST /api/listings`, `GET/PATCH/DELETE /api/listings/{id}`. Admin-auth only. Validation: GECKO needs ≥1 gecko; PACKAGE needs ≥1 component; SUPPLY requires an SKU; SKU unique across listings.
- Backend: public API compat. `/api/public/geckos` and `/api/public/geckos/{code}` LEFT JOIN against listings so the existing storefront keeps showing `list_price_usd`. List endpoint filters to geckos that have a `status='LISTED'`, `type='GECKO'` listing (since AVAILABLE now means "commerce-active," not just "biology-available").
- Admin UI: new `Listings` sidebar item + `ListingsView` (grid + type/status filter chips + "Create listing" button), `ListingCard` (tile showing title, type + status badges, price, counts), `ListingFormSheet` (slide-in drawer like `GeckoFormSheet` — type-specific form sections, component picker for packages, gecko picker for gecko listings).
- Admin gecko edit drawer: price field removed (now listing-owned).
- Admin gecko detail: new "Listings" section showing any listings referencing this gecko + a "Create listing for this gecko" button that pre-fills the sheet with `type='GECKO'` + the current gecko attached.

**Out of scope (next session)**

- Full storefront rewire (`/api/public/listings`, `PublicListingCard`, browse-by-type). Storefront continues to call `/api/public/geckos` which now joins against listings.
- SUPPLY/PACKAGE detail pages on the storefront. Until the full rewire, the storefront only shows gecko listings; supplies and packages live admin-side.
- `stock_on_hand` column on listings (Tier 2 — when actual physical stock is held).
- Photo upload for SUPPLY/PACKAGE cover images. Tonight uses a manual `cover_photo_url` text input; upload can come later.
- Strict status transition rules (e.g. "must be DRAFT before LISTED"). Tonight any transition is allowed.
- Multi-level package recursion (package inside package). Flat bundling only; DB enforces no self-reference but not deeper cycles.
- SEO / i18n / analytics on storefront.
- Listing photo galleries beyond a single cover.
- Seed supplies — Zen adds them manually via admin UI to test.

## Architecture

```
    biology row                          commerce row
    ─────────────                        ────────────
      geckos                                listings
      (no price)                            (price + status + type + sku)
         ▲                                  ▲
         │                                  │
         │   listing_geckos                 │   listing_components
         └───────junction──────────┐        └──────junction──────────┐
                                   ▼                                 ▼
                                listings                           listings
                                (GECKO)                            (component)
```

One commerce object per sellable unit. Geckos and listings meet through `listing_geckos`. Packages reference component listings through `listing_components`. A GECKO listing may attach multiple gecko rows (pair / trio bundles); a PACKAGE may include GECKO or SUPPLY listings as components.

## Backend

### Migrations

One migration file: `backend/migrations/20260421XXXXXX_commerce_model.sql` (pick a sequence number higher than the last applied migration).

**Up:**
```sql
CREATE TYPE listing_type   AS ENUM ('GECKO', 'PACKAGE', 'SUPPLY');
CREATE TYPE listing_status AS ENUM ('DRAFT', 'LISTED', 'RESERVED', 'SOLD', 'ARCHIVED');

CREATE TABLE listings (
  id               SERIAL PRIMARY KEY,
  sku              VARCHAR(64) UNIQUE,
  type             listing_type NOT NULL,
  title            VARCHAR(200) NOT NULL,
  description      TEXT,
  price_usd        NUMERIC(10,2) NOT NULL,
  deposit_usd      NUMERIC(10,2),
  status           listing_status NOT NULL DEFAULT 'DRAFT',
  cover_photo_url  VARCHAR(500),
  listed_at        TIMESTAMP,
  sold_at          TIMESTAMP,
  archived_at      TIMESTAMP,
  created_at       TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at       TIMESTAMP DEFAULT NOW() NOT NULL
);
CREATE INDEX listings_type_status_idx ON listings (type, status);

CREATE TABLE listing_geckos (
  listing_id  INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  gecko_id    INTEGER NOT NULL REFERENCES geckos(id)   ON DELETE RESTRICT,
  created_at  TIMESTAMP DEFAULT NOW() NOT NULL,
  PRIMARY KEY (listing_id, gecko_id)
);
CREATE INDEX listing_geckos_gecko_idx ON listing_geckos (gecko_id);

CREATE TABLE listing_components (
  listing_id             INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  component_listing_id   INTEGER NOT NULL REFERENCES listings(id) ON DELETE RESTRICT,
  quantity               INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
  created_at             TIMESTAMP DEFAULT NOW() NOT NULL,
  PRIMARY KEY (listing_id, component_listing_id),
  CHECK (listing_id <> component_listing_id)
);
CREATE INDEX listing_components_component_idx ON listing_components (component_listing_id);

-- Data migration: existing priced geckos become LISTED listings.
-- Do this in a single pass via a CTE so title + junction stay linked.
WITH inserted AS (
  INSERT INTO listings (type, title, price_usd, status, listed_at)
  SELECT 'GECKO',
         COALESCE(g.name, g.code),
         g.list_price_usd,
         'LISTED',
         NOW()
  FROM geckos g
  WHERE g.list_price_usd IS NOT NULL
  RETURNING id, title
)
INSERT INTO listing_geckos (listing_id, gecko_id)
SELECT i.id, g.id
FROM inserted i
JOIN geckos g ON COALESCE(g.name, g.code) = i.title;
-- Note: relies on gecko name/code being unique, which it effectively is —
-- `geckos.code` has a UNIQUE constraint and distinct names among priced
-- geckos. If two priced geckos shared a name AND neither had a code, this
-- would attach both to one listing; the CHECK on listing_geckos PK would
-- reject a subsequent duplicate insert. Not an issue today (5 geckos,
-- all with distinct codes).

ALTER TABLE geckos DROP COLUMN list_price_usd;
```

**Down:**
```sql
ALTER TABLE geckos ADD COLUMN list_price_usd NUMERIC(10,2);

-- Best-effort restore: copy price back to gecko if the gecko is attached
-- to exactly one GECKO listing.
UPDATE geckos g
SET list_price_usd = l.price_usd
FROM listings l
JOIN listing_geckos lg ON lg.listing_id = l.id
WHERE l.type = 'GECKO' AND lg.gecko_id = g.id;

DROP TABLE listing_components;
DROP TABLE listing_geckos;
DROP TABLE listings;
DROP TYPE  listing_status;
DROP TYPE  listing_type;
```

### sqlc queries (backend/internal/queries/listings.sql)

- `ListListings :many` — returns listing rows ordered by status then created_at DESC, with an aggregated `gecko_count` and `component_count` via LEFT JOIN + GROUP BY.
- `ListListingsByType :many` — same, filtered by type param.
- `GetListing :one` — single row by id.
- `GetListingBySku :one` — used for SKU validation.
- `CreateListing :one` — INSERT with RETURNING.
- `UpdateListing :one` — UPDATE with RETURNING (ID + all fields).
- `DeleteListing :exec` — DELETE by id.
- `AttachGeckoToListing :exec` — INSERT INTO listing_geckos.
- `DetachGeckosForListing :exec` — DELETE FROM listing_geckos WHERE listing_id = $1.
- `ListGeckosForListing :many` — SELECT gecko_id + gecko.code + gecko.name + species.code joined.
- `SetListingComponent :exec` — INSERT INTO listing_components … ON CONFLICT (listing_id, component_listing_id) DO UPDATE SET quantity.
- `DeleteComponentsForListing :exec` — DELETE FROM listing_components WHERE listing_id = $1.
- `ListComponentsForListing :many` — SELECT component + title + type + price joined.

### Public API compat (modifications to existing handlers)

`backend/internal/http/public.go`:
- `listAvailable` — replace the existing query with one that joins against `listings + listing_geckos` (type=GECKO, status=LISTED) and surfaces `price_usd` from the listing. Only return geckos with a LISTED listing.
- `getByCode` — same, LEFT JOIN for price; 404 if no LISTED listing exists for the gecko.

### New HTTP handlers (backend/internal/http/listings.go)

`MountListings(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner)`:
- `GET /api/listings` — returns all listings (admin view — includes DRAFT, SOLD, ARCHIVED). Optional query params: `?type=GECKO|PACKAGE|SUPPLY`, `?status=...`.
- `GET /api/listings/{id}` — full detail with `geckos: [...]` (for GECKO) or `components: [...]` (for PACKAGE) arrays.
- `POST /api/listings` — body: `{type, title, description, price_usd, deposit_usd, status, cover_photo_url, sku, geckos: [{gecko_id}], components: [{component_listing_id, quantity}]}`. Validation:
  - `type` required, one of enum.
  - `title` required, ≤200 chars.
  - `price_usd` required, ≥ 0.
  - `GECKO`: `geckos` array must have ≥1 entry; `sku` may be null.
  - `PACKAGE`: `components` array must have ≥1 entry; `sku` may be null.
  - `SUPPLY`: `sku` required, non-empty, unique across listings.
  - `cover_photo_url` optional for all types.
- `PATCH /api/listings/{id}` — partial updates; `type` is immutable after create; junctions can be swapped in full (delete-all + re-insert pattern for simplicity, like admin's existing trait-swap on gecko update).
- `DELETE /api/listings/{id}` — deletes the listing + cascades `listing_geckos` + `listing_components` rows (via ON DELETE CASCADE on the junctions).

**Error handling** mirrors existing patterns:
- 400 on validation failure with `{"error": "..."}`.
- 404 on unknown id.
- 409 on SKU conflict (unique violation via `pgerrcode.UniqueViolation`).
- 500 on other errors.

### Admin gecko DTO change

`backend/internal/http/geckos.go`:
- Remove `ListPriceUsd` field from `geckoDTO`.
- Remove `list_price_usd` from `CreateGeckoReq` and `UpdateGeckoReq`.
- Corresponding handler code no longer parses or persists the price.

## Admin frontend

### New types + composables

`apps/admin/src/types/listing.ts`:

```ts
export type ListingType = 'GECKO' | 'PACKAGE' | 'SUPPLY';
export type ListingStatus = 'DRAFT' | 'LISTED' | 'RESERVED' | 'SOLD' | 'ARCHIVED';

export interface ListingGeckoRef {
  gecko_id: number;
  code: string;
  name: string | null;
  species_code: string;
}
export interface ListingComponentRef {
  component_listing_id: number;
  title: string;
  type: ListingType;
  price_usd: string;
  quantity: number;
}
export interface Listing {
  id: number;
  sku: string | null;
  type: ListingType;
  title: string;
  description: string | null;
  price_usd: string;
  deposit_usd: string | null;
  status: ListingStatus;
  cover_photo_url: string | null;
  listed_at: string | null;
  sold_at: string | null;
  archived_at: string | null;
  created_at: string;
  updated_at: string;
  gecko_count: number;
  component_count: number;
  // Present on detail fetch, undefined on list fetch.
  geckos?: ListingGeckoRef[];
  components?: ListingComponentRef[];
}
```

`apps/admin/src/composables/useListings.ts`:
- `useListings()` — lists all with counts.
- `useListing(id)` — single with joined arrays.
- `useCreateListing()` — mutation.
- `useUpdateListing()` — mutation.
- `useDeleteListing()` — mutation.

Each mutation invalidates the `['listings']` cache + the specific `['listings', id]` on success.

### `ListingsView.vue`

Layout mirrors `GeckosView`:
- `PageHeader` with "Listings" title + "Create listing" button.
- Filter bar: type chips (ALL / GECKO / PACKAGE / SUPPLY) + status chips (ALL / DRAFT / LISTED / RESERVED / SOLD / ARCHIVED).
- Search: free text over title + SKU.
- Grid: `ListingCard` tiles, 3 per row at lg.
- Loading skeleton, error banner, empty states by filter.

### `ListingCard.vue`

- Thumbnail: `cover_photo_url` if set, else gecko cover for GECKO type, else a type-specific placeholder icon over gradient.
- Top-left: type badge (GECKO soft-gold / PACKAGE soft-cream / SUPPLY muted).
- Top-right: status badge (LISTED = success, DRAFT = muted, RESERVED = warn, SOLD = outline, ARCHIVED = outline with strikethrough).
- Body: title (serif large), SKU (mono small) if present, price, quantity summary ("2 geckos" / "3 items" depending on type).
- Click → opens edit drawer.

### `ListingFormSheet.vue`

Slide-in drawer pattern identical to `GeckoFormSheet`. Sections:

1. **Common fields** — title, description (textarea), price, deposit, status, cover_photo_url (plain input; no upload yet).
2. **Type-specific** — `type` radio group at top, only visible when creating; **locked** in edit mode (changing type breaks junction semantics).
    - **SUPPLY:** SKU input (required).
    - **GECKO:** multi-select gecko list. Reuse of `GeckoPicker` with a new "multi" mode, OR a simpler list: "Add gecko" button opens a popover picker, selected geckos render as removable chips. Simpler pattern picked: chip list + popover.
    - **PACKAGE:** component list — each row has a listing picker (filtered to exclude current listing + exclude other PACKAGEs to avoid nesting) + quantity input. Show a running "components total: $X" under the list for reference (informational; price is still explicit).
3. **Footer** — Cancel + Save.

On save:
- Assembles the full payload (type, common fields, geckos/components arrays as relevant).
- Calls `useCreateListing` or `useUpdateListing`.
- Toast success with the title or SKU.
- Closes drawer, invalidates cache so grid refreshes.

### Admin shell + router

- `apps/admin/src/layouts/AppShell.vue` — add nav item "Listings" with `Tag` lucide icon, between "Geckos" and "Waitlist".
- `apps/admin/src/router/index.ts` — `{ path: 'listings', name: 'listings', component: ListingsView }`.

### Admin gecko touch-ups

- `apps/admin/src/components/GeckoFormSheet.vue` — remove price input field; remove `list_price_usd` from the submit payload; `Gecko` type loses the field.
- `apps/admin/src/components/GeckoCard.vue` — price display falls back to nothing (the list API still returns the field for storefront compat, but the admin list view no longer receives it — or it's deprecated with undefined).
- `apps/admin/src/views/GeckoDetailView.vue` — add a new "Listings" section below the traits block showing any listings this gecko is attached to. "Create listing" button opens the listing drawer pre-populated with `type='GECKO'` + the current gecko attached.
- `apps/admin/src/types/gecko.ts` — drop `list_price_usd` from `Gecko` interface.

### Public gecko shape

Because the public endpoint now joins against listings, the admin's `Gecko` type needs to reflect that `list_price_usd` is gone from the admin DTO but the *public* DTO (separate) still exposes it. Keep the two types separate in `apps/admin/src/types/` vs `apps/storefront/src/types/` — admin's `Gecko` loses the field; storefront's `PublicGecko` keeps it (sourced from listing).

## Data flow

```
  Admin (create GECKO listing)
         │
         ▼
  ListingFormSheet ──▶ useCreateListing ──▶ POST /api/listings
                                                  │
                                              validation
                                                  │
                                     tx: INSERT listings + listing_geckos
                                                  │
                                              return detail
         ◀── query cache invalidated ────────────┘

  Storefront (list available)
         │
         ▼
  /api/public/geckos  ──▶  public.go handler
                                │
                                ▼
                          LEFT JOIN listings l ON lg.gecko_id = g.id
                                          AND l.type='GECKO'
                                          AND l.status='LISTED'
                                WHERE l.id IS NOT NULL (filter)
                                │
                                ▼
                          same sanitized DTO as before,
                          but list_price_usd now from l.price_usd
```

## Error handling

- **Validation errors** → 400 with the failing rule in the message.
- **Unknown id on PATCH/DELETE/GET** → 404.
- **SKU collision** → 409 with a clear message so the drawer can show "This SKU is already used."
- **Component cycle (direct self-ref)** → handled by DB CHECK; 500 surfaces; practically unreachable because the frontend excludes the current listing from the component picker.
- **Admin edit that would leave GECKO listing without geckos** → 400 at save time; the sheet's chip list should visually warn if empty.
- **Storefront** still handles 404 on `/api/public/geckos/:code` gracefully (existing behavior unchanged).

## Testing

**Backend (go test)**
- `TestCreateListing_gecko_requiresAtLeastOne` — POST GECKO with empty `geckos` → 400.
- `TestCreateListing_package_requiresAtLeastOneComponent` — POST PACKAGE with empty `components` → 400.
- `TestCreateListing_supply_requiresSku` — POST SUPPLY without SKU → 400.
- `TestCreateListing_sku_uniqueness` — two SUPPLY listings, same SKU → second returns 409.
- `TestCreateListing_geckoHappy` — creates GECKO listing, asserts junction row + listing row exist.
- `TestUpdateListing_replacesTraits` — hypothetical rename; for listings: `TestUpdateListing_swapsGeckos` — PATCH with different gecko set replaces junction rows.
- `TestPublicListAvailable_onlyListedGeckoListings` — a gecko without a LISTED listing is NOT returned by `/api/public/geckos`.
- `TestPublicGetGecko_byCode_respectsListing` — a gecko with an ARCHIVED listing returns 404 on the public detail.

**Admin (vitest)**
- No new coverage; existing 7 stay green.

## Rollout

Commits grouped by concern:

1. Migration + sqlc queries + generated Go.
2. Listings HTTP handlers (`MountListings`) + route mount.
3. Public API compat join (modifications to `public.go`).
4. Admin gecko DTO price removal + handler cleanup + admin frontend `Gecko` type change.
5. Admin `useListings` composable + types.
6. Admin `ListingsView` + `ListingCard` + router + nav.
7. Admin `ListingFormSheet` with all three type branches.
8. Admin gecko detail "Listings" section + pre-filled create.
9. Backend tests.
10. Final smoke + push + Zen verifies in admin.

Pushed together so the public storefront never runs against a half-migrated backend (price column drops the same tx as the listings table lands).
