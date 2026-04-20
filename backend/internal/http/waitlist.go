package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

type waitlistDeps struct {
	q *db.Queries
}

// MountWaitlist registers admin-only waitlist routes on the given router.
func MountWaitlist(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &waitlistDeps{q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/waitlist", d.list)
	})
}

type waitlistEntryDTO struct {
	ID           int32      `json:"id"`
	Email        string     `json:"email"`
	Telegram     string     `json:"telegram"`
	Phone        string     `json:"phone"`
	InterestedIn string     `json:"interested_in"`
	Source       string     `json:"source"`
	Notes        string     `json:"notes"`
	ContactedAt  *time.Time `json:"contacted_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type waitlistListResp struct {
	Entries []waitlistEntryDTO `json:"entries"`
	Total   int64              `json:"total"`
}

func (d *waitlistDeps) list(w http.ResponseWriter, r *http.Request) {
	limit := parseInt32(r.URL.Query().Get("limit"), 50, 1, 200)
	offset := parseInt32(r.URL.Query().Get("offset"), 0, 0, 10000)

	rows, err := d.q.ListWaitlistEntries(r.Context(), db.ListWaitlistEntriesParams{
		Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}
	total, err := d.q.CountWaitlistEntries(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "count failed"})
		return
	}

	out := make([]waitlistEntryDTO, 0, len(rows))
	for _, e := range rows {
		out = append(out, waitlistEntryDTO{
			ID:           e.ID,
			Email:        e.Email,
			Telegram:     textOrEmpty(e.Telegram),
			Phone:        textOrEmpty(e.Phone),
			InterestedIn: textOrEmpty(e.InterestedIn),
			Source:       textOrEmpty(e.Source),
			Notes:        textOrEmpty(e.Notes),
			ContactedAt:  timeOrNil(e.ContactedAt),
			CreatedAt:    e.CreatedAt.Time,
		})
	}

	writeJSON(w, http.StatusOK, waitlistListResp{Entries: out, Total: total})
}

func parseInt32(s string, def, min, max int32) int32 {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	v := int32(n)
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func timeOrNil(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
