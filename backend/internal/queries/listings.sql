-- name: ListListings :many
SELECT
  l.id, l.sku, l.type, l.title, l.description, l.price_usd, l.deposit_usd,
  l.status, l.cover_photo_url, l.listed_at, l.sold_at, l.archived_at,
  l.created_at, l.updated_at,
  COALESCE(gc.n, 0)::int AS gecko_count,
  COALESCE(cc.n, 0)::int AS component_count
FROM listings l
LEFT JOIN (
  SELECT listing_id, COUNT(*) AS n
  FROM listing_geckos GROUP BY listing_id
) gc ON gc.listing_id = l.id
LEFT JOIN (
  SELECT listing_id, COUNT(*) AS n
  FROM listing_components GROUP BY listing_id
) cc ON cc.listing_id = l.id
ORDER BY
  CASE l.status
    WHEN 'LISTED'   THEN 0
    WHEN 'RESERVED' THEN 1
    WHEN 'DRAFT'    THEN 2
    WHEN 'SOLD'     THEN 3
    WHEN 'ARCHIVED' THEN 4
  END,
  l.created_at DESC;

-- name: GetListing :one
SELECT
  l.id, l.sku, l.type, l.title, l.description, l.price_usd, l.deposit_usd,
  l.status, l.cover_photo_url, l.listed_at, l.sold_at, l.archived_at,
  l.created_at, l.updated_at
FROM listings l
WHERE l.id = $1
LIMIT 1;

-- name: GetListingBySku :one
SELECT id FROM listings WHERE sku = $1 LIMIT 1;

-- name: CreateListing :one
INSERT INTO listings (
  sku, type, title, description, price_usd, deposit_usd, status,
  cover_photo_url, listed_at
)
VALUES (
  $1, $2, $3, $4, $5, $6, COALESCE($7, 'DRAFT'::listing_status),
  $8,
  CASE WHEN COALESCE($7, 'DRAFT'::listing_status) = 'LISTED'::listing_status THEN NOW() ELSE NULL END
)
RETURNING
  id, sku, type, title, description, price_usd, deposit_usd, status,
  cover_photo_url, listed_at, sold_at, archived_at, created_at, updated_at;

-- name: UpdateListing :one
-- Status transitions into LISTED/SOLD/ARCHIVED auto-stamp the matching
-- timestamp the first time it happens; existing stamps are preserved so
-- the audit trail survives later status moves.
UPDATE listings SET
  sku             = $2,
  title           = $3,
  description     = $4,
  price_usd       = $5,
  deposit_usd     = $6,
  status          = $7,
  cover_photo_url = $8,
  listed_at       = CASE
                      WHEN $7 = 'LISTED'::listing_status AND listed_at IS NULL THEN NOW()
                      ELSE listed_at
                    END,
  sold_at         = CASE
                      WHEN $7 = 'SOLD'::listing_status AND sold_at IS NULL THEN NOW()
                      ELSE sold_at
                    END,
  archived_at     = CASE
                      WHEN $7 = 'ARCHIVED'::listing_status AND archived_at IS NULL THEN NOW()
                      ELSE archived_at
                    END,
  updated_at      = NOW()
WHERE id = $1
RETURNING
  id, sku, type, title, description, price_usd, deposit_usd, status,
  cover_photo_url, listed_at, sold_at, archived_at, created_at, updated_at;

-- name: DeleteListing :exec
DELETE FROM listings WHERE id = $1;

-- name: AttachGeckoToListing :exec
INSERT INTO listing_geckos (listing_id, gecko_id)
VALUES ($1, $2);

-- name: DetachGeckosForListing :exec
DELETE FROM listing_geckos WHERE listing_id = $1;

-- name: ListGeckosForListing :many
SELECT
  lg.gecko_id,
  g.code,
  g.name,
  sp.code AS species_code
FROM listing_geckos lg
JOIN geckos g   ON g.id = lg.gecko_id
JOIN species sp ON sp.id = g.species_id
WHERE lg.listing_id = $1
ORDER BY g.code;

-- name: ListListingsForGecko :many
-- Used by admin gecko detail page.
SELECT
  l.id, l.title, l.type, l.status, l.price_usd
FROM listings l
JOIN listing_geckos lg ON lg.listing_id = l.id
WHERE lg.gecko_id = $1
ORDER BY l.created_at DESC;

-- name: SetListingComponent :exec
INSERT INTO listing_components (listing_id, component_listing_id, quantity)
VALUES ($1, $2, $3)
ON CONFLICT (listing_id, component_listing_id) DO UPDATE SET quantity = EXCLUDED.quantity;

-- name: DeleteComponentsForListing :exec
DELETE FROM listing_components WHERE listing_id = $1;

-- name: ListComponentsForListing :many
SELECT
  lc.component_listing_id,
  lc.quantity,
  l.title,
  l.type,
  l.price_usd
FROM listing_components lc
JOIN listings l ON l.id = lc.component_listing_id
WHERE lc.listing_id = $1
ORDER BY l.title;
