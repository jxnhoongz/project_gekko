package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountDashboard registers GET /api/admin/dashboard.
func MountDashboard(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &dashboardDeps{q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/admin/dashboard", d.get)
	})
}

type dashboardDeps struct {
	q *db.Queries
}

// ---- DTOs (JSON shapes) ----

type dashStats struct {
	TotalGeckos int64 `json:"total_geckos"`
	Breeding    int64 `json:"breeding"`
	Available   int64 `json:"available"`
	Waitlist    int64 `json:"waitlist"`
}

type dashItem struct {
	Kind    string    `json:"kind"`
	Title   string    `json:"title"`
	Detail  string    `json:"detail"`
	At      time.Time `json:"at"`
	RefKind string    `json:"ref_kind"`
	RefID   int32     `json:"ref_id"`
}

// due_at is treated as the "at" timestamp too — same field name in JSON
// so the frontend can render the same list component for both panels.
type dashResponse struct {
	Stats           dashStats  `json:"stats"`
	NeedsAttention  []dashItem `json:"needs_attention"`
	RecentActivity  []dashItem `json:"recent_activity"`
}

// ---- handler ----

func (d *dashboardDeps) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var (
		stats      db.DashboardStatsRow
		needsRows  []db.DashboardNeedsAttentionRow
		recentRows []db.DashboardRecentActivityRow
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		s, err := d.q.DashboardStats(gctx)
		if err != nil {
			return fmt.Errorf("stats: %w", err)
		}
		stats = s
		return nil
	})
	g.Go(func() error {
		rows, err := d.q.DashboardNeedsAttention(gctx)
		if err != nil {
			return fmt.Errorf("needs_attention: %w", err)
		}
		needsRows = rows
		return nil
	})
	g.Go(func() error {
		rows, err := d.q.DashboardRecentActivity(gctx)
		if err != nil {
			return fmt.Errorf("recent_activity: %w", err)
		}
		recentRows = rows
		return nil
	})

	if err := g.Wait(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := dashResponse{
		Stats: dashStats{
			TotalGeckos: stats.TotalGeckos,
			Breeding:    stats.Breeding,
			Available:   stats.Available,
			Waitlist:    stats.Waitlist,
		},
		NeedsAttention: composeNeedsAttention(needsRows),
		RecentActivity: composeRecentActivity(recentRows),
	}
	writeJSON(w, http.StatusOK, resp)
}

// composeNeedsAttention turns raw SQL rows into user-facing title/detail
// strings. Split out as its own function so the formatting is testable.
func composeNeedsAttention(rows []db.DashboardNeedsAttentionRow) []dashItem {
	out := make([]dashItem, 0, len(rows))
	for _, r := range rows {
		days := daysAgo(r.DueAt.Time)
		item := dashItem{
			Kind:    r.Kind,
			At:      r.DueAt.Time,
			RefKind: r.RefKind,
			RefID:   r.RefID,
		}
		switch r.Kind {
		case "waitlist_stale":
			item.Title = "Follow up with " + r.Subject
			if r.DetailHint != "" {
				item.Detail = fmt.Sprintf("Waitlist · %d days since signup · %s", days, r.DetailHint)
			} else {
				item.Detail = fmt.Sprintf("Waitlist · %d days since signup", days)
			}
		case "hold_stale":
			item.Title = r.Subject + " on HOLD"
			item.Detail = fmt.Sprintf("%s · %d days", r.DetailHint, days)
		default:
			item.Title = r.Subject
			item.Detail = r.DetailHint
		}
		out = append(out, item)
	}
	return out
}

// composeRecentActivity turns raw SQL rows into dashItems with already-
// formatted titles from SQL, keeping only minor Go-side tidying.
func composeRecentActivity(rows []db.DashboardRecentActivityRow) []dashItem {
	out := make([]dashItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, dashItem{
			Kind:    r.Kind,
			Title:   r.Title,
			Detail:  r.Detail,
			At:      r.At.Time,
			RefKind: r.RefKind,
			RefID:   r.RefID,
		})
	}
	return out
}

// daysAgo returns whole days between t and now (never negative).
func daysAgo(t time.Time) int {
	d := int(timeNow().Sub(t).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}
