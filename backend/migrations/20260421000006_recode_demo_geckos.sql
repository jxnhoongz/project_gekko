-- +goose Up
-- +goose StatementBegin
-- Re-code the 6 demo geckos to the new ZG<species>-<year>-<nnn> format.
-- Uses a join on species so the migration is species-aware.
UPDATE geckos SET code = 'ZGLP-2026-001' WHERE code = 'ZG-001';
UPDATE geckos SET code = 'ZGLP-2026-002' WHERE code = 'ZG-002';
UPDATE geckos SET code = 'ZGCR-2026-001' WHERE code = 'ZG-003';
UPDATE geckos SET code = 'ZGCR-2026-002' WHERE code = 'ZG-004';
UPDATE geckos SET code = 'ZGAF-2026-001' WHERE code = 'ZG-005';
UPDATE geckos SET code = 'ZGLP-2026-003' WHERE code = 'ZG-006';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
UPDATE geckos SET code = 'ZG-001' WHERE code = 'ZGLP-2026-001';
UPDATE geckos SET code = 'ZG-002' WHERE code = 'ZGLP-2026-002';
UPDATE geckos SET code = 'ZG-003' WHERE code = 'ZGCR-2026-001';
UPDATE geckos SET code = 'ZG-004' WHERE code = 'ZGCR-2026-002';
UPDATE geckos SET code = 'ZG-005' WHERE code = 'ZGAF-2026-001';
UPDATE geckos SET code = 'ZG-006' WHERE code = 'ZGLP-2026-003';
-- +goose StatementEnd
