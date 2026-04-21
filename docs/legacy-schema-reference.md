# Legacy Data Schema — Reference (informational only)

> **What this is:** A reference copy of the schema from the deprecated Node/Hono/Drizzle backend (`jxnhoongz/gekko_backend`), ported here for the Go rewrite to reference. **This is not a spec.** The new backend does not have to mirror this 1:1. A good chunk of the old schema was aspirational / over-engineered for a pre-breeding business.
>
> **How to use it:** When planning a new phase (geckos, breeding, products, etc.), read the relevant domain section, understand the shape, and pull over only what the phase needs. Every section ends with a "what to actually port" verdict.
>
> **Date**: Captured 2026-04-21. Source: `gekko_legacy/gekko_backend/src/db/schema.ts`.

---

## 1. At a glance

- **28 tables** grouped into **9 domains**.
- **10 enums** (status, sex, zygosity, etc.).
- **~50 foreign keys**, including self-referencing (gecko parentage) and polymorphic soft references (translations).
- Built with **Drizzle ORM** on Postgres 15. Timestamps default to `NOW()`, all primary keys are `serial` (auto-increment integers).
- Seed data existed for: 2 species, 30+ genetic traits, 7 geckos, 1 pairing, 1 clutch, 16 inventory items, 15+ products, 52 translations (16 traits in zh-CN, 10 in km-KH).

### Domain map

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────────┐
│    users    │     │   species    │◄────│  genetic_dictionary │
└─────────────┘     └──────┬───────┘     └──────────┬──────────┘
                           │                        │
                           ▼                        │
                    ┌──────────────┐                │
                    │   geckos ◄───┼────┐           │
                    │  (self-ref   │    │ sire/dam  │
                    │  parentage)  │    │           │
                    └──────┬───────┘    └───────────┼─────┐
                           │                        │     │
              ┌────────────┼────────────┐           ▼     ▼
              ▼            ▼            ▼     ┌───────────────┐
       ┌──────────┐ ┌──────────┐ ┌──────────┐ │  gecko_genes  │
       │  media   │ │ feedings │ │ weights  │ └───────────────┘
       └──────────┘ └──────────┘ └──────────┘
              │
              │       ┌──────────┐       ┌──────────┐       ┌──────────┐       ┌──────────┐
              │       │ pairings │──────▶│ clutches │──────▶│   eggs   │──────▶│ geckos   │
              │       └──────────┘       └──────────┘       └──────────┘       │ (hatched)│
              │             │                                     ▲            └──────────┘
              │             ▼                                     │
              │       ┌──────────┐                         ┌──────────────┐
              │       │  media   │                         │  incubators  │
              │       └──────────┘                         └──────────────┘
              │
              ▼
       (entity: gecko / pairing)

╔══════════════════════════════════════════════════════════════════════╗
║ PRODUCTS / INVENTORY / SUPPLIERS / SHIPMENTS                         ║
║                                                                      ║
║  products ──► product_supply ──► inventory_items ──► inventory_stock ║
║        ╲  ──► product_gecko ──► geckos                               ║
║         ╲ ──► product_components ──► products (recursive, packages)  ║
║                                                                      ║
║  suppliers ──► purchase_orders ──► purchase_order_items              ║
║                  ╲                   │                               ║
║                   ╲─► shipments ──►  receipts ──► receipt_items      ║
║                   ╲─► shipment_costs  │          ──► receipt_item_costs ║
║                                       │                              ║
║                                       └─► inventory_movements        ║
║                                                                      ║
║  pricing_rules (by product_type)                                     ║
╚══════════════════════════════════════════════════════════════════════╝

┌──────────────────────────────────────────────────────────────┐
│  translations  (polymorphic: entity_type + entity_id)        │
│  One row per (entity, field, language).                      │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. Enums (pgEnum)

All PG enums. New enum values require a migration (`ALTER TYPE ... ADD VALUE ...`).

| Enum | Values | Used by |
|---|---|---|
| `species_code` | `LP`, `AF` | `species.code` |
| `sex` | `M`, `F`, `U` | `geckos.sex` |
| `zygosity` | `HOM`, `HET`, `POSS_HET` | `gecko_genes.zygosity` |
| `status` | `ACTIVE`, `INACTIVE`, `SOLD`, `DECEASED` | `geckos.status` |
| `product_type` | `SUPPLY`, `PACKAGE`, `GECKO` | `products.type`, `pricing_rules.product_type` |
| `movement_type` | `IN`, `OUT`, `ADJUST` | `inventory_movements.movement_type` |
| `media_type` | `PROFILE`, `GALLERY`, `PAIRING`, `HUSBANDRY` | `media.type` |
| `egg_status` | `LAID`, `INCUBATING`, `HATCHED`, `INFERTILE`, `DIED` | `eggs.status` |
| `purchase_order_status` | `DRAFT`, `SUBMITTED`, `CONFIRMED`, `RECEIVED`, `CANCELLED` | `purchase_orders.status` |
| `shipment_status` | `PENDING`, `IN_TRANSIT`, `ARRIVED`, `CLEARED` | `shipments.status` |
| `user_role` | `ADMIN`, `STAFF`, `PUBLIC` | `users.role` (superseded — new backend uses `admin_users` table instead) |

**For Go**: `sqlc` maps PG enums to Go string types by default. The new backend can redefine these as typed string consts.

---

## 3. USERS & AUTHENTICATION

### `users` (superseded)

Old legacy table. **Already replaced** in the new backend by `admin_users` (migration `20260420000002_admin_users.sql`). Do NOT port.

```
users
├── id                serial PK
├── email             varchar(255) UNIQUE NOT NULL
├── password_hash     varchar(255) NOT NULL
├── role              user_role NOT NULL DEFAULT 'STAFF'
├── first_name        varchar(100)
├── last_name         varchar(100)
├── is_active         boolean NOT NULL DEFAULT true
├── last_login_at     timestamp
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── updated_at        timestamp DEFAULT NOW() NOT NULL
```

**Verdict**: ⛔ Skip. New `admin_users` table is simpler and already done.

---

## 4. SPECIES & GENETICS

### `species`

```
species
├── id                serial PK
├── code              species_code UNIQUE NOT NULL    -- 'LP' | 'AF'
├── common_name       varchar(100) NOT NULL           -- 'Leopard Gecko'
├── scientific_name   varchar(150)                    -- 'Eublepharis macularius'
├── description       text
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── updated_at        timestamp DEFAULT NOW() NOT NULL
```

Row count in legacy: **2** (LP = Leopard, AF = African Fat-tailed).

### `genetic_dictionary`

The catalog of possible genetic traits per species.

```
genetic_dictionary
├── id                serial PK
├── species_id        integer FK → species.id NOT NULL
├── trait_name        varchar(100) NOT NULL           -- 'Tremper Albino'
├── trait_code        varchar(50)                     -- 'TREMPER'
├── description       text
├── is_dominant       boolean DEFAULT false
├── created_at        timestamp DEFAULT NOW() NOT NULL
├── updated_at        timestamp DEFAULT NOW() NOT NULL
└── UNIQUE (species_id, trait_name)
```

Row count in legacy: **30+** traits. Examples: Tremper Albino, Mack Snow, Eclipse, Blizzard, Rainwater, Bell, etc.

**Verdict for both**: ✅ **Port early (Phase 3 — Geckos CRUD)**. These are the vocabulary the Geckos UI needs for the "assign traits" form. Seed data easy to pull from legacy `sample_translations.sql` + a fresh species seed.

---

## 5. GECKOS

### `geckos`

The central table. Self-referencing for lineage (sire/dam).

```
geckos
├── id                serial PK
├── code              varchar(20) UNIQUE NOT NULL     -- 'LP-004', 'AF-001'
├── name              varchar(100)                    -- 'Star', 'Shadow', etc.
├── species_id        integer FK → species.id NOT NULL
├── sex               sex NOT NULL                    -- 'M' | 'F' | 'U'
├── hatch_date        date
├── acquired_date     date
├── status            status NOT NULL DEFAULT 'ACTIVE'
├── sire_id           integer FK → geckos.id  ◄── self-ref (father)
├── dam_id            integer FK → geckos.id  ◄── self-ref (mother)
├── clutch_id         integer FK → clutches.id       -- where they were born
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
├── updated_at        timestamp DEFAULT NOW() NOT NULL
├── INDEX on species_id
└── INDEX on status
```

### `gecko_genes`

Junction: which traits each gecko carries, and their zygosity.

```
gecko_genes
├── id                serial PK
├── gecko_id          integer FK → geckos.id ON DELETE CASCADE NOT NULL
├── trait_id          integer FK → genetic_dictionary.id NOT NULL
├── zygosity          zygosity NOT NULL               -- HOM | HET | POSS_HET
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── UNIQUE (gecko_id, trait_id)
```

**Verdict**: ✅ **Port early (Phase 3)**. These are the core tables for admin v1. Key design points to preserve:
- Self-referencing FKs on `sire_id`/`dam_id` — keep, even though initially null (no breeding yet).
- `clutch_id` — can be null for foundational animals; populated only for hatched offspring.
- UNIQUE on `code` — matches the user's "LP-004"-style coding convention.
- Cascade delete on `gecko_genes` means removing a gecko wipes its trait assignments. Fine.

---

## 6. MEDIA

### `media`

Polymorphic-ish: a photo can attach to EITHER a gecko OR a pairing. (Not both. Enforced by app logic, not SQL.)

```
media
├── id                serial PK
├── gecko_id          integer FK → geckos.id ON DELETE CASCADE
├── pairing_id        integer FK → pairings.id ON DELETE CASCADE
├── url               varchar(500) NOT NULL
├── type              media_type NOT NULL DEFAULT 'GALLERY'
├── caption           text
├── display_order     integer DEFAULT 0
└── uploaded_at       timestamp DEFAULT NOW() NOT NULL
```

**Verdict**: ✅ **Port in Phase 3** (Geckos CRUD). Simplify for v1 — drop the `pairing_id` column until breeding actually happens. Add it back in the breeding phase via migration. The `display_order` field is how the frontend picks the cover photo (lowest order = cover).

**New backend note**: `url` stores a relative path (`/uploads/geckos/<id>/<file>`) served by the Go static file handler in v1, and a full CDN URL in v2 (R2). No schema change when we flip.

---

## 7. BREEDING

### `pairings`

A male × female × season combination.

```
pairings
├── id                serial PK
├── male_id           integer FK → geckos.id NOT NULL
├── female_id         integer FK → geckos.id NOT NULL
├── season            varchar(20) NOT NULL            -- 'Spring 2026'
├── start_date        date
├── end_date          date
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
├── updated_at        timestamp DEFAULT NOW() NOT NULL
└── UNIQUE (male_id, female_id, season)              -- same pair can re-pair in different seasons
```

### `clutches`

A group of eggs laid together.

```
clutches
├── id                serial PK
├── pairing_id        integer FK → pairings.id NOT NULL
├── lay_date          date NOT NULL
├── egg_count         integer NOT NULL DEFAULT 0
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── updated_at        timestamp DEFAULT NOW() NOT NULL
```

### `incubators`

Physical incubator boxes.

```
incubators
├── id                serial PK
├── name              varchar(100) NOT NULL
├── location          varchar(100)
├── temperature_celsius  decimal(4,1)
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── updated_at        timestamp DEFAULT NOW() NOT NULL
```

### `eggs`

Individual eggs.

```
eggs
├── id                serial PK
├── clutch_id         integer FK → clutches.id NOT NULL
├── incubator_id      integer FK → incubators.id
├── egg_number        integer NOT NULL                -- 1-based within the clutch
├── status            egg_status NOT NULL DEFAULT 'LAID'
├── lay_date          date NOT NULL
├── hatch_date        date
├── gecko_id          integer FK → geckos.id         -- populated on HATCHED
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
├── updated_at        timestamp DEFAULT NOW() NOT NULL
└── UNIQUE (clutch_id, egg_number)
```

**Verdict for breeding domain**: 🟡 **Defer to Phase 7+ (month the first pairing is set up)**. No point modeling now. Waiting produces a better data model once we know how the operator actually tracks breeding. The shape above is a reasonable starting point to return to.

**Key design pattern worth remembering**: The pipeline is `pairings (1) → clutches (N) → eggs (N)`, and when an egg hatches, `eggs.gecko_id` points to the resulting `geckos.id`, which in turn has `clutch_id` pointing back to the clutch. That circular-but-not-cyclic reference is how lineage reconstructs: offspring.clutch → clutch.pairing → pairing.male/female = grandparents.

---

## 8. HUSBANDRY

### `feedings`

```
feedings
├── id                serial PK
├── gecko_id          integer FK → geckos.id ON DELETE CASCADE NOT NULL
├── fed_at            timestamp NOT NULL
├── food_type         varchar(100)                    -- 'dubia roaches', 'crickets', 'mealworms'
├── quantity          integer                         -- number of insects, free-form
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── INDEX (gecko_id, fed_at)
```

### `weights`

```
weights
├── id                serial PK
├── gecko_id          integer FK → geckos.id ON DELETE CASCADE NOT NULL
├── weighed_at        timestamp NOT NULL
├── weight_grams      decimal(6,2) NOT NULL           -- e.g. 47.50
├── notes             text
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── INDEX (gecko_id, weighed_at)
```

**Verdict**: 🟡 **Port when daily logging becomes a habit** (probably Phase 5-6). Weights are useful for pre-breeding conditioning (target weight to breed), so might come earlier than breeding records. The `WeightSparkline.vue` component is already built against mock data — ready to accept real rows.

---

## 9. TRANSLATIONS (i18n)

### `translations` — polymorphic

```
translations
├── id                serial PK
├── entity_type       varchar(50) NOT NULL            -- 'genetic_trait', 'species', 'product'
├── entity_id         integer NOT NULL                -- FK-shaped but not enforced
├── field_name        varchar(50) NOT NULL            -- 'trait_name', 'common_name', 'description'
├── language          varchar(10) NOT NULL            -- 'zh-CN', 'km-KH', 'th-TH'
├── value             text NOT NULL
├── created_at        timestamp DEFAULT NOW() NOT NULL
├── updated_at        timestamp DEFAULT NOW() NOT NULL
├── INDEX (entity_type, entity_id, field_name, language)
└── UNIQUE (entity_type, entity_id, field_name, language)
```

**Design pattern**: one row per (what, which field, which language). Scales to new entities and new languages without schema changes. The foreign-key-shaped `entity_id` isn't a real FK — orphans are possible if an entity is deleted. In practice, acceptable trade-off; cleanup can be a cron job.

**Legacy had 52 rows loaded**: 16 traits in Simplified Chinese, 10 in Khmer. Ready to reimport via `gekko_legacy/gekko_backend/sample_translations.sql`.

**Verdict**: 🟢 **Port in Phase 6 (storefront polish) or Phase 3 (if operator wants multilingual trait names in admin)**. The table is tiny and easy. SQL example:

```sql
SELECT gd.trait_name,
       COALESCE(t.value, gd.trait_name) AS translated
FROM genetic_dictionary gd
LEFT JOIN translations t ON
  t.entity_type = 'genetic_trait' AND
  t.entity_id = gd.id AND
  t.field_name = 'trait_name' AND
  t.language = $1
WHERE gd.id = $2;
```

For Go, the query above compiles fine via sqlc. Consider a `:many` variant that returns all traits for a species with translations applied.

---

## 10. PRODUCTS (3 subtypes via extension tables)

This domain is where the legacy schema got ambitious. **Most of it is skippable for v1.**

### `products` (parent)

```
products
├── id                serial PK
├── sku               varchar(50) UNIQUE NOT NULL
├── name              varchar(200) NOT NULL
├── description       text
├── type              product_type NOT NULL           -- 'SUPPLY' | 'PACKAGE' | 'GECKO'
├── is_active         boolean NOT NULL DEFAULT true
├── created_at        timestamp DEFAULT NOW() NOT NULL
└── updated_at        timestamp DEFAULT NOW() NOT NULL
```

### `product_supply` — extension for supply products (food, feeders, etc.)

```
product_supply
├── id                serial PK
├── product_id        integer FK → products.id ON DELETE CASCADE UNIQUE NOT NULL
├── inventory_item_id integer FK → inventory_items.id
└── created_at        timestamp DEFAULT NOW() NOT NULL
```

### `product_components` — packages are bundles of other products

```
product_components
├── id                     serial PK
├── package_product_id     integer FK → products.id ON DELETE CASCADE NOT NULL
├── component_product_id   integer FK → products.id NOT NULL
├── quantity               integer NOT NULL
├── created_at             timestamp DEFAULT NOW() NOT NULL
└── UNIQUE (package_product_id, component_product_id)
```

Recursive: a "starter kit" package contains N other products.

### `product_gecko` — extension for live-animal products

```
product_gecko
├── id                serial PK
├── product_id        integer FK → products.id ON DELETE CASCADE UNIQUE NOT NULL
├── gecko_id          integer FK → geckos.id NOT NULL
├── list_price        decimal(10,2)
└── created_at        timestamp DEFAULT NOW() NOT NULL
```

**Verdict (revised 2026-04-21):** 🟢 **Port when commerce arrives — Phase 6.5.** Not as the legacy 4-table polymorphic model (that's what the original verdict correctly rejected), but as a cleaner 2- or 3-table design:

```
listings
├── id              serial PK
├── sku             varchar (nullable — for non-gecko items)
├── type            enum('GECKO', 'PACKAGE', 'SUPPLY')
├── title           varchar
├── description     text
├── price_usd       numeric (explicit — even packages store price directly so bundles can discount)
├── deposit_usd     numeric (nullable)
├── status          enum('DRAFT','LISTED','RESERVED','SOLD','ARCHIVED')
├── cover_photo_url varchar (for SUPPLY/PACKAGE; GECKO listings pull from media)
├── listed_at / sold_at / archived_at / created_at / updated_at

listing_geckos              (junction for GECKO-type listings — supports pair/trio bundles)
├── listing_id
├── gecko_id
└── PK (listing_id, gecko_id)

listing_components          (junction for PACKAGE-type listings — flat bundling)
├── listing_id              (the package)
├── component_listing_id    (the contained supply/gecko listing)
├── quantity                integer
└── PK (listing_id, component_listing_id)
```

Gecko row stays pure biology — no `list_price_usd`. A gecko is for sale when a `LISTED` listing references it; a gecko on HOLD is a biology-status (long stay) independent of whether a listing is reserved.

**Why this beats the legacy 4-table model:** 2 tables, not 4. One product type column, no polymorphic extension tables. Packages compose flat (no recursive package-in-package — YAGNI). Pricing lives on the listing, so bundle discounts and price-over-time both work naturally.

**Why this beats `geckos.list_price_usd` (the naive path):** lets a gecko have different prices at different times (or none), supports pair bundles, decouples "for sale" from "alive/healthy." Clean separation: biology vs commerce.

**What stays skipped:** the `product_supply → inventory_item` extension — not needed until Tier 2 (below). `product_components` recursion (package-inside-package) — flat bundling covers the basic/intermediate/premium starter kits the operator is planning.

**Scope ladder:**
- **Tier 1 (Phase 6.5 — port now or when commerce arrives):** `listings`, `listing_geckos`, `listing_components`. Data migration moves existing `geckos.list_price_usd` into listings and drops the column.
- **Tier 2 (when supply stock is physically held):** add `stock_on_hand INTEGER DEFAULT 0` to listings + a tiny "adjust stock" UI. No separate inventory tables — single column suffices for Zen's scale.
- **Tier 3 (if we ever import containers internationally):** revisit suppliers/POs/receipts/landed-cost. Unlikely for years; still 🔴 skip.

**What the original verdict got right:** 4 polymorphic tables = pain. **What it got wrong:** conflated "the 4-table model is bad" with "no commerce layer at all." Zen's vision (gecko = biology, listing = commerce, supplies + packages as first-class commerce objects) is architecturally sound and cheap to implement once as described above.

---

## 11. INVENTORY (Supplies stock-keeping)

### `inventory_items`

```
inventory_items
├── id                    serial PK
├── item_code             varchar(50) UNIQUE NOT NULL
├── name                  varchar(200) NOT NULL
├── description           text
├── unit_of_measure       varchar(20) DEFAULT 'EA'
├── avg_unit_cost         decimal(10,2) DEFAULT '0'
├── created_at            timestamp DEFAULT NOW() NOT NULL
└── updated_at            timestamp DEFAULT NOW() NOT NULL
```

### `inventory_stock` (per-location)

```
inventory_stock
├── id                      serial PK
├── inventory_item_id       integer FK → inventory_items.id ON DELETE CASCADE NOT NULL
├── location                varchar(100) DEFAULT 'DEFAULT'
├── quantity_on_hand        integer NOT NULL DEFAULT 0
├── created_at              timestamp DEFAULT NOW() NOT NULL
├── updated_at              timestamp DEFAULT NOW() NOT NULL
└── UNIQUE (inventory_item_id, location)
```

### `inventory_movements` (audit log)

```
inventory_movements
├── id                      serial PK
├── inventory_item_id       integer FK → inventory_items.id NOT NULL
├── movement_type           movement_type NOT NULL           -- IN | OUT | ADJUST
├── quantity                integer NOT NULL
├── location                varchar(100) DEFAULT 'DEFAULT'
├── unit_cost               decimal(10,2)
├── reference_type          varchar(50)                      -- 'purchase_order' | 'adjustment' | 'sale'
├── reference_id            integer                          -- soft-reference
├── notes                   text
├── moved_at                timestamp DEFAULT NOW() NOT NULL
├── created_at              timestamp DEFAULT NOW() NOT NULL
└── INDEX (inventory_item_id, moved_at)
```

**Verdict**: 🔴 **Skip**. Inventory tracking only makes sense if the operator sells supplies. Not a v1+v2 concern.

---

## 12. SUPPLIERS & PURCHASING (the landed-cost chain)

Five tables implementing landed-cost allocation: ordered cost + freight + customs ÷ quantity received = true per-unit cost.

### `suppliers`, `purchase_orders`, `purchase_order_items`, `receipts`, `receipt_items`, `receipt_item_costs`

The domain is:

```
suppliers ─┬─► purchase_orders ──► purchase_order_items
           │         │                    │
           │         ▼                    ▼
           └─► shipments ─────► receipts ──► receipt_items ──► receipt_item_costs
                     │                                          (base + landed
                     ▼                                           surcharge)
               shipment_costs
               (freight/customs)
```

Full table definitions are in the legacy `schema.ts`. Call it 6 tables of supply-chain bookkeeping that correctly computes landed unit cost per SKU when receiving goods.

**Verdict**: 🔴 **Skip**. Same reason as products — no supplies resale. A breeding business buys insects on a market stall with cash — the landed-cost machinery is designed for international container imports. Massive overkill.

---

## 13. SHIPMENTS & FREIGHT

### `shipments`, `shipment_costs`

Freight container / courier tracking. Sister to purchasing above.

**Verdict**: 🔴 **Skip**. Only relevant if we re-adopt purchasing.

---

## 14. PRICING

### `pricing_rules`

```
pricing_rules
├── id                        serial PK
├── product_type              product_type UNIQUE NOT NULL
├── default_margin_percent    decimal(5,2) NOT NULL
├── created_at                timestamp DEFAULT NOW() NOT NULL
└── updated_at                timestamp DEFAULT NOW() NOT NULL
```

Per-product-type default margin (e.g., SUPPLY = 40%, GECKO = 0% — manually priced).

**Verdict**: 🔴 **Skip**. Premature optimization. Prices live on products/geckos directly when needed.

---

## 15. Design patterns worth keeping

These are the non-obvious things the legacy author got right. Worth considering for the Go rewrite:

### 15.1 Self-referencing gecko parentage
`geckos.sire_id` and `geckos.dam_id` point back at `geckos.id`. Keep this — it's the right model for lineage, even if `sire_id`/`dam_id` are null for foundational animals.

### 15.2 Polymorphic translations
`translations (entity_type, entity_id, field_name, language)` scales cleanly. Accept the soft-FK trade-off (no DB-level cascade) for the flexibility. Great for adding Thai, Vietnamese, Japanese later without schema changes.

### 15.3 Media with polymorphic parent (soft)
`media` attaches to `gecko_id` OR `pairing_id`. Soft-enforced by app. Alternative Go model: `(entity_type, entity_id)` like translations, or just keep the two-nullable-FK approach (simpler SQL, fine for 2-3 parent types).

### 15.4 Egg → Gecko back-reference
`eggs.gecko_id` populated only after hatch, `geckos.clutch_id` populated only for hatched animals. This allows lineage walks: `gecko → clutch → pairing → male/female (grandparents)`. Clean.

### 15.5 Indexed lookup columns
Every foreign key that's queried in aggregate (e.g., `geckos.species_id`, `feedings.gecko_id`, `weights.gecko_id`) has a b-tree index. Keep this discipline in the goose migrations.

### 15.6 Unique composite indexes
- `(gecko_id, trait_id)` on `gecko_genes` — prevents duplicate trait assignments
- `(male_id, female_id, season)` on `pairings` — one pairing record per season
- `(clutch_id, egg_number)` on `eggs` — enforces 1-based numbering per clutch
- `(entity_type, entity_id, field_name, language)` on `translations` — one translation per (what, field, language)

Port all of these when porting the underlying table.

---

## 16. Patterns to AVOID from the legacy

### 16.1 Three tables for "a product subtype"
`products + product_supply + product_gecko + product_components` = 4 joins to render a listing page. The replacement design (see §10 verdict) is one `listings` table with a `type` discriminator + two junction tables (`listing_geckos` and `listing_components`). 3 tables, not 4, with no polymorphic extension tables.

### 16.2 `users` table with a PUBLIC role
The legacy `users.role` enum has `ADMIN / STAFF / PUBLIC`. But PUBLIC users never logged in — the value was aspirational. The new `admin_users` table doesn't make that mistake.

### 16.3 Over-eager cost allocation
`receipt_item_costs (base + landed_surcharge + landed_unit_cost)` correctly models container shipping landed-cost. But it's premature for anyone buying crickets locally. Add this only when you actually import stock internationally.

### 16.4 Ambiguous "reference" columns
`inventory_movements.reference_type + reference_id` is a soft FK to any source document (purchase_order, adjustment, sale). Without strict enforcement, this gets messy over time. If you port inventory later, consider a proper discriminated-union approach or a dedicated `stock_adjustments` table per movement reason.

---

## 17. Recommended porting order (mapped to new phases)

| Phase | Legacy tables to port | Rationale |
|---|---|---|
| **2 — Waitlist** (done) | None | `waitlist_entries` is already a new table. |
| **3 — Geckos CRUD + photos + genetics** (done) | `species`, `genetic_dictionary`, `geckos`, `gecko_genes`, `media` | Core of admin v1. |
| **4 — Data visualizer** (done) | None | Reads existing tables via schema introspection. |
| **5 — Dashboard stats** (done) | None | Aggregate queries on tables from Phase 3. |
| **6 — Storefront MVP** (done) | None | No new tables — reuses `geckos`, `waitlist_entries`. |
| **6.5 — Commerce model** | Replace `geckos.list_price_usd` with new `listings`, `listing_geckos`, `listing_components` tables (Tier 1 of §10 verdict). | Separates biology from commerce. Supports supplies + starter-kit packages as first-class listings. Triggered by operator planning to sell supplies alongside geckos. |
| **7 — Husbandry quick-log** | `feedings`, `weights` | When daily logging becomes habitual. |
| **8 — Breeding season 1** | `pairings`, `clutches`, `incubators`, `eggs` | When operator sets up first pair. Also re-enables `media.pairing_id` and `geckos.clutch_id`. |
| **i18n (optional — can slot into 6.5 or later)** | `translations` | When multilingual trait names / storefront copy are wanted. |
| **Never (probably)** | Legacy `users` (superseded), `products`+extension tables (replaced by new `listings`), `inventory_*` beyond a simple stock column, `suppliers`, `purchase_orders`, `purchase_order_items`, `receipts`, `receipt_items`, `receipt_item_costs`, `shipments`, `shipment_costs`, `pricing_rules` | Container-import and supplies-resale infrastructure the operator isn't building. |

**Total worth porting**: ~13 tables (11 legacy + 2 new listings junctions). The rest is archaeological interest.

---

## 18. Seed data available to reimport

Inside the legacy GitHub repo `jxnhoongz/gekko_backend`:

- `migrations/*.sql` — raw Drizzle-generated schema for all 28 tables. **Hard-port with caution**; only pull the tables you're actually using, don't blind-apply.
- `sample_translations.sql` — 52 translation rows (Chinese + Khmer for common traits). Ready to `psql < sample_translations.sql` after `translations` table is created.
- `TRANSLATIONS.md` / `SUMMARY_TRANSLATIONS.md` — implementation notes for the i18n system, copied from `gekko_legacy/gekko_backend/` if needed.
- The 2 species rows and ~30 genetic traits were seeded via a Node script that no longer exists. For the new backend, write a Go seed command (similar to the existing `cmd/gekko-seed/` pattern used for admin bootstrap) that inserts:
  - `species`: `{code: 'LP', common_name: 'Leopard Gecko', scientific_name: 'Eublepharis macularius'}`, `{code: 'AF', common_name: 'African Fat-tailed Gecko', scientific_name: 'Hemitheconyx caudicinctus'}`
  - `genetic_dictionary`: trait list (ask the operator or pull from the legacy DB dump if still accessible on Mac)

---

## 19. How to use this doc

- **Before each new phase**: re-read the domain section for the tables involved. The "verdict" line tells you whether to port as-is, adapt, or skip.
- **Don't port in bulk**. Port table-by-table as phases demand. Each port is a goose migration + sqlc query file + handler. Small batches are easier to review and test.
- **Contradict it freely**. If the Go rewrite has a better idea (e.g., a simpler product model), use it. This doc is informational — the Go schema is authoritative.
- **Update this doc** when porting: mark ported tables with a ✅ and a link to the new migration file, so the doc evolves into an "old→new mapping" as work progresses.

---

## 20. Quick reference — legacy FK chart

```
users            (no inbound FKs)
species          ← geckos, genetic_dictionary
genetic_dictionary ← gecko_genes
geckos           ← geckos (sire/dam), gecko_genes, media, feedings, weights,
                    pairings (M & F), eggs (hatched), product_gecko
                    (self-ref: sire_id, dam_id, clutch_id)
media            (attaches to gecko OR pairing)
pairings         ← clutches, media
clutches         ← eggs, geckos (hatched, via geckos.clutch_id)
incubators       ← eggs
eggs             (attaches to clutch + optional incubator + optional hatched gecko)
feedings         (attaches to gecko)
weights          (attaches to gecko)
translations     (polymorphic, no hard FKs)
products         ← product_supply, product_gecko, product_components (×2)
product_supply   (product ↔ inventory_item)
product_components (product ↔ product, bundle of other products)
product_gecko    (product ↔ gecko)
inventory_items  ← inventory_stock, inventory_movements, product_supply,
                    purchase_order_items, receipt_items
suppliers        ← purchase_orders, shipments
purchase_orders  ← purchase_order_items, receipts
purchase_order_items  ← receipt_items
receipts         ← receipt_items
receipt_items    ← receipt_item_costs
shipments        ← shipment_costs, receipts
pricing_rules    (no inbound FKs)
```

---

End of reference. Questions: open an issue or raise in the next session.
