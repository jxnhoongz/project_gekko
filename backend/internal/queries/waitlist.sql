-- name: CreateWaitlistEntry :one
INSERT INTO waitlist_entries (email, telegram, phone, interested_in, source, notes)
VALUES ($1, $2, $3, $4, COALESCE($5, 'website'), $6)
RETURNING id, email, telegram, phone, interested_in, source, notes, contacted_at, created_at, updated_at;

-- name: ListWaitlistEntries :many
SELECT id, email, telegram, phone, interested_in, source, notes, contacted_at, created_at, updated_at
FROM waitlist_entries
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountWaitlistEntries :one
SELECT COUNT(*) FROM waitlist_entries;
