-- name: ListSpecies :many
SELECT id, code, common_name, scientific_name, description, created_at, updated_at
FROM species
ORDER BY common_name;

-- name: GetSpeciesByCode :one
SELECT id, code, common_name, scientific_name, description, created_at, updated_at
FROM species
WHERE code = $1
LIMIT 1;

-- name: GetSpeciesByID :one
SELECT id, code, common_name, scientific_name, description, created_at, updated_at
FROM species
WHERE id = $1
LIMIT 1;

-- name: CreateSpecies :one
INSERT INTO species (code, common_name, scientific_name, description)
VALUES ($1, $2, $3, $4)
RETURNING id, code, common_name, scientific_name, description, created_at, updated_at;
