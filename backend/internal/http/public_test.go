package http

import (
	"bytes"
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

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func publicSetup(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	r := chi.NewRouter()
	MountPublic(r, pool)
	return r, pool
}

func makePublicGecko(t *testing.T, pool *pgxpool.Pool, code string, status db.GeckoStatus) int32 {
	t.Helper()
	q := db.New(pool)
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT id FROM species ORDER BY id LIMIT 1").Scan(&speciesID))
	g, err := q.CreateGecko(context.Background(), db.CreateGeckoParams{
		Code:      code,
		SpeciesID: speciesID,
		Sex:       db.SexM,
		Column7:   db.NullGeckoStatus{GeckoStatus: status, Valid: true},
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM listing_geckos WHERE gecko_id = $1", g.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM listings WHERE id IN (SELECT listing_id FROM listing_geckos WHERE gecko_id = $1)", g.ID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM geckos WHERE id = $1", g.ID)
	})
	return g.ID
}

// seedGeckoListing attaches a LISTED type=GECKO listing to the given gecko so
// the public endpoints will surface it. Returns the listing id. Cleanup of the
// listing + junction happens via makePublicGecko's cleanup hook (which deletes
// listing_geckos + listings first, then the gecko).
func seedGeckoListing(t *testing.T, pool *pgxpool.Pool, geckoID int32, priceUsd string) int32 {
	t.Helper()
	var listingID int32
	err := pool.QueryRow(context.Background(), `
		WITH l AS (
			INSERT INTO listings (title, type, status, price_usd, listed_at)
			VALUES ($1, 'GECKO', 'LISTED', $2, now())
			RETURNING id
		),
		j AS (
			INSERT INTO listing_geckos (listing_id, gecko_id)
			SELECT l.id, $3 FROM l
			RETURNING listing_id
		)
		SELECT id FROM l
	`, "test listing "+priceUsd, priceUsd, geckoID).Scan(&listingID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM listing_geckos WHERE listing_id = $1", listingID)
		_, _ = pool.Exec(context.Background(), "DELETE FROM listings WHERE id = $1", listingID)
	})
	return listingID
}

func TestPublicListGeckos_onlyListed(t *testing.T) {
	router, pool := publicSetup(t)
	stamp := time.Now().Format("150405000")
	// Public visibility is now driven by the presence of a LISTED type=GECKO
	// listing, not by gecko.status. Only PA gets a listing; PH + PB do not
	// (even though PA happens to be AVAILABLE here, that's incidental).
	paID := makePublicGecko(t, pool, "PA-"+stamp, db.GeckoStatusAVAILABLE)
	seedGeckoListing(t, pool, paID, "250.00")
	makePublicGecko(t, pool, "PH-"+stamp, db.GeckoStatusHOLD)
	makePublicGecko(t, pool, "PB-"+stamp, db.GeckoStatusBREEDING)

	req := httptest.NewRequest(http.MethodGet, "/api/public/geckos", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())

	var body struct {
		Geckos []publicGeckoDTO `json:"geckos"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

	var codes []string
	var paPrice *string
	for _, g := range body.Geckos {
		codes = append(codes, g.Code)
		if g.Code == "PA-"+stamp {
			paPrice = g.ListPriceUsd
		}
	}
	assert.Contains(t, codes, "PA-"+stamp)
	assert.NotContains(t, codes, "PH-"+stamp)
	assert.NotContains(t, codes, "PB-"+stamp)
	if assert.NotNil(t, paPrice, "list_price_usd should be populated from listing") {
		assert.Equal(t, "250.00", *paPrice)
	}
}

func TestPublicGetGecko_byCode_available(t *testing.T) {
	router, pool := publicSetup(t)
	stamp := time.Now().Format("150405000")
	code := "PG-" + stamp
	id := makePublicGecko(t, pool, code, db.GeckoStatusAVAILABLE)
	seedGeckoListing(t, pool, id, "425.00")

	req := httptest.NewRequest(http.MethodGet, "/api/public/geckos/"+code, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	raw := rr.Body.String()
	assert.Contains(t, raw, `"code":"`+code+`"`)
	assert.Contains(t, raw, `"list_price_usd":"425.00"`)
	assert.NotContains(t, raw, `"notes"`)
	assert.NotContains(t, raw, `"sire_id"`)
	assert.NotContains(t, raw, `"dam_id"`)
	assert.NotContains(t, raw, `"status"`)
}

func TestPublicGetGecko_byCode_notAvailable(t *testing.T) {
	router, pool := publicSetup(t)
	stamp := time.Now().Format("150405000")
	code := "PN-" + stamp
	makePublicGecko(t, pool, code, db.GeckoStatusHOLD)

	req := httptest.NewRequest(http.MethodGet, "/api/public/geckos/"+code, nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestPublicWaitlist_create(t *testing.T) {
	router, pool := publicSetup(t)
	email := "t+" + time.Now().Format("150405.000000000") + "@example.com"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM waitlist_entries WHERE email = $1", email)
	})

	body := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
	req := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())
	var got publicWaitlistResp
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.NotNil(t, got.ID)
}

func TestPublicWaitlist_duplicate(t *testing.T) {
	router, pool := publicSetup(t)
	email := "dup+" + time.Now().Format("150405.000000000") + "@example.com"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM waitlist_entries WHERE email = $1", email)
	})

	body1 := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
	req1 := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body1)
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()
	router.ServeHTTP(rr1, req1)
	require.Equal(t, http.StatusCreated, rr1.Code)

	body2 := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
	req2 := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body2)
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	router.ServeHTTP(rr2, req2)
	require.Equal(t, http.StatusOK, rr2.Code, "body=%s", rr2.Body.String())

	var got publicWaitlistResp
	require.NoError(t, json.Unmarshal(rr2.Body.Bytes(), &got))
	assert.True(t, got.Deduplicated)
	assert.Nil(t, got.ID)
}

func TestPublicWaitlist_rateLimit(t *testing.T) {
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// Build a router with a fresh limiter so we're not affected by other tests.
	r := chi.NewRouter()
	d := &publicDeps{pool: pool, q: db.New(pool)}
	limiter := NewIPRateLimiter(5, time.Hour)
	r.With(limiter.Middleware).Post("/api/public/waitlist", d.createWaitlist)

	emails := []string{}
	t.Cleanup(func() {
		for _, e := range emails {
			_, _ = pool.Exec(context.Background(), "DELETE FROM waitlist_entries WHERE email = $1", e)
		}
	})

	okCount, throttled := 0, 0
	for i := 0; i < 6; i++ {
		email := "rl+" + time.Now().Format("150405.000000000") + "@example.com"
		emails = append(emails, email)
		body := bytes.NewReader([]byte(`{"email":"` + email + `"}`))
		req := httptest.NewRequest(http.MethodPost, "/api/public/waitlist", body)
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.9.8.7:12345" // force the same IP for every request
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		switch rr.Code {
		case http.StatusCreated:
			okCount++
		case http.StatusTooManyRequests:
			throttled++
		}
	}
	assert.Equal(t, 5, okCount, "first 5 should be accepted")
	assert.Equal(t, 1, throttled, "6th should be throttled")

	_ = pgtype.Int4{} // keep pgtype import alive for other tests referencing it in the package
}
