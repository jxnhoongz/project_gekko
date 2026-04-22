# Genetics Refactor — Morph Combos & Inheritance Types Design

## Goal

Extend the genetics model so the system knows *how* each base trait is inherited and *which named combos* are formed by combining traits. This unlocks:

- **Display:** "Diablo Blanco" instead of "Tremper Albino Eclipse Blizzard"
- **Auto-detection:** backend labels a gecko's morph by matching its trait set against known combos
- **Knowledge catalog:** each base trait and combo has a description, operator notes, and an example photo URL
- **Calculator foundation:** inheritance_type on every trait enables future Mendelian outcome predictions

Scope: **Leopard Gecko (LP) only.** African Fat-tail and Crested Gecko genetics added later.

---

## Data Model

### 1. `genetic_dictionary` — new columns

```sql
ALTER TABLE genetic_dictionary
  ADD COLUMN inheritance_type   inheritance_type NOT NULL DEFAULT 'RECESSIVE',
  ADD COLUMN super_form_name    VARCHAR(100),
  ADD COLUMN example_photo_url  VARCHAR(500),
  ADD COLUMN notes              TEXT;
```

New enum:

```sql
CREATE TYPE inheritance_type AS ENUM (
  'RECESSIVE',    -- 2 copies required to express visually
  'CO_DOMINANT',  -- 1 copy = partial expression; 2 copies = named "super" form
  'DOMINANT',     -- 1 copy expresses; HOM usually lethal
  'POLYGENIC'     -- line-bred; no clean Mendelian ratio
);
```

`super_form_name` is only populated for `CO_DOMINANT` traits — it is the industry name for the homozygous form (e.g. `"Super Snow"` for Mack Snow, `"Super Giant"` for Giant).

`notes` is the operator's informal observations, separate from the public-facing `description`.

### 2. `morph_combos` — new table

Named combinations of 2+ base traits.

```sql
CREATE TABLE morph_combos (
  id               SERIAL PRIMARY KEY,
  species_id       INTEGER NOT NULL REFERENCES species(id) ON DELETE RESTRICT,
  name             VARCHAR(100) NOT NULL,
  code             VARCHAR(50),
  description      TEXT,
  notes            TEXT,
  example_photo_url VARCHAR(500),
  created_at       TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at       TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE (species_id, name)
);
CREATE INDEX morph_combos_species_idx ON morph_combos (species_id);
```

### 3. `morph_combo_traits` — junction table

Defines which traits (and at what minimum zygosity) are required for a gecko to be labeled as a given combo.

```sql
CREATE TABLE morph_combo_traits (
  combo_id          INTEGER NOT NULL REFERENCES morph_combos(id) ON DELETE CASCADE,
  trait_id          INTEGER NOT NULL REFERENCES genetic_dictionary(id) ON DELETE RESTRICT,
  required_zygosity zygosity NOT NULL DEFAULT 'HOM',
  PRIMARY KEY (combo_id, trait_id)
);
CREATE INDEX morph_combo_traits_trait_idx ON morph_combo_traits (trait_id);
```

`required_zygosity` uses the existing `zygosity` enum (`HOM`, `HET`, `POSS_HET`). For recessive combos all constituent traits require `HOM`. For combos involving dominant traits, `HET` is sufficient.

---

## Seed Data — LP Traits (corrected)

### Rows removed from `genetic_dictionary`

| Trait | Reason |
|---|---|
| Raptor | Combo morph — moves to `morph_combos` |
| Super Snow | Derived form — auto-generated from `CO_DOMINANT` Mack Snow at `HOM` zygosity |

### Updated LP trait catalog

| Trait Name | Code | inheritance_type | super_form_name |
|---|---|---|---|
| Tremper Albino | TREM | RECESSIVE | — |
| Bell Albino | BELL | RECESSIVE | — |
| Rainwater Albino | RAIN | RECESSIVE | — |
| Blizzard | BLIZ | RECESSIVE | — |
| Murphy Patternless | MP | RECESSIVE | — |
| Eclipse | ECL | RECESSIVE | — |
| Marble Eye | ME | RECESSIVE | — |
| Mack Snow | MACK | CO_DOMINANT | Super Snow |
| Giant | GIANT | CO_DOMINANT | Super Giant |
| Lemon Frost ⚠️ | LF | CO_DOMINANT | Super Lemon Frost |
| Enigma ⚠️ | ENIG | DOMINANT | — |
| White & Yellow | WY | DOMINANT | — |
| Tangerine | TANG | POLYGENIC | — |
| Carrot Tail | CT | POLYGENIC | — |
| Hypo | HYPO | POLYGENIC | — |
| Super Hypo | SHYPO | POLYGENIC | — |
| Baldy | BALDY | POLYGENIC | — |
| Bold Stripe | BSTR | POLYGENIC | — |
| Melanistic / Black Night | BN | POLYGENIC | — |

⚠️ Health warnings stored in `notes`:
- **Lemon Frost:** linked to iridophoroma (white cell tumors); no way to avoid in phenotype.
- **Enigma:** causes Enigma Syndrome (neurological — star-gazing, seizures, head-tilting); homozygous likely lethal.

### Seeded LP morph combos

| Combo | Code | Required traits (all HOM unless noted) |
|---|---|---|
| Raptor | RAPT | Tremper Albino, Eclipse |
| Diablo Blanco | DBLAN | Tremper Albino, Blizzard, Eclipse |
| Firewater | FIRW | Rainwater Albino, Eclipse |
| Electric | ELEC | Bell Albino, Eclipse |
| Blazing Blizzard | BLAZBLIZ | Blizzard, Tremper Albino |

More combos can be added via the admin UI at any time — no migration required.

---

## Morph Detection Logic (Go)

A pure function `DetectMorph(species int, traits []GeckoGeneRow, combos []MorphComboWithTraits) string` replaces both `composePublicMorph` (public.go) and `morphFromTraits` (frontend TypeScript).

Algorithm:

```
matched_combos = []
covered_trait_ids = set()

for each combo in combos (ordered by len(requirements) DESC — longest match first):
  if all combo.requirements satisfied by gecko traits (zygosity >= required_zygosity):
    matched_combos.append(combo.name)
    cover_trait_ids.union(combo.trait_ids)

remaining = []
for each gecko trait not in covered_trait_ids:
  if trait.inheritance_type == CO_DOMINANT and trait.zygosity == HOM:
    remaining.append(trait.super_form_name)
  elif trait.zygosity == HET:
    remaining.append("het " + trait.trait_name)
  elif trait.zygosity == POSS_HET:
    remaining.append("poss. het " + trait.trait_name)
  else:  # HOM, non-combo
    remaining.append(trait.trait_name)

if matched_combos + remaining is empty:
  return "Normal"

return join(matched_combos + remaining, " ")
```

Example outputs:
- HOM Tremper + HOM Eclipse → `"Raptor"`
- HOM Tremper + HOM Eclipse + HET Mack Snow → `"Raptor het Mack Snow"`
- HOM Mack Snow + HOM Tremper + HOM Eclipse → `"Raptor Super Snow"`
- HOM Tremper + HOM Blizzard + HOM Eclipse → `"Diablo Blanco"`
- HOM Mack Snow alone → `"Super Snow"`
- HOM Tangerine alone → `"Tangerine"` (POLYGENIC, HOM displayed as-is)
- No traits → `"Normal"`

The combos list is fetched from the DB once per request (or per gecko-list batch) and passed in — no N+1 queries.

---

## Backend API

### Updated gecko responses

`ListGeckos`, `GetGeckoByID`, `GetGeckoByCode`, `ListAvailableGeckos`, `GetAvailableGeckoByCode` all gain a computed `morph_label` string field, replacing the client-side composition. The raw `traits` array is still returned on detail endpoints for the genetics tab.

### New: `MountMorphCombos` handler

```
GET    /api/morph-combos         — list all (query param: ?species_code=LP)
POST   /api/morph-combos         — create combo + junction rows (transaction)
GET    /api/morph-combos/:id     — detail with full trait requirements
PATCH  /api/morph-combos/:id     — update metadata + replace junction rows
DELETE /api/morph-combos/:id     — cascade via FK
```

All endpoints require admin JWT auth (same as existing admin endpoints).

### Updated: genetic_dictionary via schema endpoint

`GET /api/schema` (existing) — returns `inheritance_type`, `super_form_name`, `notes`, `example_photo_url` in the trait objects. No new endpoints needed.

---

## Admin UI

### Morph Combos page

New nav entry (under Schema section). Shows a grid of combo cards:
- Combo name + code
- Constituent trait badges (color-coded by zygosity requirement)
- Example photo thumbnail if set
- Edit / Delete actions

"Add combo" opens a slide-in sheet:
- Name, code, species selector, description, notes, example_photo_url
- Trait picker: choose a trait, set required_zygosity, add to list

### Updated Genetic Dictionary section

Existing SchemaView trait table gains:
- `inheritance_type` badge (color-coded: blue = RECESSIVE, amber = CO_DOMINANT, red = DOMINANT, gray = POLYGENIC)
- `super_form_name` column (shown for CO_DOMINANT rows)
- Health warning icon (⚠️) for Enigma and Lemon Frost
- `notes` and `example_photo_url` fields visible on row expansion or edit sheet

### Updated gecko displays

`GeckoCard` and `GeckoDetailView` render `morph_label` from the API response directly. The `morphFromTraits` TypeScript function is removed. No other UI changes needed.

---

## What Does Not Change

- `gecko_genes` table schema — unchanged.
- Trait assignment flow in the admin — unchanged.
- Storefront public API shape — `morph_label` replaces the computed morph string, same field name in the DTO.
- Zygosity enum values — unchanged.

---

## Out of Scope

- African Fat-tail and Crested Gecko trait/combo catalogs (added later as reference data).
- Mendelian outcome calculator UI (data foundation is ready; UI is a separate phase).
- Locus modeling (allelic relationships between Tremper/Bell/Rainwater — a future enhancement if calculator needs it).
- Gallery UI for the knowledge catalog (separate phase).
- Trait assignment UI improvements (separate phase).
