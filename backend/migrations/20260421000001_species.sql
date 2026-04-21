-- +goose Up
-- +goose StatementBegin
CREATE TYPE species_code AS ENUM ('LP', 'AF', 'CR');

CREATE TABLE species (
  id              SERIAL PRIMARY KEY,
  code            species_code UNIQUE NOT NULL,
  common_name     VARCHAR(100) NOT NULL,
  scientific_name VARCHAR(150),
  description     TEXT,
  created_at      TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at      TIMESTAMP DEFAULT NOW() NOT NULL
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE species;
DROP TYPE species_code;
-- +goose StatementEnd
