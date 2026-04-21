-- +goose Up
-- +goose StatementBegin

-- Enums first so the table definition can reference them.
CREATE TYPE listing_type   AS ENUM ('GECKO', 'PACKAGE', 'SUPPLY');
CREATE TYPE listing_status AS ENUM ('DRAFT', 'LISTED', 'RESERVED', 'SOLD', 'ARCHIVED');

CREATE TABLE listings (
  id               SERIAL PRIMARY KEY,
  sku              VARCHAR(64) UNIQUE,
  type             listing_type NOT NULL,
  title            VARCHAR(200) NOT NULL,
  description      TEXT,
  price_usd        NUMERIC(10,2) NOT NULL,
  deposit_usd      NUMERIC(10,2),
  status           listing_status NOT NULL DEFAULT 'DRAFT',
  cover_photo_url  VARCHAR(500),
  listed_at        TIMESTAMP,
  sold_at          TIMESTAMP,
  archived_at      TIMESTAMP,
  created_at       TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at       TIMESTAMP DEFAULT NOW() NOT NULL
);
CREATE INDEX listings_type_status_idx ON listings (type, status);

CREATE TABLE listing_geckos (
  listing_id  INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  gecko_id    INTEGER NOT NULL REFERENCES geckos(id)   ON DELETE RESTRICT,
  created_at  TIMESTAMP DEFAULT NOW() NOT NULL,
  PRIMARY KEY (listing_id, gecko_id)
);
CREATE INDEX listing_geckos_gecko_idx ON listing_geckos (gecko_id);

CREATE TABLE listing_components (
  listing_id             INTEGER NOT NULL REFERENCES listings(id) ON DELETE CASCADE,
  component_listing_id   INTEGER NOT NULL REFERENCES listings(id) ON DELETE RESTRICT,
  quantity               INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
  created_at             TIMESTAMP DEFAULT NOW() NOT NULL,
  PRIMARY KEY (listing_id, component_listing_id),
  CHECK (listing_id <> component_listing_id)
);
CREATE INDEX listing_components_component_idx ON listing_components (component_listing_id);

-- Data migration: every gecko with a non-null list_price_usd becomes a
-- LISTED gecko listing with a matching junction row. Title defaults to
-- the gecko's name (or code if no name). Junction is found by title
-- match, which is safe because geckos.code is unique.
WITH inserted AS (
  INSERT INTO listings (type, title, price_usd, status, listed_at)
  SELECT 'GECKO'::listing_type,
         COALESCE(g.name, g.code),
         g.list_price_usd,
         'LISTED'::listing_status,
         NOW()
  FROM geckos g
  WHERE g.list_price_usd IS NOT NULL
  RETURNING id, title
)
INSERT INTO listing_geckos (listing_id, gecko_id)
SELECT i.id, g.id
FROM inserted i
JOIN geckos g ON COALESCE(g.name, g.code) = i.title;

ALTER TABLE geckos DROP COLUMN list_price_usd;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE geckos ADD COLUMN list_price_usd NUMERIC(10,2);

-- Best-effort restore: copy price back to gecko when attached to a single
-- GECKO listing.
UPDATE geckos g
SET list_price_usd = l.price_usd
FROM listings l
JOIN listing_geckos lg ON lg.listing_id = l.id
WHERE l.type = 'GECKO' AND lg.gecko_id = g.id;

DROP TABLE listing_components;
DROP TABLE listing_geckos;
DROP TABLE listings;
DROP TYPE  listing_status;
DROP TYPE  listing_type;

-- +goose StatementEnd
