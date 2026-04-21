package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// listingsSetup spins up a pgxpool, creates an admin, mounts the listings
// router, and returns (router, bearer-token, pool). All DB objects clean up
// via t.Cleanup so tests are idempotent.
func listingsSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := "listings+" + time.Now().Format("150405.000000000") + "@example.com"
	hash, err := auth.HashPassword("test-password-123")
	require.NoError(t, err)
	q := db.New(pool)
	admin, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email:        email,
		PasswordHash: hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM admin_users WHERE id = $1", admin.ID)
	})

	signer := auth.NewJWTSigner("test-secret", time.Hour)
	tok, err := signer.Issue(int64(admin.ID), admin.Email)
	require.NoError(t, err)

	r := chi.NewRouter()
	MountListings(r, pool, signer)
	return r, tok, pool
}

// seedGeckoForListing creates a gecko row under an arbitrary species with a
// unique code (so multiple calls in a single test don't collide on the code
// unique index). Registers cleanup that deletes the junction first (to skirt
// listing_geckos ON DELETE RESTRICT) then the gecko itself.
func seedGeckoForListing(t *testing.T, pool *pgxpool.Pool, suffix string) int32 {
	t.Helper()
	q := db.New(pool)
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM species ORDER BY id LIMIT 1").Scan(&speciesID))

	// geckos.code is varchar(20), so keep the code tight. nanoseconds mod
	// 10^7 gives 7 digits — plenty of entropy for a single test run.
	code := fmt.Sprintf("TL%07d%s", time.Now().UnixNano()%10_000_000, suffix)
	if len(code) > 20 {
		code = code[:20]
	}
	g, err := q.CreateGecko(context.Background(), db.CreateGeckoParams{
		Code:      code,
		SpeciesID: speciesID,
		Sex:       db.SexU,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		// Junction has ON DELETE RESTRICT on gecko_id, so remove junction rows
		// first in case a test's listing delete didn't cascade for some reason.
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM listing_geckos WHERE gecko_id = $1", g.ID)
		_, _ = pool.Exec(context.Background(),
			"DELETE FROM geckos WHERE id = $1", g.ID)
	})
	return g.ID
}

func postListingReq(t *testing.T, router http.Handler, tok, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/listings", bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func patchListingReq(t *testing.T, router http.Handler, tok string, id int32, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPatch, "/api/listings/"+strconvItoa(int(id)),
		bytes.NewReader([]byte(body)))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func getListingReq(t *testing.T, router http.Handler, tok string, id int32) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/listings/"+strconvItoa(int(id)), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func deleteListingReq(t *testing.T, router http.Handler, tok string, id int32) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, "/api/listings/"+strconvItoa(int(id)), nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

// uniqueSku builds a SKU that won't collide across concurrent test runs.
// Nanosecond-resolution timestamp + test-specific tag.
func uniqueSku(tag string) string {
	return fmt.Sprintf("%s-%d", tag, time.Now().UnixNano())
}

// cleanupListing registers a t.Cleanup that unwinds junctions + the listing
// row in the FK-safe order (components, geckos, listing).
func cleanupListing(t *testing.T, pool *pgxpool.Pool, id int32) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, "DELETE FROM listing_components WHERE listing_id = $1 OR component_listing_id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM listing_geckos WHERE listing_id = $1", id)
		_, _ = pool.Exec(ctx, "DELETE FROM listings WHERE id = $1", id)
	})
}

// 1. SUPPLY happy path: SKU, status=LISTED, price, expects 201 with
//    listed_at non-null and both counts zero.
func TestCreateListing_supply_happy(t *testing.T) {
	router, tok, pool := listingsSetup(t)
	sku := uniqueSku("SUPPLY-OK")

	rr := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+sku+`","title":"Glass tank 20g","price_usd":"89.99","status":"LISTED"}`)
	require.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	var got listingDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	cleanupListing(t, pool, got.ID)

	assert.Equal(t, "SUPPLY", got.Type)
	assert.Equal(t, "LISTED", got.Status)
	assert.Equal(t, sku, got.Sku)
	assert.Equal(t, "89.99", got.PriceUsd)
	assert.NotNil(t, got.ListedAt, "listed_at should be auto-stamped on LISTED create")
	assert.Equal(t, int32(0), got.GeckoCount)
	assert.Equal(t, int32(0), got.ComponentCount)
}

// 2. GECKO happy path: seed a gecko, POST GECKO listing referencing it,
//    expect 201 + GET detail shows the gecko attached with count=1.
func TestCreateListing_gecko_happy(t *testing.T) {
	router, tok, pool := listingsSetup(t)
	geckoID := seedGeckoForListing(t, pool, "g1")

	body := `{"type":"GECKO","title":"Leopard juvie","price_usd":"275.00","status":"LISTED","geckos":[{"gecko_id":` + strconvItoa(int(geckoID)) + `}]}`
	rr := postListingReq(t, router, tok, body)
	require.Equal(t, http.StatusCreated, rr.Code, "body=%s", rr.Body.String())

	var created listingDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &created))
	cleanupListing(t, pool, created.ID)
	assert.Equal(t, "GECKO", created.Type)
	assert.NotNil(t, created.ListedAt)

	// Re-GET for a freshly rendered detail body.
	detailRR := getListingReq(t, router, tok, created.ID)
	require.Equal(t, http.StatusOK, detailRR.Code, "body=%s", detailRR.Body.String())

	var got listingDTO
	require.NoError(t, json.Unmarshal(detailRR.Body.Bytes(), &got))
	require.Len(t, got.Geckos, 1)
	assert.Equal(t, geckoID, got.Geckos[0].GeckoID)
	assert.Equal(t, int32(1), got.GeckoCount)
}

// 3. PACKAGE happy path: seed two SUPPLY listings, POST a PACKAGE that
//    references them with distinct quantities, verify GET detail echoes
//    both components with the quantities we sent.
func TestCreateListing_package_happy(t *testing.T) {
	router, tok, pool := listingsSetup(t)

	tankSku := uniqueSku("PKG-TANK")
	hideSku := uniqueSku("PKG-HIDE")

	tankRR := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+tankSku+`","title":"Tank","price_usd":"90.00"}`)
	require.Equal(t, http.StatusCreated, tankRR.Code, "body=%s", tankRR.Body.String())
	var tank listingDTO
	require.NoError(t, json.Unmarshal(tankRR.Body.Bytes(), &tank))
	cleanupListing(t, pool, tank.ID)

	hideRR := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+hideSku+`","title":"Hide","price_usd":"8.00"}`)
	require.Equal(t, http.StatusCreated, hideRR.Code, "body=%s", hideRR.Body.String())
	var hide listingDTO
	require.NoError(t, json.Unmarshal(hideRR.Body.Bytes(), &hide))
	cleanupListing(t, pool, hide.ID)

	pkgBody := fmt.Sprintf(
		`{"type":"PACKAGE","title":"Starter kit","price_usd":"99.00","components":[{"component_listing_id":%d,"quantity":1},{"component_listing_id":%d,"quantity":2}]}`,
		tank.ID, hide.ID,
	)
	pkgRR := postListingReq(t, router, tok, pkgBody)
	require.Equal(t, http.StatusCreated, pkgRR.Code, "body=%s", pkgRR.Body.String())
	var pkg listingDTO
	require.NoError(t, json.Unmarshal(pkgRR.Body.Bytes(), &pkg))
	cleanupListing(t, pool, pkg.ID)

	// GET the detail — it re-reads from DB, so this also validates the join.
	detailRR := getListingReq(t, router, tok, pkg.ID)
	require.Equal(t, http.StatusOK, detailRR.Code, "body=%s", detailRR.Body.String())
	var got listingDTO
	require.NoError(t, json.Unmarshal(detailRR.Body.Bytes(), &got))

	require.Len(t, got.Components, 2)
	byID := map[int32]int32{}
	for _, c := range got.Components {
		byID[c.ComponentListingID] = c.Quantity
	}
	assert.Equal(t, int32(1), byID[tank.ID])
	assert.Equal(t, int32(2), byID[hide.ID])
	assert.Equal(t, int32(2), got.ComponentCount)
}

// 4. Duplicate SKU surfaces as 409 with the handler's canonical error.
func TestCreateListing_duplicateSku_409(t *testing.T) {
	router, tok, pool := listingsSetup(t)
	sku := uniqueSku("DUP-SKU")

	rr1 := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+sku+`","title":"First","price_usd":"5.00"}`)
	require.Equal(t, http.StatusCreated, rr1.Code, "body=%s", rr1.Body.String())
	var first listingDTO
	require.NoError(t, json.Unmarshal(rr1.Body.Bytes(), &first))
	cleanupListing(t, pool, first.ID)

	rr2 := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+sku+`","title":"Second","price_usd":"5.00"}`)
	assert.Equal(t, http.StatusConflict, rr2.Code, "body=%s", rr2.Body.String())
	assert.Contains(t, rr2.Body.String(), "sku already in use")
}

// 5. GECKO listing with no gecko_ids is rejected at validation time
//    (currently 400 per listings.go; plan describes this as 422-or-400).
func TestCreateListing_geckoWithoutGeckoIds_422(t *testing.T) {
	router, tok, _ := listingsSetup(t)

	rr := postListingReq(t, router, tok,
		`{"type":"GECKO","title":"Empty gecko listing","price_usd":"100.00","geckos":[]}`)
	// handler emits 400; accept 400 OR 422 per the plan's flexibility.
	assert.True(t, rr.Code == http.StatusBadRequest || rr.Code == http.StatusUnprocessableEntity,
		"got %d body=%s", rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "at least one gecko")
}

// 6. Negative price is rejected with a "non-negative" message.
func TestCreateListing_negativePrice_400(t *testing.T) {
	router, tok, _ := listingsSetup(t)
	sku := uniqueSku("NEG-PRICE")

	rr := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+sku+`","title":"Bad price","price_usd":"-1.00"}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code, "body=%s", rr.Body.String())
	assert.Contains(t, rr.Body.String(), "non-negative")
}

// 7. DRAFT → LISTED flips listed_at from null to non-null. LISTED → DRAFT
//    preserves listed_at (audit trail), so the timestamp remains non-null.
func TestUpdateListing_statusTransition_autoStampsListedAt(t *testing.T) {
	router, tok, pool := listingsSetup(t)
	sku := uniqueSku("DRAFT2LISTED")

	createRR := postListingReq(t, router, tok,
		`{"type":"SUPPLY","sku":"`+sku+`","title":"Timeline","price_usd":"12.00","status":"DRAFT"}`)
	require.Equal(t, http.StatusCreated, createRR.Code, "body=%s", createRR.Body.String())
	var created listingDTO
	require.NoError(t, json.Unmarshal(createRR.Body.Bytes(), &created))
	cleanupListing(t, pool, created.ID)
	assert.Equal(t, "DRAFT", created.Status)
	assert.Nil(t, created.ListedAt, "DRAFT create must not stamp listed_at")

	// DRAFT -> LISTED.
	patch1 := `{"type":"SUPPLY","sku":"` + sku + `","title":"Timeline","price_usd":"12.00","status":"LISTED"}`
	rr1 := patchListingReq(t, router, tok, created.ID, patch1)
	require.Equal(t, http.StatusOK, rr1.Code, "body=%s", rr1.Body.String())

	afterListedRR := getListingReq(t, router, tok, created.ID)
	require.Equal(t, http.StatusOK, afterListedRR.Code)
	var afterListed listingDTO
	require.NoError(t, json.Unmarshal(afterListedRR.Body.Bytes(), &afterListed))
	assert.Equal(t, "LISTED", afterListed.Status)
	require.NotNil(t, afterListed.ListedAt, "listed_at must stamp on first LISTED transition")
	firstStamp := *afterListed.ListedAt

	// LISTED -> DRAFT. listed_at is preserved (per listings.sql audit rule).
	patch2 := `{"type":"SUPPLY","sku":"` + sku + `","title":"Timeline","price_usd":"12.00","status":"DRAFT"}`
	rr2 := patchListingReq(t, router, tok, created.ID, patch2)
	require.Equal(t, http.StatusOK, rr2.Code, "body=%s", rr2.Body.String())

	afterDraftRR := getListingReq(t, router, tok, created.ID)
	require.Equal(t, http.StatusOK, afterDraftRR.Code)
	var afterDraft listingDTO
	require.NoError(t, json.Unmarshal(afterDraftRR.Body.Bytes(), &afterDraft))
	assert.Equal(t, "DRAFT", afterDraft.Status)
	require.NotNil(t, afterDraft.ListedAt, "listed_at must survive a revert to DRAFT (audit trail)")
	assert.True(t, afterDraft.ListedAt.Equal(firstStamp),
		"listed_at should be the original stamp, got %v want %v", *afterDraft.ListedAt, firstStamp)
}

// 8. Deleting a GECKO listing cascades listing_geckos via ON DELETE CASCADE.
func TestDeleteListing_cascadesJunction(t *testing.T) {
	router, tok, pool := listingsSetup(t)
	geckoID := seedGeckoForListing(t, pool, "del")

	body := `{"type":"GECKO","title":"To be deleted","price_usd":"50.00","geckos":[{"gecko_id":` + strconvItoa(int(geckoID)) + `}]}`
	createRR := postListingReq(t, router, tok, body)
	require.Equal(t, http.StatusCreated, createRR.Code, "body=%s", createRR.Body.String())
	var created listingDTO
	require.NoError(t, json.Unmarshal(createRR.Body.Bytes(), &created))
	// Safety-net cleanup in case the DELETE below doesn't actually fire.
	cleanupListing(t, pool, created.ID)

	var before int
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM listing_geckos WHERE listing_id = $1", created.ID,
	).Scan(&before))
	require.Equal(t, 1, before, "junction row must exist pre-delete")

	delRR := deleteListingReq(t, router, tok, created.ID)
	require.Equal(t, http.StatusNoContent, delRR.Code, "body=%s", delRR.Body.String())

	var after int
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM listing_geckos WHERE listing_id = $1", created.ID,
	).Scan(&after))
	assert.Equal(t, 0, after, "junction must cascade away when the listing is deleted")

	var listings int
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM listings WHERE id = $1", created.ID,
	).Scan(&listings))
	assert.Equal(t, 0, listings, "listing row should be gone")
}
