-- name: ListAvailableGeckos :many
SELECT
    g.id,
    g.code,
    g.name,
    g.sex,
    g.hatch_date,
    s.code         AS species_code,
    s.common_name  AS species_common_name,
    l.price_usd    AS list_price_usd
FROM geckos g
JOIN species s       ON s.id = g.species_id
JOIN listing_geckos lg ON lg.gecko_id = g.id
JOIN listings l        ON l.id = lg.listing_id
                      AND l.type = 'GECKO'
                      AND l.status = 'LISTED'
ORDER BY g.created_at DESC;

-- name: GetAvailableGeckoByCode :one
SELECT
    g.id,
    g.code,
    g.name,
    g.sex,
    g.hatch_date,
    s.code         AS species_code,
    s.common_name  AS species_common_name,
    l.price_usd    AS list_price_usd
FROM geckos g
JOIN species s       ON s.id = g.species_id
JOIN listing_geckos lg ON lg.gecko_id = g.id
JOIN listings l        ON l.id = lg.listing_id
                      AND l.type = 'GECKO'
                      AND l.status = 'LISTED'
WHERE g.code = $1
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
