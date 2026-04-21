-- name: ListMediaForGecko :many
SELECT id, gecko_id, url, type, caption, display_order, uploaded_at
FROM media
WHERE gecko_id = $1
ORDER BY display_order, uploaded_at;

-- name: GetCoverForGecko :one
SELECT id, gecko_id, url, type, caption, display_order, uploaded_at
FROM media
WHERE gecko_id = $1
ORDER BY display_order, uploaded_at
LIMIT 1;

-- name: CreateMedia :one
INSERT INTO media (gecko_id, url, type, caption, display_order)
VALUES ($1, $2, COALESCE($3, 'GALLERY'::media_type), $4, COALESCE($5, 0))
RETURNING id, gecko_id, url, type, caption, display_order, uploaded_at;

-- name: GetMediaByID :one
SELECT id, gecko_id, url, type, caption, display_order, uploaded_at
FROM media
WHERE id = $1
LIMIT 1;

-- name: DeleteMedia :exec
DELETE FROM media WHERE id = $1;

-- name: CountMediaForGecko :one
SELECT COUNT(*) FROM media WHERE gecko_id = $1;
