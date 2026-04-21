-- name: ListTraits :many
SELECT id, species_id, trait_name, trait_code, description, is_dominant, created_at, updated_at
FROM genetic_dictionary
ORDER BY species_id, trait_name;

-- name: ListTraitsBySpecies :many
SELECT id, species_id, trait_name, trait_code, description, is_dominant, created_at, updated_at
FROM genetic_dictionary
WHERE species_id = $1
ORDER BY trait_name;

-- name: GetTraitByNameAndSpecies :one
SELECT id, species_id, trait_name, trait_code, description, is_dominant, created_at, updated_at
FROM genetic_dictionary
WHERE species_id = $1 AND LOWER(trait_name) = LOWER($2)
LIMIT 1;

-- name: CreateTrait :one
INSERT INTO genetic_dictionary (species_id, trait_name, trait_code, description, is_dominant)
VALUES ($1, $2, $3, $4, COALESCE($5, FALSE))
RETURNING id, species_id, trait_name, trait_code, description, is_dominant, created_at, updated_at;
