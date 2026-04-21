-- name: ListGeckos :many
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.acquired_date,
  g.status, g.sire_id, g.dam_id, g.notes,
  g.created_at, g.updated_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
ORDER BY g.code;

-- name: GetGeckoByID :one
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.acquired_date,
  g.status, g.sire_id, g.dam_id, g.notes,
  g.created_at, g.updated_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
WHERE g.id = $1
LIMIT 1;

-- name: GetGeckoByCode :one
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.acquired_date,
  g.status, g.sire_id, g.dam_id, g.notes,
  g.created_at, g.updated_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
WHERE g.code = $1
LIMIT 1;

-- name: CountGeckos :one
SELECT COUNT(*) FROM geckos;

-- name: CountGeckosByStatus :many
SELECT status, COUNT(*) AS count
FROM geckos
GROUP BY status;

-- name: CreateGecko :one
INSERT INTO geckos (
  code, name, species_id, sex, hatch_date, acquired_date,
  status, sire_id, dam_id, notes
)
VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, 'AVAILABLE'::gecko_status), $8, $9, $10)
RETURNING
  id, code, name, species_id, sex, hatch_date, acquired_date,
  status, sire_id, dam_id, notes,
  created_at, updated_at;

-- name: UpdateGecko :one
UPDATE geckos SET
  name           = $2,
  species_id     = $3,
  sex            = $4,
  hatch_date     = $5,
  acquired_date  = $6,
  status         = $7,
  sire_id        = $8,
  dam_id         = $9,
  notes          = $10,
  updated_at     = NOW()
WHERE id = $1
RETURNING
  id, code, name, species_id, sex, hatch_date, acquired_date,
  status, sire_id, dam_id, notes,
  created_at, updated_at;

-- name: DeleteGecko :exec
DELETE FROM geckos WHERE id = $1;

-- name: NextGeckoSequenceForSpeciesYear :one
-- $1 is the LIKE pattern, e.g. 'ZGLP-2026-%'. NULLIF guards against
-- legacy codes that don't have a 3rd segment (SPLIT_PART returns '').
SELECT COALESCE(
  MAX(NULLIF(SPLIT_PART(code, '-', 3), '')::integer),
  0
) + 1 AS next_num
FROM geckos
WHERE code LIKE $1;

-- name: DeleteGenesForGecko :exec
DELETE FROM gecko_genes WHERE gecko_id = $1;
