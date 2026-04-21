-- name: ListGeckos :many
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.acquired_date,
  g.status, g.sire_id, g.dam_id, g.list_price_usd, g.notes,
  g.created_at, g.updated_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
ORDER BY g.code;

-- name: GetGeckoByID :one
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.acquired_date,
  g.status, g.sire_id, g.dam_id, g.list_price_usd, g.notes,
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
  g.status, g.sire_id, g.dam_id, g.list_price_usd, g.notes,
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
  status, sire_id, dam_id, list_price_usd, notes
)
VALUES ($1, $2, $3, $4, $5, $6, COALESCE($7, 'AVAILABLE'::gecko_status), $8, $9, $10, $11)
RETURNING
  id, code, name, species_id, sex, hatch_date, acquired_date,
  status, sire_id, dam_id, list_price_usd, notes,
  created_at, updated_at;
