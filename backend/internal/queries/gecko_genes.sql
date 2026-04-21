-- name: ListGeckoGenes :many
SELECT
  gg.id, gg.gecko_id, gg.trait_id, gg.zygosity, gg.created_at,
  gd.trait_name, gd.trait_code, gd.is_dominant, gd.species_id
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
ORDER BY gg.gecko_id, gd.trait_name;

-- name: ListGenesForGecko :many
SELECT
  gg.id, gg.gecko_id, gg.trait_id, gg.zygosity, gg.created_at,
  gd.trait_name, gd.trait_code, gd.is_dominant, gd.species_id
FROM gecko_genes gg
JOIN genetic_dictionary gd ON gd.id = gg.trait_id
WHERE gg.gecko_id = $1
ORDER BY gd.trait_name;

-- name: CreateGeckoGene :one
INSERT INTO gecko_genes (gecko_id, trait_id, zygosity)
VALUES ($1, $2, $3)
RETURNING id, gecko_id, trait_id, zygosity, created_at;
