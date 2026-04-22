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

func morphCombosSetup(t *testing.T) (http.Handler, string, *pgxpool.Pool) {
	t.Helper()
	_ = godotenv.Load("../../.env.local")
	pool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	email := fmt.Sprintf("morphcombos+%d@example.com", time.Now().UnixNano())
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
	MountMorphCombos(r, pool, signer)
	return r, tok, pool
}

func TestMorphCombos_CRUD(t *testing.T) {
	handler, tok, pool := morphCombosSetup(t)

	// Resolve LP species ID.
	var speciesID int32
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM species WHERE code = 'LP'").Scan(&speciesID))

	// Resolve two existing trait IDs (Tremper Albino + Eclipse).
	var tremperID, eclipseID int32
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM genetic_dictionary WHERE species_id = $1 AND trait_name = 'Tremper Albino'",
		speciesID).Scan(&tremperID))
	require.NoError(t, pool.QueryRow(context.Background(),
		"SELECT id FROM genetic_dictionary WHERE species_id = $1 AND trait_name = 'Eclipse'",
		speciesID).Scan(&eclipseID))

	// Use a nanosecond-unique code to avoid conflicts with seeded combos.
	code := fmt.Sprintf("TC%07d", time.Now().UnixNano()%10_000_000)
	body, _ := json.Marshal(map[string]any{
		"species_id":  speciesID,
		"name":        fmt.Sprintf("Test Combo %d", time.Now().UnixNano()),
		"code":        code,
		"description": "test",
		"requirements": []map[string]any{
			{"trait_id": tremperID, "required_zygosity": "HOM"},
			{"trait_id": eclipseID, "required_zygosity": "HOM"},
		},
	})

	// Create
	req := httptest.NewRequest(http.MethodPost, "/api/morph-combos", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code, "create body=%s", w.Body.String())

	var created morphComboDTO
	require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
	assert.Equal(t, 2, len(created.Requirements))
	comboID := created.ID
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM morph_combos WHERE id = $1", comboID)
	})

	// Get
	req2 := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/morph-combos/%d", comboID), nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code, "get body=%s", w2.Body.String())

	// Update — remove one requirement
	upd, _ := json.Marshal(map[string]any{
		"species_id":   speciesID,
		"name":         created.Name,
		"code":         code + "2",
		"requirements": []map[string]any{{"trait_id": tremperID, "required_zygosity": "HOM"}},
	})
	req3 := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/morph-combos/%d", comboID), bytes.NewReader(upd))
	req3.Header.Set("Authorization", "Bearer "+tok)
	req3.Header.Set("Content-Type", "application/json")
	w3 := httptest.NewRecorder()
	handler.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code, "update body=%s", w3.Body.String())
	var updated morphComboDTO
	require.NoError(t, json.NewDecoder(w3.Body).Decode(&updated))
	assert.Equal(t, 1, len(updated.Requirements))

	// Delete
	req4 := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/morph-combos/%d", comboID), nil)
	req4.Header.Set("Authorization", "Bearer "+tok)
	w4 := httptest.NewRecorder()
	handler.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusNoContent, w4.Code, "delete body=%s", w4.Body.String())
}

func TestMorphCombos_List_BySpeciesCode(t *testing.T) {
	handler, tok, _ := morphCombosSetup(t)

	req := httptest.NewRequest(http.MethodGet, "/api/morph-combos?species_code=LP", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp morphCombosListResp
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.GreaterOrEqual(t, len(resp.Combos), 5) // at least the 5 seeded combos
}
