-- name: DashboardStats :one
SELECT
  (SELECT COUNT(*) FROM geckos)                              AS total_geckos,
  (SELECT COUNT(*) FROM geckos WHERE status = 'BREEDING')    AS breeding,
  (SELECT COUNT(*) FROM geckos WHERE status = 'AVAILABLE')   AS available,
  (SELECT COUNT(*) FROM waitlist_entries)                    AS waitlist;

-- name: DashboardNeedsAttention :many
-- Waitlist entries that have been sitting uncontacted >7 days, and
-- geckos that have been on HOLD >7 days without any status change.
-- Top 6 merged by staleness (oldest first).
(
  SELECT 'waitlist_stale'::text AS kind,
         w.id            AS ref_id,
         'waitlist'::text AS ref_kind,
         w.email         AS subject,
         COALESCE(w.interested_in, '') AS detail_hint,
         w.created_at    AS due_at
  FROM waitlist_entries w
  WHERE w.contacted_at IS NULL
    AND w.created_at < NOW() - INTERVAL '7 days'
)
UNION ALL
(
  SELECT 'hold_stale'::text AS kind,
         g.id         AS ref_id,
         'gecko'::text AS ref_kind,
         COALESCE(g.name, g.code) AS subject,
         g.code       AS detail_hint,
         g.updated_at AS due_at
  FROM geckos g
  WHERE g.status = 'HOLD'
    AND g.updated_at < NOW() - INTERVAL '7 days'
)
ORDER BY due_at ASC
LIMIT 6;

-- name: DashboardRecentActivity :many
-- Top 15 most-recent events across three sources. Keep the shape
-- identical across branches so sqlc generates a single row type.
(
  SELECT 'gecko_created'::text AS kind,
         g.id                  AS ref_id,
         'gecko'::text         AS ref_kind,
         COALESCE(g.name, g.code) AS title,
         g.code                AS detail,
         g.created_at          AS at
  FROM geckos g
)
UNION ALL
(
  SELECT 'waitlist_created'::text AS kind,
         w.id                      AS ref_id,
         'waitlist'::text          AS ref_kind,
         'New waitlist signup'     AS title,
         w.email                   AS detail,
         w.created_at              AS at
  FROM waitlist_entries w
)
UNION ALL
(
  SELECT 'media_uploaded'::text AS kind,
         m.id                   AS ref_id,
         'gecko'::text          AS ref_kind,
         'Photo added to ' || COALESCE(g.name, g.code, '(unknown)') AS title,
         ''                     AS detail,
         m.uploaded_at          AS at
  FROM media m
  LEFT JOIN geckos g ON g.id = m.gecko_id
  WHERE m.gecko_id IS NOT NULL
)
ORDER BY at DESC
LIMIT 15;
