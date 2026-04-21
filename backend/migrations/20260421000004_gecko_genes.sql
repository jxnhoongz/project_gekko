-- +goose Up
-- +goose StatementBegin
CREATE TYPE zygosity AS ENUM ('HOM', 'HET', 'POSS_HET');

CREATE TABLE gecko_genes (
  id         SERIAL PRIMARY KEY,
  gecko_id   INTEGER NOT NULL REFERENCES geckos(id) ON DELETE CASCADE,
  trait_id   INTEGER NOT NULL REFERENCES genetic_dictionary(id) ON DELETE RESTRICT,
  zygosity   zygosity NOT NULL,
  created_at TIMESTAMP DEFAULT NOW() NOT NULL,
  UNIQUE (gecko_id, trait_id)
);
CREATE INDEX gecko_genes_gecko_idx ON gecko_genes (gecko_id);
CREATE INDEX gecko_genes_trait_idx ON gecko_genes (trait_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE gecko_genes;
DROP TYPE zygosity;
-- +goose StatementEnd
