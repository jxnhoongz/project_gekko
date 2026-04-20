-- +goose Up
-- +goose StatementBegin
CREATE TABLE waitlist_entries (
  id            SERIAL PRIMARY KEY,
  email         VARCHAR(255) NOT NULL,
  telegram      VARCHAR(100),
  phone         VARCHAR(32),
  interested_in VARCHAR(100),
  source        VARCHAR(50) DEFAULT 'website',
  notes         TEXT,
  contacted_at  TIMESTAMP,
  created_at    TIMESTAMP DEFAULT NOW() NOT NULL,
  updated_at    TIMESTAMP DEFAULT NOW() NOT NULL
);
CREATE UNIQUE INDEX waitlist_entries_email_idx ON waitlist_entries (LOWER(email));
CREATE INDEX waitlist_entries_created_idx ON waitlist_entries (created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE waitlist_entries;
-- +goose StatementEnd
