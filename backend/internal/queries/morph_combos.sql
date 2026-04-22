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
