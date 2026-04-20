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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// testSetup returns a router + a freshly-created admin row. Cleans up after.
func testSetup(t *testing.T) (http.Handler, string, string) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")

	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := "test+" + time.Now().Format("150405.000000000") + "@example.com"
	plain := "test-password-123"
	hash, err := auth.HashPassword(plain)
	require.NoError(t, err)

	q := db.New(pool)
	created, err := q.CreateAdmin(context.Background(), db.CreateAdminParams{
		Email:        email,
		PasswordHash: hash,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM admin_users WHERE id = $1", created.ID)
	})

	signer := auth.NewJWTSigner("test-secret", time.Hour)
	router := NewAuthRouter(pool, signer)
	return router, email, plain
}

func TestLogin_happyPath(t *testing.T) {
	r, email, password := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "body=%s", rr.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &got))
	assert.NotEmpty(t, got["token"])
	admin, _ := got["admin"].(map[string]any)
	assert.Equal(t, email, admin["email"])
}

func TestLogin_wrongPassword(t *testing.T) {
	r, email, _ := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": email, "password": "nope"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestLogin_unknownEmail(t *testing.T) {
	r, _, _ := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": "nobody+xyz@example.com", "password": "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestMe_withToken(t *testing.T) {
	r, email, password := testSetup(t)

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRR := httptest.NewRecorder()
	r.ServeHTTP(loginRR, loginReq)
	require.Equal(t, http.StatusOK, loginRR.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(loginRR.Body.Bytes(), &payload))
	tok := payload["token"].(string)

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+tok)
	meRR := httptest.NewRecorder()
	r.ServeHTTP(meRR, meReq)

	require.Equal(t, http.StatusOK, meRR.Code, "body=%s", meRR.Body.String())
	var me map[string]any
	require.NoError(t, json.Unmarshal(meRR.Body.Bytes(), &me))
	assert.Equal(t, email, me["email"])
}

func TestMe_missingToken(t *testing.T) {
	r, _, _ := testSetup(t)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
