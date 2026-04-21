-- +goose Up
-- +goose StatementBegin
CREATE TYPE media_type AS ENUM ('PROFILE', 'GALLERY', 'HUSBANDRY');

CREATE TABLE media (
  id            SERIAL PRIMARY KEY,
  gecko_id      INTEGER REFERENCES geckos(id) ON DELETE CASCADE,
  url           VARCHAR(500) NOT NULL,
  type          media_type NOT NULL DEFAULT 'GALLERY',
  caption       TEXT,
  display_order INTEGER NOT NULL DEFAULT 0,
  uploaded_at   TIMESTAMP DEFAULT NOW() NOT NULL
);
CREATE INDEX media_gecko_idx ON media (gecko_id, display_order);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE media;
DROP TYPE media_type;
-- +goose StatementEnd
