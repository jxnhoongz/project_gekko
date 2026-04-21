-- name: ListAvailableGeckos :many
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.list_price_usd,
  g.created_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
WHERE g.status = 'AVAILABLE'
ORDER BY g.created_at DESC;

-- name: GetAvailableGeckoByCode :one
SELECT
  g.id, g.code, g.name, g.species_id, g.sex, g.hatch_date, g.list_price_usd,
  g.created_at,
  sp.code AS species_code,
  sp.common_name AS species_common_name
FROM geckos g
JOIN species sp ON sp.id = g.species_id
WHERE g.status = 'AVAILABLE' AND g.code = $1
LIMIT 1;

-- name: ListPublicGenesByGeckoIDs :many
-- Used to compose morphs for the list endpoint in one round trip.
SELECT
  gg.gecko_id,
  gd.trait_name,
  gd.trait_code,
  gg.zygosity
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
WHERE gg.gecko_id = ANY($1::int[])
ORDER BY gg.gecko_id, gd.trait_name;

-- name: ListPublicMediaByGeckoIDs :many
-- Used to pre-load cover photos (first by display_order) for the list view.
SELECT DISTINCT ON (gecko_id) gecko_id, url, caption, display_order, uploaded_at
FROM media
WHERE gecko_id = ANY($1::int[])
ORDER BY gecko_id, display_order, uploaded_at;
