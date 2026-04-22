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

-- name: GetTraitByID :one
SELECT id, species_id, trait_name, trait_code, description, is_dominant,
       inheritance_type, super_form_name, example_photo_url, notes,
       created_at, updated_at
FROM genetic_dictionary
WHERE id = $1;

-- name: UpdateTrait :one
UPDATE genetic_dictionary
SET trait_code        = $2,
    description       = $3,
    notes             = $4,
    inheritance_type  = $5,
    super_form_name   = $6,
    example_photo_url = $7,
    updated_at        = NOW()
WHERE id = $1
RETURNING id, species_id, trait_name, trait_code, description, is_dominant,
          inheritance_type, super_form_name, example_photo_url, notes,
          created_at, updated_at;
