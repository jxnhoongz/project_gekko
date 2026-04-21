-- +goose Up
-- +goose StatementBegin

-- Reference data: the species catalog. Idempotent via ON CONFLICT.
INSERT INTO species (code, common_name, scientific_name, description) VALUES
  ('LP', 'Leopard Gecko',            'Eublepharis macularius',     'Popular pet gecko species from Pakistan, Afghanistan, and India.'),
  ('AF', 'African Fat-tailed Gecko', 'Hemitheconyx caudicinctus',  'Hardy gecko species from West Africa.'),
  ('CR', 'Crested Gecko',            'Correlophus ciliatus',       'Arboreal gecko species from New Caledonia.')
ON CONFLICT (code) DO NOTHING;

-- Reference data: genetic trait catalog.
-- Ported from legacy gekko_backend/migrations/002_seed_data.sql.
-- CR (Crested Gecko) traits left empty for now — add when operator starts breeding that species.
INSERT INTO genetic_dictionary (species_id, trait_name, trait_code, description, is_dominant) VALUES
  -- Leopard Gecko (LP)
  ((SELECT id FROM species WHERE code = 'LP'), 'Tremper Albino',        'TREM',   'Recessive albino strain with brown eyes.',                         FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Bell Albino',           'BELL',   'Recessive albino strain with ruby eyes.',                          FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Rainwater Albino',      'RAIN',   'Recessive albino strain with pink eyes.',                          FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Blizzard',              'BLIZ',   'Recessive pattern morph, solid white/yellow.',                     FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Murphy Patternless',    'MP',     'Recessive patternless morph.',                                     FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Enigma',                'ENIG',   'Dominant pattern morph associated with neurological issues.',      TRUE),
  ((SELECT id FROM species WHERE code = 'LP'), 'W&Y (White and Yellow)','WY',     'Dominant polygenic trait.',                                        TRUE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Mack Snow',             'MACK',   'Co-dominant color morph.',                                         TRUE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Snow',            'SSNOW',  'Homozygous Mack Snow.',                                            TRUE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Raptor',                'RAPT',   'Combo: Tremper Albino + Eclipse + Patternless.',                   FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Eclipse',               'ECL',    'Recessive eye morph (solid eyes).',                                FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Tangerine',             'TANG',   'Polygenic orange coloration.',                                     FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Hypo',                  'HYPO',   'Reduced pattern/spots.',                                           FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Hypo',            'SHYPO',  'No body spots.',                                                   FALSE),
  ((SELECT id FROM species WHERE code = 'LP'), 'Bold Stripe',           'BSTR',   'Striped pattern.',                                                 FALSE),

  -- African Fat-tailed Gecko (AF)
  ((SELECT id FROM species WHERE code = 'AF'), 'Albino',                'ALB',    'Recessive albino.',                                                FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Patternless',           'PATL',   'Recessive patternless.',                                           FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Oreo',                  'OREO',   'Black and white pattern.',                                         FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Ghost',                 'GHST',   'Reduced pattern and color.',                                       FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Stinger',               'STING',  'Yellow tail trait.',                                               FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Whiteout',              'WOUT',   'Extreme leucistic.',                                               FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Caramel',               'CARA',   'Caramel coloration.',                                              FALSE),
  ((SELECT id FROM species WHERE code = 'AF'), 'Zulu',                  'ZULU',   'Dark coloration.',                                                 FALSE)
ON CONFLICT (species_id, trait_name) DO NOTHING;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Remove only the reference rows added by this migration, leaving any
-- operator-added traits intact.
DELETE FROM genetic_dictionary
WHERE (species_id, trait_name) IN (
  ((SELECT id FROM species WHERE code = 'LP'), 'Tremper Albino'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Bell Albino'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Rainwater Albino'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Blizzard'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Murphy Patternless'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Enigma'),
  ((SELECT id FROM species WHERE code = 'LP'), 'W&Y (White and Yellow)'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Mack Snow'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Snow'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Raptor'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Eclipse'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Tangerine'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Hypo'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Super Hypo'),
  ((SELECT id FROM species WHERE code = 'LP'), 'Bold Stripe'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Albino'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Patternless'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Oreo'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Ghost'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Stinger'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Whiteout'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Caramel'),
  ((SELECT id FROM species WHERE code = 'AF'), 'Zulu')
);

DELETE FROM species WHERE code IN ('LP', 'AF', 'CR');
-- +goose StatementEnd
