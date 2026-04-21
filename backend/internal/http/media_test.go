package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/config"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func mediaSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool, *config.Config) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := "media+" + time.Now().Format("150405.000000000") + "@example.com"
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

	cfg := &config.Config{UploadDir: filepath.Join(t.TempDir(), "uploads")}
	r := chi.NewRouter()
	MountMedia(r, pool, signer, cfg)
	return r, tok, pool, cfg
}

// Helper: insert a media row directly via sqlc (avoids testing multipart in these unit tests).
func insertTestMedia(t *testing.T, pool *pgxpool.Pool, geckoID int32, url string, order int32) int32 {
	t.Helper()
	q := db.New(pool)
	m, err := q.CreateMedia(context.Background(), db.CreateMediaParams{
		GeckoID: pgtype.Int4{Int32: geckoID, Valid: true},
		Url:     url,
		Column3: db.NullMediaType{MediaType: db.MediaTypeGALLERY, Valid: true},
		Caption: pgtype.Text{},
		Column5: order,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM media WHERE id = $1", m.ID)
	})
	return m.ID
}

// Helper: needs a gecko row (media.gecko_id FK); creates one and cleans up.
func createTestGecko(t *testing.T, pool *pgxpool.Pool) int32 {
	t.Helper()
	q := db.New(pool)
	// Pick any existing species so the FK is satisfied.
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(), "SELECT id FROM species ORDER BY id LIMIT 1").Scan(&speciesID))

	code := "TG-" + time.Now().Format("150405000")
	g, err := q.CreateGecko(context.Background(), db.CreateGeckoParams{
		Code:      code,
		SpeciesID: speciesID,
		Sex:       db.SexU,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM geckos WHERE id = $1", g.ID)
	})
	return g.ID
}

func TestPatchMedia_empty(t *testing.T) {
	router, tok, _, _ := mediaSetup(t)

	body := bytes.NewReader([]byte(`{}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/media/999999", body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "no fields to update")
}

func TestPatchMedia_captionOnly(t *testing.T) {
	router, tok, pool, _ := mediaSetup(t)
	geckoID := createTestGecko(t, pool)
	mediaID := insertTestMedia(t, pool, geckoID, "/uploads/test/a.png", 0)

	body := bytes.NewReader([]byte(`{"caption":"a new caption"}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/media/"+itoa(mediaID), body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got mediaDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, "a new caption", got.Caption)
	assert.Equal(t, int32(0), got.DisplayOrder)
}

func TestPatchMedia_displayOrderOnly(t *testing.T) {
	router, tok, pool, _ := mediaSetup(t)
	geckoID := createTestGecko(t, pool)
	mediaID := insertTestMedia(t, pool, geckoID, "/uploads/test/b.png", 5)

	body := bytes.NewReader([]byte(`{"display_order":9}`))
	req := httptest.NewRequest(http.MethodPatch, "/api/media/"+itoa(mediaID), body)
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got mediaDTO
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.Equal(t, int32(9), got.DisplayOrder)
}

func TestSetCover_reseqencesAllMedia(t *testing.T) {
	router, tok, pool, _ := mediaSetup(t)
	geckoID := createTestGecko(t, pool)
	id1 := insertTestMedia(t, pool, geckoID, "/uploads/test/1.png", 0)
	id2 := insertTestMedia(t, pool, geckoID, "/uploads/test/2.png", 1)
	id3 := insertTestMedia(t, pool, geckoID, "/uploads/test/3.png", 2)

	req := httptest.NewRequest(http.MethodPost, "/api/media/"+itoa(id3)+"/set-cover", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusNoContent, rr.Code, "body=%s", rr.Body.String())

	// After set-cover on id3: id3 → 0, id1 → 1, id2 → 2.
	q := db.New(pool)
	m1, _ := q.GetMediaByID(context.Background(), id1)
	m2, _ := q.GetMediaByID(context.Background(), id2)
	m3, _ := q.GetMediaByID(context.Background(), id3)
	assert.Equal(t, int32(0), m3.DisplayOrder, "target becomes display_order=0")
	assert.Equal(t, int32(1), m1.DisplayOrder, "former first shifts to 1")
	assert.Equal(t, int32(2), m2.DisplayOrder, "former second shifts to 2")
}

func TestSetCover_notFound(t *testing.T) {
	router, tok, _, _ := mediaSetup(t)

	req := httptest.NewRequest(http.MethodPost, "/api/media/999999/set-cover", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

// itoa keeps the test imports light.
func itoa(n int32) string {
	return strconvItoa(int(n))
}
