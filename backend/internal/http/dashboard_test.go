package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func dashSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool, int32) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// Admin so we get a token.
	email := "dash+" + time.Now().Format("150405.000000000") + "@example.com"
	hash, err := auth.HashPassword("test-password-123")
	require.NoError(t, err)
	q := db.New(pool)
	admin, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email: email, PasswordHash: hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM admin_users WHERE id = $1", admin.ID)
	})

	signer := auth.NewJWTSigner("test-secret", time.Hour)
	tok, err := signer.Issue(int64(admin.ID), admin.Email)
	require.NoError(t, err)

	r := chi.NewRouter()
	MountDashboard(r, pool, signer)
	return r, tok, pool, admin.ID
}

func TestDashboard_endpoint_returnsAll(t *testing.T) {
	router, tok, _, _ := dashSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var body struct {
		Stats          dashStats  `json:"stats"`
		NeedsAttention []dashItem `json:"needs_attention"`
		RecentActivity []dashItem `json:"recent_activity"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

	assert.GreaterOrEqual(t, body.Stats.TotalGeckos, int64(0))
	assert.GreaterOrEqual(t, body.Stats.Waitlist, int64(0))
	assert.NotNil(t, body.NeedsAttention, "needs_attention must always be an array, never null")
	assert.NotNil(t, body.RecentActivity, "recent_activity must always be an array, never null")
}

func TestDashboard_endpoint_requiresAuth(t *testing.T) {
	router, _, _, _ := dashSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestComposeNeedsAttention_waitlistStale(t *testing.T) {
	rows := []db.DashboardNeedsAttentionRow{
		{
			Kind:       "waitlist_stale",
			RefID:      42,
			RefKind:    "waitlist",
			Subject:    "maly@example.com",
			DetailHint: "Crested gecko",
			DueAt:      pgtype.Timestamp{Time: time.Now().Add(-11 * 24 * time.Hour), Valid: true},
		},
	}
	out := composeNeedsAttention(rows)
	require.Len(t, out, 1)
	assert.Equal(t, "waitlist_stale", out[0].Kind)
	assert.Equal(t, "Follow up with maly@example.com", out[0].Title)
	assert.Contains(t, out[0].Detail, "Crested gecko")
	assert.Contains(t, out[0].Detail, "11 days")
	assert.Equal(t, "waitlist", out[0].RefKind)
	assert.Equal(t, int32(42), out[0].RefID)
}

func TestComposeNeedsAttention_holdStale(t *testing.T) {
	rows := []db.DashboardNeedsAttentionRow{
		{
			Kind:       "hold_stale",
			RefID:      7,
			RefKind:    "gecko",
			Subject:    "Veasna",
			DetailHint: "ZGLP-2026-003",
			DueAt:      pgtype.Timestamp{Time: time.Now().Add(-9 * 24 * time.Hour), Valid: true},
		},
	}
	out := composeNeedsAttention(rows)
	require.Len(t, out, 1)
	assert.Equal(t, "Veasna on HOLD", out[0].Title)
	assert.Contains(t, out[0].Detail, "ZGLP-2026-003")
	assert.Contains(t, out[0].Detail, "9 days")
}
