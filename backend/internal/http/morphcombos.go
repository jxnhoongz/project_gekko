package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountMorphCombos registers admin-only CRUD for morph combos.
func MountMorphCombos(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &morphCombosDeps{pool: pool, q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/morph-combos", d.list)
		pr.Post("/api/morph-combos", d.create)
		pr.Get("/api/morph-combos/{id}", d.get)
		pr.Patch("/api/morph-combos/{id}", d.update)
		pr.Delete("/api/morph-combos/{id}", d.delete)
	})
}

type morphCombosDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// ---- DTOs ----

type morphComboTraitDTO struct {
	TraitID          int32  `json:"trait_id"`
	TraitName        string `json:"trait_name"`
	TraitCode        string `json:"trait_code"`
	RequiredZygosity string `json:"required_zygosity"`
}

type morphComboDTO struct {
	ID              int32                `json:"id"`
	SpeciesID       int32                `json:"species_id"`
	Name            string               `json:"name"`
	Code            string               `json:"code"`
	Description     string               `json:"description"`
	Notes           string               `json:"notes"`
	ExamplePhotoUrl string               `json:"example_photo_url"`
	Requirements    []morphComboTraitDTO `json:"requirements"`
}

type morphCombosListResp struct {
	Combos []morphComboDTO `json:"combos"`
	Total  int             `json:"total"`
}

// ---- requests ----

type morphComboTraitInput struct {
	TraitID          int32  `json:"trait_id"`
	RequiredZygosity string `json:"required_zygosity"`
}

type createMorphComboReq struct {
	SpeciesID       int32                  `json:"species_id"`
	Name            string                 `json:"name"`
	Code            string                 `json:"code"`
	Description     string                 `json:"description"`
	Notes           string                 `json:"notes"`
	ExamplePhotoUrl string                 `json:"example_photo_url"`
	Requirements    []morphComboTraitInput `json:"requirements"`
}

type updateMorphComboReq = createMorphComboReq

// ---- handlers ----

func (d *morphCombosDeps) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var rows []db.MorphCombo
	var err error

	if sc := r.URL.Query().Get("species_code"); sc != "" {
		var speciesID int32
		if err2 := d.pool.QueryRow(ctx,
			"SELECT id FROM species WHERE code = $1", sc).Scan(&speciesID); err2 != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown species_code"})
			return
		}
		rows, err = d.q.ListMorphCombosBySpecies(ctx, speciesID)
	} else {
		rows, err = d.q.ListMorphCombos(ctx)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}

	// Preload all requirements in one query.
	ids := make([]int32, len(rows))
	for i, mc := range rows {
		ids[i] = mc.ID
	}
	traitRows, err := d.q.ListMorphComboTraits(ctx, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list traits failed"})
		return
	}
	reqsByCombo := map[int32][]morphComboTraitDTO{}
	for _, t := range traitRows {
		reqsByCombo[t.ComboID] = append(reqsByCombo[t.ComboID], morphComboTraitDTO{
			TraitID:          t.TraitID,
			TraitName:        t.TraitName,
			TraitCode:        textOrEmpty(t.TraitCode),
			RequiredZygosity: string(t.RequiredZygosity),
		})
	}

	out := make([]morphComboDTO, 0, len(rows))
	for _, mc := range rows {
		reqs := reqsByCombo[mc.ID]
		if reqs == nil {
			reqs = []morphComboTraitDTO{}
		}
		out = append(out, morphComboDTO{
			ID:              mc.ID,
			SpeciesID:       mc.SpeciesID,
			Name:            mc.Name,
			Code:            textOrEmpty(mc.Code),
			Description:     textOrEmpty(mc.Description),
			Notes:           textOrEmpty(mc.Notes),
			ExamplePhotoUrl: textOrEmpty(mc.ExamplePhotoUrl),
			Requirements:    reqs,
		})
	}
	writeJSON(w, http.StatusOK, morphCombosListResp{Combos: out, Total: len(out)})
}

func (d *morphCombosDeps) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ctx := r.Context()
	mc, err := d.q.GetMorphCombo(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}
	traitRows, err := d.q.ListMorphComboTraits(ctx, []int32{mc.ID})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch traits failed"})
		return
	}
	reqs := make([]morphComboTraitDTO, 0, len(traitRows))
	for _, t := range traitRows {
		reqs = append(reqs, morphComboTraitDTO{
			TraitID:          t.TraitID,
			TraitName:        t.TraitName,
			TraitCode:        textOrEmpty(t.TraitCode),
			RequiredZygosity: string(t.RequiredZygosity),
		})
	}
	writeJSON(w, http.StatusOK, morphComboDTO{
		ID: mc.ID, SpeciesID: mc.SpeciesID, Name: mc.Name,
		Code: textOrEmpty(mc.Code), Description: textOrEmpty(mc.Description),
		Notes: textOrEmpty(mc.Notes), ExamplePhotoUrl: textOrEmpty(mc.ExamplePhotoUrl),
		Requirements: reqs,
	})
}

func (d *morphCombosDeps) create(w http.ResponseWriter, r *http.Request) {
	var req createMorphComboReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.SpeciesID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and species_id required"})
		return
	}
	ctx := r.Context()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	mc, err := qtx.CreateMorphCombo(ctx, db.CreateMorphComboParams{
		SpeciesID:       req.SpeciesID,
		Name:            req.Name,
		Code:            pgText(req.Code),
		Description:     pgText(req.Description),
		Notes:           pgText(req.Notes),
		ExamplePhotoUrl: pgText(req.ExamplePhotoUrl),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create failed: " + err.Error()})
		return
	}
	if err := applyComboTraits(ctx, qtx, mc.ID, req.Requirements); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}
	// Re-fetch to return full response (with trait names).
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(mc.ID)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	sr := &statusRecorder{ResponseWriter: w, status: http.StatusCreated}
	d.get(sr, r2)
}

func (d *morphCombosDeps) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req updateMorphComboReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.Name == "" || req.SpeciesID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and species_id required"})
		return
	}
	ctx := r.Context()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	mc, err := qtx.UpdateMorphCombo(ctx, db.UpdateMorphComboParams{
		ID:              int32(id),
		Name:            req.Name,
		Code:            pgText(req.Code),
		Description:     pgText(req.Description),
		Notes:           pgText(req.Notes),
		ExamplePhotoUrl: pgText(req.ExamplePhotoUrl),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	if err := qtx.DeleteMorphComboTraits(ctx, mc.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clear traits failed"})
		return
	}
	if err := applyComboTraits(ctx, qtx, mc.ID, req.Requirements); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(mc.ID)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	d.get(w, r2)
}

func (d *morphCombosDeps) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := d.q.DeleteMorphCombo(r.Context(), int32(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyComboTraits inserts junction rows; validates zygosity.
func applyComboTraits(ctx context.Context, q *db.Queries, comboID int32, reqs []morphComboTraitInput) error {
	for _, req := range reqs {
		zyg, ok := validZygosity[req.RequiredZygosity]
		if !ok {
			return fmt.Errorf("invalid required_zygosity %q", req.RequiredZygosity)
		}
		if err := q.InsertMorphComboTrait(ctx, db.InsertMorphComboTraitParams{
			ComboID:          comboID,
			TraitID:          req.TraitID,
			RequiredZygosity: zyg,
		}); err != nil {
			return fmt.Errorf("insert trait %d: %w", req.TraitID, err)
		}
	}
	return nil
}
