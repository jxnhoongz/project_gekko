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

-- Unique code per species (partial — allows NULL codes).
CREATE UNIQUE INDEX morph_combos_code_idx ON morph_combos (species_id, code) WHERE code IS NOT NULL;

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
DROP INDEX IF EXISTS morph_combos_code_idx;
DROP TABLE IF EXISTS morph_combo_traits;
DROP TABLE IF EXISTS morph_combos;

-- Restore deleted traits (data only — no gecko_genes rows pointed at these).
INSERT INTO genetic_dictionary (species_id, trait_name, trait_code, description, is_dominant) VALUES
  ((SELECT id FROM species WHERE code = 'LP'), 'Raptor',     'RAPT',  'Combo: Tremper Albino + Eclipse.', FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Snow', 'SSNOW', 'Homozygous Mack Snow.',             TRUE)
ON CONFLICT DO NOTHING;

-- Remove traits added by this migration's Up.
DELETE FROM genetic_dictionary
WHERE (species_id, trait_name) IN (
  ((SELECT id FROM species WHERE code = 'LP'), 'Marble Eye'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Giant'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Lemon Frost'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Carrot Tail'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Baldy'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Melanistic / Black Night')
);

ALTER TABLE genetic_dictionary
  DROP COLUMN IF EXISTS inheritance_type,
  DROP COLUMN IF EXISTS super_form_name,
  DROP COLUMN IF EXISTS example_photo_url,
  DROP COLUMN IF EXISTS notes;

DROP TYPE IF EXISTS inheritance_type;
-- +goose StatementEnd
