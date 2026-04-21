-- +goose Up
-- +goose StatementBegin
CREATE TABLE genetic_dictionary (
  id          SERIAL PRIMARY KEY,
  species_id  INTEGER NOT NULL REFERENCES species(id) ON DELETE RESTRICT,
  trait_name  VARCHAR(100) NOT NULL,
  trait_code  VARCHAR(50),
  description TEXT,
  is_dominant BOOLEAN NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at  TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE (species_id, trait_name)
);
CREATE INDEX genetic_dictionary_species_idx ON genetic_dictionary (species_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE genetic_dictionary;
-- +goose StatementEnd
