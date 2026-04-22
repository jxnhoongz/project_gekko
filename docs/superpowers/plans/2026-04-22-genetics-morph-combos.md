# Genetics Morph Combos Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Model named morph combos and inheritance types so the backend auto-labels each gecko's morph (e.g. "Diablo Blanco" instead of "Tremper Albino Eclipse Blizzard"), replacing the client-side `morphFromTraits` TypeScript function.

**Architecture:** New DB tables (`morph_combos`, `morph_combo_traits`) and new columns on `genetic_dictionary` (`inheritance_type`, `super_form_name`, `notes`, `example_photo_url`). A pure Go function `morph.Detect` matches traits against combos with longest-match-first greedy ordering. All gecko API responses gain a `morph_label` field. Admin UI gains a Morph Combos management page.

**Tech Stack:** Go 1.23, chi/v5, pgx/v5, sqlc v1.27, goose — Vue 3.5, TypeScript, TanStack Vue Query, shadcn-vue, lucide-vue-next

---

## File Map

| File | Action |
|---|---|
| `backend/migrations/20260422000009_genetics_morph_combos.sql` | Create |
| `backend/internal/queries/genetic_dictionary.sql` | Modify |
| `backend/internal/queries/gecko_genes.sql` | Modify |
| `backend/internal/queries/public.sql` | Modify |
| `backend/internal/queries/morph_combos.sql` | Create |
| `backend/internal/db/` | Generate (sqlc) |
| `backend/internal/morph/detect.go` | Create |
| `backend/internal/morph/detect_test.go` | Create |
| `backend/internal/http/geckos.go` | Modify |
| `backend/internal/http/public.go` | Modify |
| `backend/internal/http/morphcombos.go` | Create |
| `backend/internal/http/morphcombos_test.go` | Create |
| `backend/cmd/gekko/main.go` | Modify |
| `apps/admin/src/types/morph.ts` | Create |
| `apps/admin/src/types/gecko.ts` | Modify |
| `apps/admin/src/composables/useMorphCombos.ts` | Create |
| `apps/admin/src/views/MorphCombosView.vue` | Create |
| `apps/admin/src/components/MorphComboFormSheet.vue` | Create |
| `apps/admin/src/layouts/AppShell.vue` | Modify |
| `apps/admin/src/router/index.ts` | Modify |
| `apps/admin/src/components/GeckoCard.vue` | Modify |

---

## Task 1: Migration

**Files:**
- Create: `backend/migrations/20260422000009_genetics_morph_combos.sql`

- [ ] **Step 1: Write the migration**

```sql
-- +goose Up
-- +goose StatementBegin

CREATE TYPE inheritance_type AS ENUM (
  'RECESSIVE',
  'CO_DOMINANT',
  'DOMINANT',
  'POLYGENIC'
);

ALTER TABLE genetic_dictionary
  ADD COLUMN inheritance_type   inheritance_type NOT NULL DEFAULT 'RECESSIVE',
  ADD COLUMN super_form_name    VARCHAR(100),
  ADD COLUMN example_photo_url  VARCHAR(500),
  ADD COLUMN notes              TEXT;

-- Remove traits being moved to morph_combos.
-- Safety: verified 0 gecko_genes rows reference these IDs as of 2026-04-22.
-- If this ever runs against a DB that does have references, rewrite those
-- gecko_genes rows to component traits before deleting.
DELETE FROM genetic_dictionary
WHERE (species_id, trait_name) IN (
  ((SELECT id FROM species WHERE code = 'LP'), 'Raptor'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Snow')
);

-- Set correct inheritance types for existing LP traits.
-- Default 'RECESSIVE' already covers: Tremper Albino, Bell Albino,
-- Rainwater Albino, Blizzard, Murphy Patternless, Eclipse, Tangerine,
-- Hypo, Super Hypo, Bold Stripe (all previously is_dominant=FALSE).

UPDATE genetic_dictionary
SET inheritance_type = 'CO_DOMINANT', super_form_name = 'Super Snow'
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name = 'Mack Snow';

UPDATE genetic_dictionary
SET inheritance_type = 'DOMINANT',
    notes = 'Causes Enigma Syndrome (neurological — star-gazing, seizures, head-tilting); homozygous likely lethal.'
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name = 'Enigma';

UPDATE genetic_dictionary
SET inheritance_type = 'DOMINANT'
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name = 'W&Y (White and Yellow)';

UPDATE genetic_dictionary
SET inheritance_type = 'POLYGENIC'
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name IN ('Tangerine', 'Hypo', 'Super Hypo', 'Bold Stripe');

-- Add LP traits missing from original seed.
INSERT INTO genetic_dictionary
  (species_id, trait_name, trait_code, description, is_dominant, inheritance_type, super_form_name, notes)
VALUES
  ((SELECT id FROM species WHERE code = 'LP'),
   'Marble Eye', 'ME', 'Recessive eye morph (marble pattern).', FALSE, 'RECESSIVE', NULL, NULL),
  ((SELECT id FROM species WHERE code = 'LP'),
   'Giant', 'GIANT', 'Co-dominant size morph.', FALSE, 'CO_DOMINANT', 'Super Giant', NULL),
  ((SELECT id FROM species WHERE code = 'LP'),
   'Lemon Frost', 'LF', 'Co-dominant yellow/white morph.', FALSE, 'CO_DOMINANT', 'Super Lemon Frost',
   'Linked to iridophoroma (white cell tumors); no way to avoid in phenotype.'),
  ((SELECT id FROM species WHERE code = 'LP'),
   'Carrot Tail', 'CT', 'Polygenic orange tail.', FALSE, 'POLYGENIC', NULL, NULL),
  ((SELECT id FROM species WHERE code = 'LP'),
   'Baldy', 'BALDY', 'No head pattern.', FALSE, 'POLYGENIC', NULL, NULL),
  ((SELECT id FROM species WHERE code = 'LP'),
   'Melanistic / Black Night', 'BN', 'Polygenic dark/black coloration.', FALSE, 'POLYGENIC', NULL, NULL)
ON CONFLICT (species_id, trait_name) DO NOTHING;

-- Named combo table.
CREATE TABLE morph_combos (
  id                SERIAL PRIMARY KEY,
  species_id        INTEGER NOT NULL REFERENCES species(id) ON DELETE RESTRICT,
  name              VARCHAR(100) NOT NULL,
  code              VARCHAR(50),
  description       TEXT,
  notes             TEXT,
  example_photo_url VARCHAR(500),
  created_at        TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at        TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE (species_id, name)
);
CREATE INDEX morph_combos_species_idx ON morph_combos (species_id);

-- Junction: which traits (at what minimum zygosity) form a combo.
CREATE TABLE morph_combo_traits (
  combo_id          INTEGER NOT NULL REFERENCES morph_combos(id) ON DELETE CASCADE,
  trait_id          INTEGER NOT NULL REFERENCES genetic_dictionary(id) ON DELETE RESTRICT,
  required_zygosity zygosity NOT NULL DEFAULT 'HOM',
  PRIMARY KEY (combo_id, trait_id)
);
CREATE INDEX morph_combo_traits_trait_idx ON morph_combo_traits (trait_id);

-- Seed LP combos.
INSERT INTO morph_combos (species_id, name, code) VALUES
  ((SELECT id FROM species WHERE code = 'LP'), 'Raptor',           'RAPT'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Diablo Blanco',    'DBLAN'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Firewater',        'FIRW'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Electric',         'ELEC'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Blazing Blizzard', 'BLAZBLIZ');

-- Raptor = Tremper Albino HOM + Eclipse HOM
INSERT INTO morph_combo_traits (combo_id, trait_id, required_zygosity)
SELECT (SELECT id FROM morph_combos WHERE name = 'Raptor'), id, 'HOM'
FROM genetic_dictionary
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name IN ('Tremper Albino', 'Eclipse');

-- Diablo Blanco = Tremper Albino HOM + Blizzard HOM + Eclipse HOM
INSERT INTO morph_combo_traits (combo_id, trait_id, required_zygosity)
SELECT (SELECT id FROM morph_combos WHERE name = 'Diablo Blanco'), id, 'HOM'
FROM genetic_dictionary
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name IN ('Tremper Albino', 'Blizzard', 'Eclipse');

-- Firewater = Rainwater Albino HOM + Eclipse HOM
INSERT INTO morph_combo_traits (combo_id, trait_id, required_zygosity)
SELECT (SELECT id FROM morph_combos WHERE name = 'Firewater'), id, 'HOM'
FROM genetic_dictionary
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name IN ('Rainwater Albino', 'Eclipse');

-- Electric = Bell Albino HOM + Eclipse HOM
INSERT INTO morph_combo_traits (combo_id, trait_id, required_zygosity)
SELECT (SELECT id FROM morph_combos WHERE name = 'Electric'), id, 'HOM'
FROM genetic_dictionary
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name IN ('Bell Albino', 'Eclipse');

-- Blazing Blizzard = Blizzard HOM + Tremper Albino HOM
INSERT INTO morph_combo_traits (combo_id, trait_id, required_zygosity)
SELECT (SELECT id FROM morph_combos WHERE name = 'Blazing Blizzard'), id, 'HOM'
FROM genetic_dictionary
WHERE species_id = (SELECT id FROM species WHERE code = 'LP')
  AND trait_name IN ('Blizzard', 'Tremper Albino');

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS morph_combo_traits;
DROP TABLE IF EXISTS morph_combos;

-- Restore deleted traits (data only — no gecko_genes rows pointed at these).
INSERT INTO genetic_dictionary (species_id, trait_name, trait_code, description, is_dominant) VALUES
  ((SELECT id FROM species WHERE code = 'LP'), 'Raptor',     'RAPT',  'Combo: Tremper Albino + Eclipse.', FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Snow', 'SSNOW', 'Homozygous Mack Snow.',             TRUE)
ON CONFLICT DO NOTHING;

ALTER TABLE genetic_dictionary
  DROP COLUMN IF EXISTS inheritance_type,
  DROP COLUMN IF EXISTS super_form_name,
  DROP COLUMN IF EXISTS example_photo_url,
  DROP COLUMN IF EXISTS notes;

DROP TYPE IF EXISTS inheritance_type;
-- +goose StatementEnd
```

- [ ] **Step 2: Run the migration**

```bash
cd backend && DB_URL=$(grep DB_URL .env.local | cut -d= -f2-) \
  go run github.com/pressly/goose/v3/cmd/goose@v3 \
  -dir migrations postgres "$DB_URL" up
```

Expected: `OK   20260422000009_genetics_morph_combos.sql`

- [ ] **Step 3: Verify**

```bash
psql "$DB_URL" -c "\d genetic_dictionary" | grep -E 'inheritance|super_form|notes|example'
psql "$DB_URL" -c "SELECT name, code FROM morph_combos ORDER BY name;"
```

Expected: 4 new columns visible; 5 combo rows (Blazing Blizzard, Diablo Blanco, Electric, Firewater, Raptor).

- [ ] **Step 4: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/migrations/20260422000009_genetics_morph_combos.sql
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(db): genetics morph combos migration + LP seed"
```

---

## Task 2: sqlc SQL Queries

**Files:**
- Modify: `backend/internal/queries/genetic_dictionary.sql`
- Modify: `backend/internal/queries/gecko_genes.sql`
- Modify: `backend/internal/queries/public.sql`
- Create: `backend/internal/queries/morph_combos.sql`

- [ ] **Step 1: Update `genetic_dictionary.sql`** — add new columns to all SELECTs and RETURNING clauses

Replace entire file:

```sql
-- name: ListTraits :many
SELECT id, species_id, trait_name, trait_code, description, is_dominant,
       inheritance_type, super_form_name, example_photo_url, notes,
       created_at, updated_at
FROM genetic_dictionary
ORDER BY species_id, trait_name;

-- name: ListTraitsBySpecies :many
SELECT id, species_id, trait_name, trait_code, description, is_dominant,
       inheritance_type, super_form_name, example_photo_url, notes,
       created_at, updated_at
FROM genetic_dictionary
WHERE species_id = $1
ORDER BY trait_name;

-- name: GetTraitByNameAndSpecies :one
SELECT id, species_id, trait_name, trait_code, description, is_dominant,
       inheritance_type, super_form_name, example_photo_url, notes,
       created_at, updated_at
FROM genetic_dictionary
WHERE species_id = $1 AND LOWER(trait_name) = LOWER($2)
LIMIT 1;

-- name: CreateTrait :one
INSERT INTO genetic_dictionary
  (species_id, trait_name, trait_code, description, is_dominant)
VALUES ($1, $2, $3, $4, COALESCE($5, FALSE))
RETURNING id, species_id, trait_name, trait_code, description, is_dominant,
          inheritance_type, super_form_name, example_photo_url, notes,
          created_at, updated_at;
```

- [ ] **Step 2: Update `gecko_genes.sql`** — add `inheritance_type`, `super_form_name` to both join queries

Replace entire file:

```sql
-- name: ListGeckoGenes :many
SELECT
  gg.id, gg.gecko_id, gg.trait_id, gg.zygosity, gg.created_at,
  gd.trait_name, gd.trait_code, gd.is_dominant, gd.species_id,
  gd.inheritance_type, gd.super_form_name
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
ORDER BY gg.gecko_id, gd.trait_name;

-- name: ListGenesForGecko :many
SELECT
  gg.id, gg.gecko_id, gg.trait_id, gg.zygosity, gg.created_at,
  gd.trait_name, gd.trait_code, gd.is_dominant, gd.species_id,
  gd.inheritance_type, gd.super_form_name
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
WHERE gg.gecko_id = $1
ORDER BY gd.trait_name;

-- name: CreateGeckoGene :one
INSERT INTO gecko_genes (gecko_id, trait_id, zygosity)
VALUES ($1, $2, $3)
RETURNING id, gecko_id, trait_id, zygosity, created_at;

-- name: DeleteGenesForGecko :exec
DELETE FROM gecko_genes WHERE gecko_id = $1;
```

(Note: `DeleteGenesForGecko` already exists — keep it as-is.)

- [ ] **Step 3: Update `public.sql`** — add `trait_id`, `inheritance_type`, `super_form_name` to `ListPublicGenesByGeckoIDs`

Replace only the `ListPublicGenesByGeckoIDs` query in that file (keep the others unchanged):

```sql
-- name: ListPublicGenesByGeckoIDs :many
-- Used to compose morph labels for the list endpoint in one round trip.
SELECT
  gg.gecko_id,
  gd.id        AS trait_id,
  gd.trait_name,
  gd.trait_code,
  gg.zygosity,
  gd.inheritance_type,
  gd.super_form_name
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
WHERE gg.gecko_id = ANY($1::int[])
ORDER BY gg.gecko_id, gd.trait_name;
```

- [ ] **Step 4: Create `morph_combos.sql`**

```sql
-- name: ListMorphCombos :many
SELECT id, species_id, name, code, description, notes, example_photo_url,
       created_at, updated_at
FROM morph_combos
ORDER BY species_id, name;

-- name: ListMorphCombosBySpecies :many
SELECT id, species_id, name, code, description, notes, example_photo_url,
       created_at, updated_at
FROM morph_combos
WHERE species_id = $1
ORDER BY name;

-- name: GetMorphCombo :one
SELECT id, species_id, name, code, description, notes, example_photo_url,
       created_at, updated_at
FROM morph_combos
WHERE id = $1;

-- name: CreateMorphCombo :one
INSERT INTO morph_combos
  (species_id, name, code, description, notes, example_photo_url)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, species_id, name, code, description, notes, example_photo_url,
          created_at, updated_at;

-- name: UpdateMorphCombo :one
UPDATE morph_combos
SET name = $2, code = $3, description = $4, notes = $5,
    example_photo_url = $6, updated_at = NOW()
WHERE id = $1
RETURNING id, species_id, name, code, description, notes, example_photo_url,
          created_at, updated_at;

-- name: DeleteMorphCombo :exec
DELETE FROM morph_combos WHERE id = $1;

-- name: InsertMorphComboTrait :exec
INSERT INTO morph_combo_traits (combo_id, trait_id, required_zygosity)
VALUES ($1, $2, $3);

-- name: DeleteMorphComboTraits :exec
DELETE FROM morph_combo_traits WHERE combo_id = $1;

-- name: ListMorphComboTraits :many
SELECT mct.combo_id, mct.trait_id, mct.required_zygosity,
       gd.trait_name, gd.trait_code
FROM morph_combo_traits mct
JOIN genetic_dictionary gd ON gd.id = mct.trait_id
WHERE mct.combo_id = ANY($1::int[])
ORDER BY mct.combo_id, gd.trait_name;

-- name: ListAllMorphCombosWithTraits :many
-- Bulk load for DetectMorph — one round trip fetches the full combo catalog.
SELECT
  mc.id        AS combo_id,
  mc.name      AS combo_name,
  mc.species_id,
  mct.trait_id,
  mct.required_zygosity
FROM morph_combos mc
JOIN morph_combo_traits mct ON mct.combo_id = mc.id
ORDER BY mc.species_id, mc.id, mct.trait_id;
```

- [ ] **Step 5: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/queries/
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(sqlc): genetics + morph combo queries"
```

---

## Task 3: sqlc Codegen

**Files:**
- Generate: `backend/internal/db/` (all generated files)

- [ ] **Step 1: Run codegen**

```bash
cd backend && sqlc generate
```

Expected: no errors. New/updated files:
- `internal/db/models.go` — gains `InheritanceType` enum + `MorphCombo`, `MorphComboTrait` structs
- `internal/db/genetic_dictionary.sql.go`
- `internal/db/gecko_genes.sql.go`
- `internal/db/public.sql.go`
- `internal/db/morph_combos.sql.go`
- `internal/db/querier.go`

- [ ] **Step 2: Verify generated enum constant names**

```bash
grep -n 'InheritanceType' backend/internal/db/models.go
```

Expected constants: `InheritanceTypeRECESSIVE`, `InheritanceTypeCODOMINANT`, `InheritanceTypeDOMINANT`, `InheritanceTypePOLYGENIC`

Note: sqlc strips underscores from enum values when building constant names. `CO_DOMINANT` → `CODOMINANT`. Confirm the exact names here before proceeding to Tasks 4–8.

- [ ] **Step 3: Build to confirm no compile errors**

```bash
cd backend && go build ./...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/db/
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "chore(sqlc): regenerate db package"
```

---

## Task 4: Morph Detection Package

**Files:**
- Create: `backend/internal/morph/detect.go`
- Create: `backend/internal/morph/detect_test.go`

- [ ] **Step 1: Write failing tests first**

Create `backend/internal/morph/detect_test.go`:

```go
package morph_test

import (
	"testing"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
	"github.com/jxnhoongz/project_gekko/backend/internal/morph"
)

const (
	idTremper   = int32(1)
	idEclipse   = int32(2)
	idBlizzard  = int32(3)
	idMackSnow  = int32(4)
	idRainwater = int32(5)
	idBell      = int32(6)
)

var testCombos = []morph.Combo{
	{
		Name: "Diablo Blanco",
		Requirements: []morph.ComboRequirement{
			{TraitID: idTremper, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idBlizzard, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idEclipse, RequiredZygosity: db.ZygosityHOM},
		},
	},
	{
		Name: "Raptor",
		Requirements: []morph.ComboRequirement{
			{TraitID: idTremper, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idEclipse, RequiredZygosity: db.ZygosityHOM},
		},
	},
	{
		Name: "Firewater",
		Requirements: []morph.ComboRequirement{
			{TraitID: idRainwater, RequiredZygosity: db.ZygosityHOM},
			{TraitID: idEclipse, RequiredZygosity: db.ZygosityHOM},
		},
	},
}

func g(id int32, name string, itype db.InheritanceType, zyg db.Zygosity, super string) morph.GeneRow {
	return morph.GeneRow{TraitID: id, TraitName: name, InheritanceType: itype, Zygosity: zyg, SuperFormName: super}
}

func TestDetect_NoTraits_ReturnsNormal(t *testing.T) {
	if got := morph.Detect(nil, testCombos); got != "Normal" {
		t.Fatalf("want Normal, got %q", got)
	}
}

func TestDetect_Raptor(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idEclipse, "Eclipse", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "Raptor" {
		t.Fatalf("want Raptor, got %q", got)
	}
}

// Critical: longest-match-first must pick Diablo Blanco over Raptor+leftover.
func TestDetect_DiabloBlanco_NotRaptorBlizzard(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idEclipse, "Eclipse", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idBlizzard, "Blizzard", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "Diablo Blanco" {
		t.Fatalf("want 'Diablo Blanco', got %q", got)
	}
}

func TestDetect_CoDominantHOM_UsesSuperFormName(t *testing.T) {
	genes := []morph.GeneRow{
		g(idMackSnow, "Mack Snow", db.InheritanceTypeCODOMINANT, db.ZygosityHOM, "Super Snow"),
	}
	if got := morph.Detect(genes, testCombos); got != "Super Snow" {
		t.Fatalf("want 'Super Snow', got %q", got)
	}
}

func TestDetect_RaptorHetMackSnow(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idEclipse, "Eclipse", db.InheritanceTypeRECESSIVE, db.ZygosityHOM, ""),
		g(idMackSnow, "Mack Snow", db.InheritanceTypeCODOMINANT, db.ZygosityHET, "Super Snow"),
	}
	if got := morph.Detect(genes, testCombos); got != "Raptor het Mack Snow" {
		t.Fatalf("want 'Raptor het Mack Snow', got %q", got)
	}
}

func TestDetect_PossHet(t *testing.T) {
	genes := []morph.GeneRow{
		g(idTremper, "Tremper Albino", db.InheritanceTypeRECESSIVE, db.ZygosityPOSSHET, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "poss. het Tremper Albino" {
		t.Fatalf("want 'poss. het Tremper Albino', got %q", got)
	}
}

func TestDetect_PolygeniHOM_UsesTraitName(t *testing.T) {
	genes := []morph.GeneRow{
		g(7, "Tangerine", db.InheritanceTypePOLYGENIC, db.ZygosityHOM, ""),
	}
	if got := morph.Detect(genes, testCombos); got != "Tangerine" {
		t.Fatalf("want 'Tangerine', got %q", got)
	}
}
```

Note: `InheritanceTypeCODOMINANT` is the sqlc-generated constant for `CO_DOMINANT`. Confirm in Task 3 output.

- [ ] **Step 2: Run tests to see them fail**

```bash
cd backend && go test ./internal/morph/... -v
```

Expected: compile error (package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

Create `backend/internal/morph/detect.go`:

```go
package morph

import (
	"sort"
	"strings"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// GeneRow is the minimal view of a gecko_genes+genetic_dictionary join row
// needed for morph detection. Callers convert DB-specific rows into this type.
type GeneRow struct {
	TraitID         int32
	TraitName       string
	InheritanceType db.InheritanceType
	Zygosity        db.Zygosity
	SuperFormName   string
}

// ComboRequirement describes one trait required by a combo.
type ComboRequirement struct {
	TraitID          int32
	RequiredZygosity db.Zygosity
}

// Combo describes one named morph combination.
type Combo struct {
	Name         string
	Requirements []ComboRequirement
}

// zygosityRank lets us compare zygosity: HOM(2) > HET(1) > POSS_HET(0).
var zygosityRank = map[db.Zygosity]int{
	db.ZygosityHOM:     2,
	db.ZygosityHET:     1,
	db.ZygosityPOSSHET: 0,
}

// Detect labels a gecko from its gene rows and the full combo catalog.
// It sorts combos longest-first so longer combos win over their subsets
// (Diablo Blanco beats Raptor+leftover-Blizzard).
// Once a combo matches, its traits are "covered" and cannot be claimed
// by a shorter combo in the same pass.
func Detect(genes []GeneRow, combos []Combo) string {
	// Sort combos: longest requirements first.
	sorted := make([]Combo, len(combos))
	copy(sorted, combos)
	sort.SliceStable(sorted, func(i, j int) bool {
		return len(sorted[i].Requirements) > len(sorted[j].Requirements)
	})

	// Build a fast lookup: traitID → zygosity.
	zygByTrait := make(map[int32]db.Zygosity, len(genes))
	for _, g := range genes {
		zygByTrait[g.TraitID] = g.Zygosity
	}

	var matchedNames []string
	covered := make(map[int32]bool)

	for _, combo := range sorted {
		// Skip if any required trait is already covered by a longer combo.
		skip := false
		for _, req := range combo.Requirements {
			if covered[req.TraitID] {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Check all requirements satisfied.
		ok := true
		for _, req := range combo.Requirements {
			actual, has := zygByTrait[req.TraitID]
			if !has || zygosityRank[actual] < zygosityRank[req.RequiredZygosity] {
				ok = false
				break
			}
		}
		if ok {
			matchedNames = append(matchedNames, combo.Name)
			for _, req := range combo.Requirements {
				covered[req.TraitID] = true
			}
		}
	}

	// Remaining uncovered traits.
	var remaining []string
	for _, g := range genes {
		if covered[g.TraitID] {
			continue
		}
		switch {
		case g.InheritanceType == db.InheritanceTypeCODOMINANT && g.Zygosity == db.ZygosityHOM:
			if g.SuperFormName != "" {
				remaining = append(remaining, g.SuperFormName)
			} else {
				remaining = append(remaining, g.TraitName)
			}
		case g.Zygosity == db.ZygosityHET:
			remaining = append(remaining, "het "+g.TraitName)
		case g.Zygosity == db.ZygosityPOSSHET:
			remaining = append(remaining, "poss. het "+g.TraitName)
		default:
			remaining = append(remaining, g.TraitName)
		}
	}

	parts := append(matchedNames, remaining...)
	if len(parts) == 0 {
		return "Normal"
	}
	return strings.Join(parts, " ")
}
```

Note: replace `InheritanceTypeCODOMINANT` with the actual constant from Task 3 if it differs.

- [ ] **Step 4: Run tests — expect all pass**

```bash
cd backend && go test ./internal/morph/... -v
```

Expected: 7 tests PASS.

- [ ] **Step 5: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/morph/
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(morph): Detect pure function + unit tests"
```

---

## Task 5: Update geckos.go

**Files:**
- Modify: `backend/internal/http/geckos.go`

Changes:
1. `geckoGeneDTO` — add `InheritanceType` + `SuperFormName`
2. `traitDTO` — add all 4 new columns (for `/api/traits` schema endpoint)
3. `geckoDTO` — add `MorphLabel string`
4. `listGeckos` — load combos once, compute morph_label per gecko
5. `getGecko` — load combos, compute morph_label
6. Import `morph` package

- [ ] **Step 1: Write failing build check**

```bash
cd backend && go build ./... 2>&1 | head -5
```

Should pass (no errors yet). We'll break it in the next step then fix.

- [ ] **Step 2: Update DTOs and handlers**

In `backend/internal/http/geckos.go`, make the following changes:

**a) Add import:**

Add to the import block:
```go
"github.com/jxnhoongz/project_gekko/backend/internal/morph"
```

**b) Update `traitDTO`** (add new fields after `IsDominant`):

```go
type traitDTO struct {
	ID              int32  `json:"id"`
	SpeciesID       int32  `json:"species_id"`
	TraitName       string `json:"trait_name"`
	TraitCode       string `json:"trait_code"`
	Description     string `json:"description"`
	IsDominant      bool   `json:"is_dominant"`
	InheritanceType string `json:"inheritance_type"`
	SuperFormName   string `json:"super_form_name"`
	ExamplePhotoUrl string `json:"example_photo_url"`
	Notes           string `json:"notes"`
}
```

**c) Update `geckoGeneDTO`** (add `InheritanceType` + `SuperFormName`):

```go
type geckoGeneDTO struct {
	TraitID         int32  `json:"trait_id"`
	TraitName       string `json:"trait_name"`
	TraitCode       string `json:"trait_code"`
	Zygosity        string `json:"zygosity"`
	IsDominant      bool   `json:"is_dominant"`
	InheritanceType string `json:"inheritance_type"`
	SuperFormName   string `json:"super_form_name"`
}
```

**d) Update `geckoDTO`** — add `MorphLabel` after `Notes`:

```go
type geckoDTO struct {
	ID            int32          `json:"id"`
	Code          string         `json:"code"`
	Name          string         `json:"name"`
	SpeciesID     int32          `json:"species_id"`
	SpeciesCode   string         `json:"species_code"`
	SpeciesName   string         `json:"species_name"`
	Sex           string         `json:"sex"`
	HatchDate     *string        `json:"hatch_date"`
	AcquiredDate  *string        `json:"acquired_date"`
	Status        string         `json:"status"`
	SireID        *int32         `json:"sire_id"`
	DamID         *int32         `json:"dam_id"`
	Notes         string         `json:"notes"`
	MorphLabel    string         `json:"morph_label"`
	CreatedAt     time.Time      `json:"created_at"`
	Traits        []geckoGeneDTO `json:"traits"`
	CoverPhotoUrl *string        `json:"cover_photo_url"`
	Photos        []mediaDTO     `json:"photos,omitempty"`
}
```

**e) Add `loadCombos` package-level function** (place before `listGeckos`):

```go
// loadCombos fetches the full morph combo catalog in one round trip.
// Both geckos.go and public.go call this — defined here as it lives in
// the same package.
func loadCombos(ctx context.Context, q *db.Queries) ([]morph.Combo, error) {
	rows, err := q.ListAllMorphCombosWithTraits(ctx)
	if err != nil {
		return nil, err
	}
	// Group by combo_id preserving order of first appearance.
	type entry struct {
		idx   int
		combo morph.Combo
	}
	byID := map[int32]*entry{}
	order := []int32{}
	for _, r := range rows {
		if _, ok := byID[r.ComboID]; !ok {
			byID[r.ComboID] = &entry{idx: len(order), combo: morph.Combo{Name: r.ComboName}}
			order = append(order, r.ComboID)
		}
		byID[r.ComboID].combo.Requirements = append(byID[r.ComboID].combo.Requirements, morph.ComboRequirement{
			TraitID:          r.TraitID,
			RequiredZygosity: r.RequiredZygosity,
		})
	}
	out := make([]morph.Combo, len(order))
	for _, id := range order {
		e := byID[id]
		out[e.idx] = e.combo
	}
	return out, nil
}
```

**f) Add `toMorphGeneRow` helper** (converts `ListGeckoGenesRow` / `ListGenesForGeckoRow` to `morph.GeneRow`):

```go
func listGeckoGenesToMorphRows(genes []db.ListGeckoGenesRow) map[int32][]morph.GeneRow {
	out := map[int32][]morph.GeneRow{}
	for _, g := range genes {
		out[g.GeckoID] = append(out[g.GeckoID], morph.GeneRow{
			TraitID:         g.TraitID,
			TraitName:       g.TraitName,
			InheritanceType: g.InheritanceType,
			Zygosity:        g.Zygosity,
			SuperFormName:   textOrEmpty(g.SuperFormName),
		})
	}
	return out
}

func listGenesForGeckoToMorphRows(genes []db.ListGenesForGeckoRow) []morph.GeneRow {
	out := make([]morph.GeneRow, 0, len(genes))
	for _, g := range genes {
		out = append(out, morph.GeneRow{
			TraitID:         g.TraitID,
			TraitName:       g.TraitName,
			InheritanceType: g.InheritanceType,
			Zygosity:        g.Zygosity,
			SuperFormName:   textOrEmpty(g.SuperFormName),
		})
	}
	return out
}
```

**g) Update `listGeckos`** — load combos at top, use morph.Detect per gecko:

Replace the section after `allGenes` grouping (around `genesByGecko`) with this pattern:

```go
// Preload combos once for this batch.
combos, err := loadCombos(ctx, d.q)
if err != nil {
    writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list combos failed"})
    return
}
morphRowsByGecko := listGeckoGenesToMorphRows(allGenes)
```

And in the `out = append(out, geckoDTO{...})` loop, add:

```go
MorphLabel: morph.Detect(morphRowsByGecko[g.ID], combos),
```

Also update the `geckoGeneDTO` mapping in `genesByGecko` loop to include new fields:

```go
genesByGecko[g.GeckoID] = append(genesByGecko[g.GeckoID], geckoGeneDTO{
    TraitID:         g.TraitID,
    TraitName:       g.TraitName,
    TraitCode:       textOrEmpty(g.TraitCode),
    Zygosity:        string(g.Zygosity),
    IsDominant:      g.IsDominant,
    InheritanceType: string(g.InheritanceType),
    SuperFormName:   textOrEmpty(g.SuperFormName),
})
```

**h) Update `getGecko`** — load combos, compute morph_label:

After fetching `genes`, add:

```go
combos, err := loadCombos(ctx, d.q)
if err != nil {
    writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list combos failed"})
    return
}
morphRows := listGenesForGeckoToMorphRows(genes)
```

Update the `geckoGeneDTO` mapping to include new fields (same as listGeckos above).

In the `out := geckoDTO{...}` literal, add:

```go
MorphLabel: morph.Detect(morphRows, combos),
```

**i) Update `listTraits`** — map new traitDTO fields:

```go
out = append(out, traitDTO{
    ID:              t.ID,
    SpeciesID:       t.SpeciesID,
    TraitName:       t.TraitName,
    TraitCode:       textOrEmpty(t.TraitCode),
    Description:     textOrEmpty(t.Description),
    IsDominant:      t.IsDominant,
    InheritanceType: string(t.InheritanceType),
    SuperFormName:   textOrEmpty(t.SuperFormName),
    ExamplePhotoUrl: textOrEmpty(t.ExamplePhotoUrl),
    Notes:           textOrEmpty(t.Notes),
})
```

- [ ] **Step 3: Build**

```bash
cd backend && go build ./...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/http/geckos.go
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(api): morph_label + inheritance_type on gecko responses"
```

---

## Task 6: Update public.go

**Files:**
- Modify: `backend/internal/http/public.go`

- [ ] **Step 1: Replace `composePublicMorph` with `morph.Detect`**

In `backend/internal/http/public.go`:

**a) Add import:** `"github.com/jxnhoongz/project_gekko/backend/internal/morph"`

**b) Delete the `composePublicMorph` function entirely** (lines ~254–291).

**c) In `listAvailable`**: after `genesByGecko` is built, add:

```go
combos, err := loadCombos(ctx, d.q)
if err != nil {
    writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list combos failed"})
    return
}
```

Then convert `genesByGecko[g.ID]` (type `[]db.ListPublicGenesByGeckoIDsRow`) to `[]morph.GeneRow` for `morph.Detect`. Replace the `Morph: composePublicMorph(genesByGecko[g.ID])` call with:

```go
Morph: morph.Detect(publicGenesToMorphRows(genesByGecko[g.ID]), combos),
```

**d) In `getByCode`**: same pattern — load combos, convert, call `morph.Detect`.

**e) Add helper** (place near the end of public.go, before `coverPtr`):

```go
func publicGenesToMorphRows(rows []db.ListPublicGenesByGeckoIDsRow) []morph.GeneRow {
	out := make([]morph.GeneRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, morph.GeneRow{
			TraitID:         r.TraitID,
			TraitName:       r.TraitName,
			InheritanceType: r.InheritanceType,
			Zygosity:        r.Zygosity,
			SuperFormName:   textOrEmpty(r.SuperFormName),
		})
	}
	return out
}
```

- [ ] **Step 2: Build**

```bash
cd backend && go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/http/public.go
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(public): replace composePublicMorph with morph.Detect"
```

---

## Task 7: morphcombos.go Handler + Mount

**Files:**
- Create: `backend/internal/http/morphcombos.go`
- Modify: `backend/cmd/gekko/main.go`

- [ ] **Step 1: Create `morphcombos.go`**

```go
package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountMorphCombos registers admin-only CRUD for morph combos.
func MountMorphCombos(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &morphCombosDeps{pool: pool, q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/morph-combos", d.list)
		pr.Post("/api/morph-combos", d.create)
		pr.Get("/api/morph-combos/{id}", d.get)
		pr.Patch("/api/morph-combos/{id}", d.update)
		pr.Delete("/api/morph-combos/{id}", d.delete)
	})
}

type morphCombosDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// ---- DTOs ----

type morphComboTraitDTO struct {
	TraitID          int32  `json:"trait_id"`
	TraitName        string `json:"trait_name"`
	TraitCode        string `json:"trait_code"`
	RequiredZygosity string `json:"required_zygosity"`
}

type morphComboDTO struct {
	ID              int32                `json:"id"`
	SpeciesID       int32                `json:"species_id"`
	Name            string               `json:"name"`
	Code            string               `json:"code"`
	Description     string               `json:"description"`
	Notes           string               `json:"notes"`
	ExamplePhotoUrl string               `json:"example_photo_url"`
	Requirements    []morphComboTraitDTO `json:"requirements"`
}

type morphCombosListResp struct {
	Combos []morphComboDTO `json:"combos"`
	Total  int             `json:"total"`
}

// ---- requests ----

type morphComboTraitInput struct {
	TraitID          int32  `json:"trait_id"`
	RequiredZygosity string `json:"required_zygosity"`
}

type createMorphComboReq struct {
	SpeciesID       int32                  `json:"species_id"`
	Name            string                 `json:"name"`
	Code            string                 `json:"code"`
	Description     string                 `json:"description"`
	Notes           string                 `json:"notes"`
	ExamplePhotoUrl string                 `json:"example_photo_url"`
	Requirements    []morphComboTraitInput `json:"requirements"`
}

type updateMorphComboReq = createMorphComboReq

// ---- handlers ----

func (d *morphCombosDeps) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var rows []db.MorphCombo
	var err error

	if sc := r.URL.Query().Get("species_code"); sc != "" {
		var speciesID int32
		if err2 := d.pool.QueryRow(ctx,
			"SELECT id FROM species WHERE code = $1", sc).Scan(&speciesID); err2 != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown species_code"})
			return
		}
		rows, err = d.q.ListMorphCombosBySpecies(ctx, speciesID)
	} else {
		rows, err = d.q.ListMorphCombos(ctx)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}

	// Preload all requirements in one query.
	ids := make([]int32, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	traitRows, err := d.q.ListMorphComboTraits(ctx, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list traits failed"})
		return
	}
	reqsByCombo := map[int32][]morphComboTraitDTO{}
	for _, t := range traitRows {
		reqsByCombo[t.ComboID] = append(reqsByCombo[t.ComboID], morphComboTraitDTO{
			TraitID:          t.TraitID,
			TraitName:        t.TraitName,
			TraitCode:        textOrEmpty(t.TraitCode),
			RequiredZygosity: string(t.RequiredZygosity),
		})
	}

	out := make([]morphComboDTO, 0, len(rows))
	for _, mc := range rows {
		reqs := reqsByCombo[mc.ID]
		if reqs == nil {
			reqs = []morphComboTraitDTO{}
		}
		out = append(out, morphComboDTO{
			ID:              mc.ID,
			SpeciesID:       mc.SpeciesID,
			Name:            mc.Name,
			Code:            textOrEmpty(mc.Code),
			Description:     textOrEmpty(mc.Description),
			Notes:           textOrEmpty(mc.Notes),
			ExamplePhotoUrl: textOrEmpty(mc.ExamplePhotoUrl),
			Requirements:    reqs,
		})
	}
	writeJSON(w, http.StatusOK, morphCombosListResp{Combos: out, Total: len(out)})
}

func (d *morphCombosDeps) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ctx := r.Context()
	mc, err := d.q.GetMorphCombo(ctx, int32(id))
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}
	traitRows, err := d.q.ListMorphComboTraits(ctx, []int32{mc.ID})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch traits failed"})
		return
	}
	reqs := make([]morphComboTraitDTO, 0, len(traitRows))
	for _, t := range traitRows {
		reqs = append(reqs, morphComboTraitDTO{
			TraitID:          t.TraitID,
			TraitName:        t.TraitName,
			TraitCode:        textOrEmpty(t.TraitCode),
			RequiredZygosity: string(t.RequiredZygosity),
		})
	}
	writeJSON(w, http.StatusOK, morphComboDTO{
		ID: mc.ID, SpeciesID: mc.SpeciesID, Name: mc.Name,
		Code: textOrEmpty(mc.Code), Description: textOrEmpty(mc.Description),
		Notes: textOrEmpty(mc.Notes), ExamplePhotoUrl: textOrEmpty(mc.ExamplePhotoUrl),
		Requirements: reqs,
	})
}

func (d *morphCombosDeps) create(w http.ResponseWriter, r *http.Request) {
	var req createMorphComboReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.SpeciesID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and species_id required"})
		return
	}
	ctx := r.Context()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	mc, err := qtx.CreateMorphCombo(ctx, db.CreateMorphComboParams{
		SpeciesID:       req.SpeciesID,
		Name:            req.Name,
		Code:            pgText(req.Code),
		Description:     pgText(req.Description),
		Notes:           pgText(req.Notes),
		ExamplePhotoUrl: pgText(req.ExamplePhotoUrl),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create failed: " + err.Error()})
		return
	}
	if err := applyComboTraits(ctx, qtx, mc.ID, req.Requirements); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}
	// Re-fetch to return full response (with trait names).
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(mc.ID)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	d.get(w, r2)
}

func (d *morphCombosDeps) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req updateMorphComboReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	ctx := r.Context()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	mc, err := qtx.UpdateMorphCombo(ctx, db.UpdateMorphComboParams{
		ID:              int32(id),
		Name:            req.Name,
		Code:            pgText(req.Code),
		Description:     pgText(req.Description),
		Notes:           pgText(req.Notes),
		ExamplePhotoUrl: pgText(req.ExamplePhotoUrl),
	})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found or update failed"})
		return
	}
	if err := qtx.DeleteMorphComboTraits(ctx, mc.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clear traits failed"})
		return
	}
	if err := applyComboTraits(ctx, qtx, mc.ID, req.Requirements); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(mc.ID)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	d.get(w, r2)
}

func (d *morphCombosDeps) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := d.q.DeleteMorphCombo(r.Context(), int32(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyComboTraits inserts junction rows; validates zygosity.
func applyComboTraits(ctx context.Context, q *db.Queries, comboID int32, reqs []morphComboTraitInput) error {
	for _, req := range reqs {
		zyg, ok := validZygosity[req.RequiredZygosity]
		if !ok {
			return fmt.Errorf("invalid required_zygosity %q", req.RequiredZygosity)
		}
		if err := q.InsertMorphComboTrait(ctx, db.InsertMorphComboTraitParams{
			ComboID:          comboID,
			TraitID:          req.TraitID,
			RequiredZygosity: zyg,
		}); err != nil {
			return fmt.Errorf("insert trait %d: %w", req.TraitID, err)
		}
	}
	return nil
}
```

Note: `applyComboTraits` uses `validZygosity` which is defined in `geckos.go` (same package). Add `"fmt"` to the import block.

- [ ] **Step 2: Mount in main.go**

In `backend/cmd/gekko/main.go`, add after `apihttp.MountListings(...)`:

```go
apihttp.MountMorphCombos(r, pool, signer)
```

- [ ] **Step 3: Build**

```bash
cd backend && go build ./...
```

Expected: exits 0.

- [ ] **Step 4: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/http/morphcombos.go backend/cmd/gekko/main.go
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(api): MountMorphCombos CRUD handler"
```

---

## Task 8: Morph Combos Integration Tests

**Files:**
- Create: `backend/internal/http/morphcombos_test.go`

- [ ] **Step 1: Write the tests**

```go
package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func morphCombosSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := fmt.Sprintf("morphcombos+%d@example.com", time.Now().UnixNano())
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
	MountMorphCombos(r, pool, signer)
	return r, tok, pool
}

func TestMorphCombos_CRUD(t *testing.T) {
	handler, tok, pool := morphCombosSetup(t)

	// Resolve LP species ID.
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM species WHERE code = 'LP'").Scan(&speciesID))

	// Resolve two existing trait IDs (Tremper Albino + Eclipse).
	var tremperID, eclipseID int32
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM genetic_dictionary WHERE species_id = $1 AND trait_name = 'Tremper Albino'",
		speciesID).Scan(&tremperID))
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM genetic_dictionary WHERE species_id = $1 AND trait_name = 'Eclipse'",
		speciesID).Scan(&eclipseID))

	body, _ := json.Marshal(map[string]any{
		"species_id":  speciesID,
		"name":        fmt.Sprintf("Test Combo %d", time.Now().UnixNano()),
		"code":        "TCOMBO",
		"description": "test",
		"requirements": []map[string]any{
			{"trait_id": tremperID, "required_zygosity": "HOM"},
			{"trait_id": eclipseID, "required_zygosity": "HOM"},
		},
	})

	// Create
	req := httptest.NewRequest(http.MethodPost, "/api/morph-combos", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var created morphComboDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.Equal(t, 2, len(created.Requirements))
	comboID := created.ID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM morph_combos WHERE id = $1", comboID)
	})

	// Get
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/morph-combos/%d", comboID), nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", fmt.Sprintf("%d", comboID))
	req2 = req2.WithContext(context.WithValue(req2.Context(), chi.RouteCtxKey, rctx))
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// Update — remove one requirement
	upd, _ := json.Marshal(map[string]any{
		"species_id":   speciesID,
		"name":         created.Name,
		"code":         "TCOMBO2",
		"requirements": []map[string]any{{"trait_id": tremperID, "required_zygosity": "HOM"}},
	})
	req3 := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/morph-combos/%d", comboID), bytes.NewReader(upd))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	req3 = req3.WithContext(context.WithValue(req3.Context(), chi.RouteCtxKey, rctx))
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	var updated morphComboDTO
	require.NoError(t, json.NewDecoder(w3.Body).Decode(&updated))
	assert.Equal(t, 1, len(updated.Requirements))

	// Delete
	req4 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/morph-combos/%d", comboID), nil)
	req4.Header.Set("Authorization", "Bearer "+tok)
	req4 = req4.WithContext(context.WithValue(req4.Context(), chi.RouteCtxKey, rctx))
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusNoContent, w4.Code)
}

func TestMorphCombos_List_BySpeciesCode(t *testing.T) {
	handler, tok, _ := morphCombosSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/morph-combos?species_code=LP", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp morphCombosListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.GreaterOrEqual(t, len(resp.Combos), 5) // at least the 5 seeded combos
}
```

- [ ] **Step 2: Run tests**

```bash
cd backend && go test ./internal/http/... -run TestMorphCombos -v
```

Expected: both tests PASS.

- [ ] **Step 3: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add backend/internal/http/morphcombos_test.go
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "test(api): morph combos integration tests"
```

---

## Task 9: Frontend Types + Composable

**Files:**
- Create: `apps/admin/src/types/morph.ts`
- Modify: `apps/admin/src/types/gecko.ts`
- Create: `apps/admin/src/composables/useMorphCombos.ts`

- [ ] **Step 1: Create `types/morph.ts`**

```typescript
export type InheritanceType = 'RECESSIVE' | 'CO_DOMINANT' | 'DOMINANT' | 'POLYGENIC';

export interface MorphComboTrait {
  trait_id: number;
  trait_name: string;
  trait_code: string;
  required_zygosity: 'HOM' | 'HET' | 'POSS_HET';
}

export interface MorphCombo {
  id: number;
  species_id: number;
  name: string;
  code: string;
  description: string;
  notes: string;
  example_photo_url: string;
  requirements: MorphComboTrait[];
}

export interface MorphCombosListResponse {
  combos: MorphCombo[];
  total: number;
}

export interface MorphComboTraitInput {
  trait_id: number;
  required_zygosity: 'HOM' | 'HET' | 'POSS_HET';
}

export interface MorphComboWritePayload {
  species_id: number;
  name: string;
  code: string;
  description: string;
  notes: string;
  example_photo_url: string;
  requirements: MorphComboTraitInput[];
}

export const INHERITANCE_TYPE_LABEL: Record<InheritanceType, string> = {
  RECESSIVE: 'Recessive',
  CO_DOMINANT: 'Co-Dominant',
  DOMINANT: 'Dominant',
  POLYGENIC: 'Polygenic',
};
```

- [ ] **Step 2: Update `types/gecko.ts`**

Make these changes:

**a) Remove** the `morphFromTraits` function and its JSDoc comment (lines ~64–79).

**b) Add import** at the top:
```typescript
import type { InheritanceType } from './morph';
export type { InheritanceType };
```

**c) Update `Trait` interface** — add 4 new fields:
```typescript
export interface Trait {
  id: number;
  species_id: number;
  trait_name: string;
  trait_code: string;
  description: string;
  is_dominant: boolean;
  inheritance_type: InheritanceType;
  super_form_name: string;
  example_photo_url: string;
  notes: string;
}
```

**d) Update `GeckoTrait` interface** — add 2 new fields:
```typescript
export interface GeckoTrait {
  trait_id: number;
  trait_name: string;
  trait_code: string;
  zygosity: Zygosity;
  is_dominant: boolean;
  inheritance_type: InheritanceType;
  super_form_name: string;
}
```

**e) Update `Gecko` interface** — add `morph_label`:
```typescript
export interface Gecko {
  id: number;
  code: string;
  name: string;
  species_id: number;
  species_code: string;
  species_name: string;
  sex: Sex;
  hatch_date: string | null;
  acquired_date: string | null;
  status: GeckoStatus;
  sire_id: number | null;
  dam_id: number | null;
  notes: string;
  morph_label: string;
  created_at: string;
  traits: GeckoTrait[];
  cover_photo_url: string | null;
  photos?: GeckoPhoto[];
}
```

- [ ] **Step 3: Create `composables/useMorphCombos.ts`**

```typescript
import { useQuery, useMutation, useQueryClient } from '@tanstack/vue-query';
import type { MaybeRef } from 'vue';
import { unref } from 'vue';
import { api } from '@/lib/api';
import type {
  MorphCombo,
  MorphCombosListResponse,
  MorphComboWritePayload,
} from '@/types/morph';

export const morphComboKeys = {
  all: ['morph-combos'] as const,
  list: (speciesCode?: string) =>
    [...morphComboKeys.all, 'list', speciesCode ?? ''] as const,
  detail: (id: number | string) =>
    [...morphComboKeys.all, 'detail', id] as const,
};

export function useMorphCombos(speciesCode?: MaybeRef<string>) {
  return useQuery({
    queryKey: morphComboKeys.list(unref(speciesCode)),
    queryFn: async () => {
      const params = unref(speciesCode)
        ? { species_code: unref(speciesCode) }
        : undefined;
      const { data } = await api.get<MorphCombosListResponse>(
        '/api/morph-combos',
        { params },
      );
      return data;
    },
    staleTime: 60_000,
  });
}

export function useMorphCombo(id: MaybeRef<number | string | null>) {
  return useQuery({
    queryKey: morphComboKeys.detail(unref(id) ?? 0),
    queryFn: async () => {
      const v = unref(id);
      if (!v) throw new Error('no id');
      const { data } = await api.get<MorphCombo>(`/api/morph-combos/${v}`);
      return data;
    },
    enabled: () => !!unref(id),
    staleTime: 60_000,
  });
}

function invalidateMorphCombos(
  qc: ReturnType<typeof useQueryClient>,
  id?: number,
) {
  qc.invalidateQueries({ queryKey: morphComboKeys.all });
  if (id !== undefined) {
    qc.invalidateQueries({ queryKey: morphComboKeys.detail(id) });
  }
}

export function useCreateMorphCombo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (payload: MorphComboWritePayload) => {
      const { data } = await api.post<MorphCombo>('/api/morph-combos', payload);
      return data;
    },
    onSuccess: (mc) => invalidateMorphCombos(qc, mc.id),
  });
}

export function useUpdateMorphCombo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({
      id,
      payload,
    }: {
      id: number;
      payload: MorphComboWritePayload;
    }) => {
      const { data } = await api.patch<MorphCombo>(
        `/api/morph-combos/${id}`,
        payload,
      );
      return data;
    },
    onSuccess: (mc) => invalidateMorphCombos(qc, mc.id),
  });
}

export function useDeleteMorphCombo() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => {
      await api.delete(`/api/morph-combos/${id}`);
      return id;
    },
    onSuccess: (id) => invalidateMorphCombos(qc, id),
  });
}
```

- [ ] **Step 4: Check TypeScript**

```bash
cd apps/admin && bunx tsc --noEmit 2>&1 | head -20
```

Expected: no errors (or only pre-existing unrelated errors).

- [ ] **Step 5: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add apps/admin/src/types/morph.ts \
     apps/admin/src/types/gecko.ts \
     apps/admin/src/composables/useMorphCombos.ts
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(admin): morph types + useMorphCombos composable"
```

---

## Task 10: MorphCombosView + MorphComboFormSheet + Nav/Route

**Files:**
- Create: `apps/admin/src/views/MorphCombosView.vue`
- Create: `apps/admin/src/components/MorphComboFormSheet.vue`
- Modify: `apps/admin/src/layouts/AppShell.vue`
- Modify: `apps/admin/src/router/index.ts`

- [ ] **Step 1: Create `MorphComboFormSheet.vue`**

```vue
<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import {
  DialogRoot,
  DialogPortal,
  DialogOverlay,
  DialogContent,
} from 'reka-ui';
import { X, Plus, Trash2 } from 'lucide-vue-next';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { useSpecies, useTraits } from '@/composables/useGeckos';
import {
  useCreateMorphCombo,
  useUpdateMorphCombo,
} from '@/composables/useMorphCombos';
import type { MorphCombo, MorphComboTraitInput } from '@/types/morph';

const props = defineProps<{
  open: boolean;
  combo: MorphCombo | null;
}>();
const emit = defineEmits<{
  'update:open': [value: boolean];
}>();

const { data: speciesData } = useSpecies();
const { data: traitsData } = useTraits();

const allTraits = computed(() => traitsData.value?.traits ?? []);

const form = ref({
  species_id: 0,
  name: '',
  code: '',
  description: '',
  notes: '',
  example_photo_url: '',
  requirements: [] as MorphComboTraitInput[],
});

const addTraitID = ref<number | null>(null);
const addZygosity = ref<'HOM' | 'HET' | 'POSS_HET'>('HOM');

watch(
  () => props.combo,
  (c) => {
    if (c) {
      form.value = {
        species_id: c.species_id,
        name: c.name,
        code: c.code,
        description: c.description,
        notes: c.notes,
        example_photo_url: c.example_photo_url,
        requirements: c.requirements.map((r) => ({
          trait_id: r.trait_id,
          required_zygosity: r.required_zygosity,
        })),
      };
    } else {
      form.value = {
        species_id:
          speciesData.value?.species?.find((s) => s.code === 'LP')?.id ?? 0,
        name: '',
        code: '',
        description: '',
        notes: '',
        example_photo_url: '',
        requirements: [],
      };
    }
  },
  { immediate: true },
);

const { mutate: createCombo, isPending: creating } = useCreateMorphCombo();
const { mutate: updateCombo, isPending: updating } = useUpdateMorphCombo();
const saving = computed(() => creating.value || updating.value);

function addRequirement() {
  if (!addTraitID.value) return;
  if (form.value.requirements.some((r) => r.trait_id === addTraitID.value))
    return;
  form.value.requirements.push({
    trait_id: addTraitID.value,
    required_zygosity: addZygosity.value,
  });
  addTraitID.value = null;
}

function removeRequirement(index: number) {
  form.value.requirements.splice(index, 1);
}

function traitNameFor(id: number) {
  return allTraits.value.find((t) => t.id === id)?.trait_name ?? `#${id}`;
}

function close() {
  emit('update:open', false);
}

function submit() {
  const payload = { ...form.value };
  if (props.combo) {
    updateCombo({ id: props.combo.id, payload }, { onSuccess: close });
  } else {
    createCombo(payload, { onSuccess: close });
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 bg-brand-dark-950/40 z-40" />
      <DialogContent
        class="fixed right-0 top-0 h-full w-full max-w-lg bg-brand-cream-50 border-l border-brand-cream-300 shadow-xl z-50 flex flex-col overflow-y-auto focus:outline-none"
      >
        <!-- Header -->
        <div
          class="flex items-center justify-between px-6 py-4 border-b border-brand-cream-300"
        >
          <h2 class="text-xl font-semibold text-brand-dark-950">
            {{ combo ? 'Edit Combo' : 'New Morph Combo' }}
          </h2>
          <button
            class="text-brand-dark-600 hover:text-brand-dark-950"
            @click="close"
          >
            <X class="w-5 h-5" />
          </button>
        </div>

        <!-- Body -->
        <div class="flex-1 px-6 py-6 space-y-5">
          <!-- Species -->
          <div class="space-y-1.5">
            <Label>Species</Label>
            <select
              v-model="form.species_id"
              :disabled="!!combo"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
            >
              <option
                v-for="s in speciesData?.species"
                :key="s.id"
                :value="s.id"
              >
                {{ s.common_name }}
              </option>
            </select>
          </div>

          <!-- Name + Code -->
          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <Label>Name <span class="text-destructive ml-0.5">*</span></Label>
              <Input v-model="form.name" placeholder="Raptor" />
            </div>
            <div class="space-y-1.5">
              <Label>Code</Label>
              <Input v-model="form.code" placeholder="RAPT" />
            </div>
          </div>

          <!-- Description -->
          <div class="space-y-1.5">
            <Label>Description</Label>
            <textarea
              v-model="form.description"
              rows="2"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500 resize-none"
            />
          </div>

          <!-- Notes (internal) -->
          <div class="space-y-1.5">
            <Label>Notes <span class="text-xs text-brand-dark-600">(internal)</span></Label>
            <textarea
              v-model="form.notes"
              rows="2"
              class="w-full rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500 resize-none"
            />
          </div>

          <!-- Requirements -->
          <div class="space-y-2">
            <Label>Required Traits</Label>
            <div
              v-if="form.requirements.length"
              class="space-y-1.5"
            >
              <div
                v-for="(req, i) in form.requirements"
                :key="req.trait_id"
                class="flex items-center justify-between bg-brand-cream-100 rounded-lg px-3 py-2"
              >
                <span class="text-sm font-medium text-brand-dark-950">
                  {{ traitNameFor(req.trait_id) }}
                </span>
                <div class="flex items-center gap-2">
                  <Badge variant="outline" class="text-xs">
                    {{ req.required_zygosity }}
                  </Badge>
                  <button
                    class="text-brand-dark-600 hover:text-destructive"
                    @click="removeRequirement(i)"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
            <p v-else class="text-sm text-brand-dark-600">No traits added yet.</p>

            <!-- Add trait row -->
            <div class="flex gap-2 mt-2">
              <select
                v-model="addTraitID"
                class="flex-1 rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
              >
                <option :value="null">Select trait…</option>
                <option
                  v-for="t in allTraits.filter(
                    (t) =>
                      t.species_id === form.species_id &&
                      !form.requirements.some((r) => r.trait_id === t.id),
                  )"
                  :key="t.id"
                  :value="t.id"
                >
                  {{ t.trait_name }}
                </option>
              </select>
              <select
                v-model="addZygosity"
                class="rounded-lg border border-brand-cream-400 bg-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-brand-gold-500"
              >
                <option value="HOM">HOM</option>
                <option value="HET">HET</option>
                <option value="POSS_HET">POSS HET</option>
              </select>
              <Button variant="outline" size="sm" @click="addRequirement">
                <Plus class="w-4 h-4" />
              </Button>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div
          class="px-6 py-4 border-t border-brand-cream-300 flex justify-end gap-3"
        >
          <Button variant="ghost" @click="close">Cancel</Button>
          <Button :disabled="saving || !form.name" @click="submit">
            {{ saving ? 'Saving…' : combo ? 'Save Changes' : 'Create Combo' }}
          </Button>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
```

- [ ] **Step 2: Create `MorphCombosView.vue`**

```vue
<script setup lang="ts">
import { ref } from 'vue';
import { Plus, Edit2, Trash2, Dna } from 'lucide-vue-next';
import { Card } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { useMorphCombos, useDeleteMorphCombo } from '@/composables/useMorphCombos';
import MorphComboFormSheet from '@/components/MorphComboFormSheet.vue';
import type { MorphCombo } from '@/types/morph';

const { data, isLoading } = useMorphCombos();
const { mutate: deleteCombo } = useDeleteMorphCombo();

const sheetOpen = ref(false);
const editing = ref<MorphCombo | null>(null);

function openCreate() {
  editing.value = null;
  sheetOpen.value = true;
}

function openEdit(combo: MorphCombo) {
  editing.value = combo;
  sheetOpen.value = true;
}

function confirmDelete(combo: MorphCombo) {
  if (confirm(`Delete "${combo.name}"?`)) {
    deleteCombo(combo.id);
  }
}
</script>

<template>
  <div class="px-4 sm:px-6 lg:px-8 py-8">
    <!-- Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-3xl font-display text-brand-dark-950">Morph Combos</h1>
        <p class="text-sm text-brand-dark-600 mt-1">
          Named combinations of base traits.
        </p>
      </div>
      <Button @click="openCreate">
        <Plus class="w-4 h-4 mr-2" />
        Add Combo
      </Button>
    </div>

    <!-- Loading -->
    <div v-if="isLoading" class="text-brand-dark-600 text-sm">Loading…</div>

    <!-- Empty -->
    <div
      v-else-if="!data?.combos?.length"
      class="text-center py-16 text-brand-dark-600"
    >
      <Dna class="w-10 h-10 mx-auto mb-3 text-brand-cream-400" />
      <p>No morph combos yet. Add one above.</p>
    </div>

    <!-- Grid -->
    <div
      v-else
      class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6"
    >
      <Card
        v-for="combo in data.combos"
        :key="combo.id"
        class="p-5 border-brand-cream-300 bg-brand-cream-50"
      >
        <div class="flex items-start justify-between mb-3">
          <div>
            <h3 class="font-semibold text-brand-dark-950">{{ combo.name }}</h3>
            <span
              v-if="combo.code"
              class="text-xs text-brand-dark-600 font-mono"
            >{{ combo.code }}</span>
          </div>
          <div class="flex gap-1 shrink-0 ml-2">
            <button
              class="p-1 text-brand-dark-600 hover:text-brand-dark-950"
              @click="openEdit(combo)"
            >
              <Edit2 class="w-4 h-4" />
            </button>
            <button
              class="p-1 text-brand-dark-600 hover:text-destructive"
              @click="confirmDelete(combo)"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- Trait badges -->
        <div class="flex flex-wrap gap-1.5">
          <Badge
            v-for="req in combo.requirements"
            :key="req.trait_id"
            variant="outline"
            class="text-xs"
          >
            {{ req.trait_name }}
            <span class="ml-1 text-brand-dark-600">{{ req.required_zygosity }}</span>
          </Badge>
        </div>

        <p
          v-if="combo.description"
          class="mt-3 text-xs text-brand-dark-600 line-clamp-2"
        >
          {{ combo.description }}
        </p>
      </Card>
    </div>

    <MorphComboFormSheet v-model:open="sheetOpen" :combo="editing" />
  </div>
</template>
```

- [ ] **Step 3: Add nav entry in AppShell.vue**

In `apps/admin/src/layouts/AppShell.vue`, find the nav array and add `Dna` to the lucide imports, then add the route:

```typescript
import { ..., Dna } from 'lucide-vue-next';
```

Add to the nav array (after Schema or in Schema section):

```typescript
{ name: 'morph-combos', label: 'Morph Combos', icon: Dna },
```

- [ ] **Step 4: Add route in router/index.ts**

```typescript
{ path: 'morph-combos', name: 'morph-combos', component: () => import('@/views/MorphCombosView.vue') },
```

Add it after the `schema` route.

- [ ] **Step 5: Verify dev server starts**

```bash
cd apps/admin && bun run dev &
```

Navigate to `/morph-combos` — should show the 5 seeded LP combos.

- [ ] **Step 6: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add apps/admin/src/views/MorphCombosView.vue \
     apps/admin/src/components/MorphComboFormSheet.vue \
     apps/admin/src/layouts/AppShell.vue \
     apps/admin/src/router/index.ts
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(admin): MorphCombosView + form sheet + nav route"
```

---

## Task 11: Update Gecko Displays

**Files:**
- Modify: `apps/admin/src/components/GeckoCard.vue`
- Modify: `apps/admin/src/views/GeckoDetailView.vue` (morph display only)

- [ ] **Step 1: Update GeckoCard.vue**

Remove `morphFromTraits` import and `displayMorph` computed. Use `props.gecko.morph_label` directly in the template.

**a) Remove from imports:**
```typescript
import { morphFromTraits, STATUS_LABEL } from '@/types/gecko';
```
Change to:
```typescript
import { STATUS_LABEL } from '@/types/gecko';
```

**b) Remove:**
```typescript
const displayMorph = computed(() => morphFromTraits(props.gecko.traits));
```

**c) In template**, replace `{{ displayMorph }}` with `{{ gecko.morph_label }}`.

- [ ] **Step 2: Update GeckoDetailView.vue morph display**

Search for any usage of `morphFromTraits` in `GeckoDetailView.vue`. Remove that import and replace with `gecko.morph_label` (the gecko detail already has the full DTO from `useGecko`).

- [ ] **Step 3: TypeScript check**

```bash
cd apps/admin && bunx tsc --noEmit 2>&1 | grep -i 'morphFromTraits\|morph_label' | head -10
```

Expected: no errors mentioning `morphFromTraits`.

- [ ] **Step 4: Commit**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  add apps/admin/src/components/GeckoCard.vue \
     apps/admin/src/views/GeckoDetailView.vue
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  commit -m "feat(admin): use morph_label from API, remove morphFromTraits"
```

---

## Task 12: Smoke Test + Push

- [ ] **Step 1: Run all backend tests**

```bash
cd backend && go test ./... -v 2>&1 | tail -20
```

Expected: all PASS, no FAIL.

- [ ] **Step 2: Start backend + visit admin**

```bash
cd backend && go run ./cmd/gekko/main.go &
cd apps/admin && bun run dev &
```

Open `http://localhost:5173`:

- Geckos list: morph column shows labels like "Raptor" or "Normal" (not raw trait names).
- Gecko detail: morph_label shown in hero card.
- `/morph-combos`: grid shows 5 seeded combos with trait badges.
- "Add Combo" opens sheet, create a combo, verify it appears.
- Edit a combo, verify changes persist.
- Delete the test combo.

- [ ] **Step 3: Check public API morph label**

```bash
curl -s http://localhost:8080/api/public/geckos | python3 -m json.tool | grep morph
```

Expected: `"morph":` field present with meaningful labels.

- [ ] **Step 4: Kill dev servers**

```bash
kill %1 %2 2>/dev/null || true
```

- [ ] **Step 5: Push**

```bash
git -c user.name=jxnhoongz -c user.email=vatanahan09@gmail.com \
  push origin main
```

---

## Self-Review Against Spec

| Spec requirement | Covered by |
|---|---|
| `inheritance_type` enum + 4 new columns on `genetic_dictionary` | Task 1 migration |
| `super_form_name` for CO_DOMINANT traits | Task 1 seed UPDATEs |
| `morph_combos` + `morph_combo_traits` tables | Task 1 migration |
| Raptor, Diablo Blanco, Firewater, Electric, Blazing Blizzard seeded | Task 1 seed INSERTs |
| Raptor + Super Snow removed from `genetic_dictionary` | Task 1 DELETE |
| New LP traits (Marble Eye, Giant, Lemon Frost, Carrot Tail, Baldy, BN) | Task 1 INSERT |
| `DetectMorph` pure function, longest-combo-first | Task 4 detect.go |
| Unit test: `[Tremper+Eclipse+Blizzard] == "Diablo Blanco"` not "Raptor Blizzard" | Task 4 test |
| `morph_label` on all gecko API responses (admin + public) | Tasks 5, 6 |
| `/api/morph-combos` CRUD admin endpoints | Task 7 |
| `inheritance_type` + `super_form_name` on `GET /api/traits` | Task 5 traitDTO |
| Frontend `MorphCombo` types + composables | Task 9 |
| Admin Morph Combos page (grid + sheet) | Task 10 |
| `morphFromTraits` removed from frontend | Tasks 9, 11 |
| `morph_label` used in GeckoCard + GeckoDetailView | Task 11 |
