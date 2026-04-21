-- +goose Up
-- +goose StatementBegin
CREATE TYPE sex AS ENUM ('M', 'F', 'U');
CREATE TYPE gecko_status AS ENUM ('AVAILABLE', 'HOLD', 'BREEDING', 'PERSONAL', 'SOLD', 'DECEASED');

CREATE TABLE geckos (
  id             SERIAL PRIMARY KEY,
  code           VARCHAR(20) UNIQUE NOT NULL,
  name           VARCHAR(100),
  species_id     INTEGER NOT NULL REFERENCES species(id) ON DELETE RESTRICT,
  sex            sex NOT NULL DEFAULT 'U',
  hatch_date     DATE,
  acquired_date  DATE,
  status         gecko_status NOT NULL DEFAULT 'AVAILABLE',
  sire_id        INTEGER REFERENCES geckos(id) ON DELETE SET NULL,
  dam_id         INTEGER REFERENCES geckos(id) ON DELETE SET NULL,
  list_price_usd NUMERIC(10, 2),
  notes          TEXT,
  created_at     TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at     TIMESTAMP DEFAULT NOW() NOT NULL
);
CREATE INDEX geckos_species_idx ON geckos (species_id);
CREATE INDEX geckos_status_idx  ON geckos (status);
CREATE INDEX geckos_sire_idx    ON geckos (sire_id);
CREATE INDEX geckos_dam_idx     ON geckos (dam_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE geckos;
DROP TYPE gecko_status;
DROP TYPE sex;
-- +goose StatementEnd
