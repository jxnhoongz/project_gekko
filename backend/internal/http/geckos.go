package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountGeckos registers admin-only read endpoints for species, traits, geckos.
func MountGeckos(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &geckosDeps{pool: pool, q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/species", d.listSpecies)
		pr.Get("/api/traits", d.listTraits)
		pr.Get("/api/geckos", d.listGeckos)
		pr.Get("/api/geckos/{id}", d.getGecko)
		pr.Post("/api/geckos", d.createGecko)
		pr.Patch("/api/geckos/{id}", d.updateGecko)
		pr.Delete("/api/geckos/{id}", d.deleteGecko)
	})
}

type geckosDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// ---- DTOs ----

type speciesDTO struct {
	ID             int32  `json:"id"`
	Code           string `json:"code"`
	CommonName     string `json:"common_name"`
	ScientificName string `json:"scientific_name"`
	Description    string `json:"description"`
}

type traitDTO struct {
	ID          int32  `json:"id"`
	SpeciesID   int32  `json:"species_id"`
	TraitName   string `json:"trait_name"`
	TraitCode   string `json:"trait_code"`
	Description string `json:"description"`
	IsDominant  bool   `json:"is_dominant"`
}

type geckoGeneDTO struct {
	TraitID   int32  `json:"trait_id"`
	TraitName string `json:"trait_name"`
	TraitCode string `json:"trait_code"`
	Zygosity  string `json:"zygosity"`
	IsDominant bool  `json:"is_dominant"`
}

type mediaDTO struct {
	ID           int32  `json:"id"`
	Url          string `json:"url"`
	Type         string `json:"type"`
	Caption      string `json:"caption"`
	DisplayOrder int32  `json:"display_order"`
}

type geckoDTO struct {
	ID            int32         `json:"id"`
	Code          string        `json:"code"`
	Name          string        `json:"name"`
	SpeciesID     int32         `json:"species_id"`
	SpeciesCode   string        `json:"species_code"`
	SpeciesName   string        `json:"species_name"`
	Sex           string        `json:"sex"`
	HatchDate     *string       `json:"hatch_date"`
	AcquiredDate  *string       `json:"acquired_date"`
	Status        string        `json:"status"`
	SireID        *int32        `json:"sire_id"`
	DamID         *int32        `json:"dam_id"`
	Notes         string        `json:"notes"`
	CreatedAt     time.Time     `json:"created_at"`
	Traits        []geckoGeneDTO `json:"traits"`
	CoverPhotoUrl *string       `json:"cover_photo_url"`
	Photos        []mediaDTO    `json:"photos,omitempty"`
}

type listGeckosResp struct {
	Geckos []geckoDTO `json:"geckos"`
	Total  int64      `json:"total"`
}

// ---- handlers ----

func (d *geckosDeps) listSpecies(w http.ResponseWriter, r *http.Request) {
	rows, err := d.q.ListSpecies(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list species failed"})
		return
	}
	out := make([]speciesDTO, 0, len(rows))
	for _, s := range rows {
		out = append(out, speciesDTO{
			ID:             s.ID,
			Code:           string(s.Code),
			CommonName:     s.CommonName,
			ScientificName: textOrEmpty(s.ScientificName),
			Description:    textOrEmpty(s.Description),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"species": out})
}

func (d *geckosDeps) listTraits(w http.ResponseWriter, r *http.Request) {
	speciesIDParam := r.URL.Query().Get("species_id")
	var traits []db.GeneticDictionary
	var err error

	if speciesIDParam != "" {
		n, perr := strconv.Atoi(speciesIDParam)
		if perr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid species_id"})
			return
		}
		traits, err = d.q.ListTraitsBySpecies(r.Context(), int32(n))
	} else {
		traits, err = d.q.ListTraits(r.Context())
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list traits failed"})
		return
	}

	out := make([]traitDTO, 0, len(traits))
	for _, t := range traits {
		out = append(out, traitDTO{
			ID:          t.ID,
			SpeciesID:   t.SpeciesID,
			TraitName:   t.TraitName,
			TraitCode:   textOrEmpty(t.TraitCode),
			Description: textOrEmpty(t.Description),
			IsDominant:  t.IsDominant,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"traits": out})
}

func (d *geckosDeps) listGeckos(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := d.q.ListGeckos(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list geckos failed"})
		return
	}

	// Preload all genes in one shot, group by gecko_id.
	allGenes, err := d.q.ListGeckoGenes(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list genes failed"})
		return
	}
	genesByGecko := map[int32][]geckoGeneDTO{}
	for _, g := range allGenes {
		genesByGecko[g.GeckoID] = append(genesByGecko[g.GeckoID], geckoGeneDTO{
			TraitID:    g.TraitID,
			TraitName:  g.TraitName,
			TraitCode:  textOrEmpty(g.TraitCode),
			Zygosity:   string(g.Zygosity),
			IsDominant: g.IsDominant,
		})
	}

	// Preload cover photos (first media per gecko) in one query.
	covers, err := d.q.ListCoverMediaForGeckos(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list covers failed"})
		return
	}
	coverByGecko := map[int32]string{}
	for _, c := range covers {
		if c.GeckoID.Valid {
			coverByGecko[c.GeckoID.Int32] = c.Url
		}
	}

	total, err := d.q.CountGeckos(ctx)
	if err != nil {
		total = int64(len(rows))
	}

	out := make([]geckoDTO, 0, len(rows))
	for _, g := range rows {
		var cover *string
		if u, ok := coverByGecko[g.ID]; ok {
			cover = &u
		}
		// Always emit an array (never null) so the frontend can iterate safely.
		traits := genesByGecko[g.ID]
		if traits == nil {
			traits = []geckoGeneDTO{}
		}
		out = append(out, geckoDTO{
			ID:            g.ID,
			Code:          g.Code,
			Name:          textOrEmpty(g.Name),
			SpeciesID:     g.SpeciesID,
			SpeciesCode:   string(g.SpeciesCode),
			SpeciesName:   g.SpeciesCommonName,
			Sex:           string(g.Sex),
			HatchDate:     dateOrNil(g.HatchDate),
			AcquiredDate:  dateOrNil(g.AcquiredDate),
			Status:        string(g.Status),
			SireID:        int32OrNil(g.SireID),
			DamID:         int32OrNil(g.DamID),
			Notes:         textOrEmpty(g.Notes),
			CreatedAt:     g.CreatedAt.Time,
			Traits:        traits,
			CoverPhotoUrl: cover,
		})
	}
	writeJSON(w, http.StatusOK, listGeckosResp{Geckos: out, Total: total})
}

func (d *geckosDeps) getGecko(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id64, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	ctx := r.Context()
	row, err := d.q.GetGeckoByID(ctx, int32(id64))
	if err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}

	genes, err := d.q.ListGenesForGecko(ctx, row.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch genes failed"})
		return
	}
	traitsOut := make([]geckoGeneDTO, 0, len(genes))
	for _, g := range genes {
		traitsOut = append(traitsOut, geckoGeneDTO{
			TraitID:    g.TraitID,
			TraitName:  g.TraitName,
			TraitCode:  textOrEmpty(g.TraitCode),
			Zygosity:   string(g.Zygosity),
			IsDominant: g.IsDominant,
		})
	}

	photos, err := d.q.ListMediaForGecko(ctx, pgtype.Int4{Int32: row.ID, Valid: true})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch media failed"})
		return
	}
	photosOut := make([]mediaDTO, 0, len(photos))
	for _, p := range photos {
		photosOut = append(photosOut, mediaDTO{
			ID:           p.ID,
			Url:          p.Url,
			Type:         string(p.Type),
			Caption:      textOrEmpty(p.Caption),
			DisplayOrder: p.DisplayOrder,
		})
	}
	var cover *string
	if len(photosOut) > 0 {
		cover = &photosOut[0].Url
	}

	out := geckoDTO{
		ID:            row.ID,
		Code:          row.Code,
		Name:          textOrEmpty(row.Name),
		SpeciesID:     row.SpeciesID,
		SpeciesCode:   string(row.SpeciesCode),
		SpeciesName:   row.SpeciesCommonName,
		Sex:           string(row.Sex),
		HatchDate:     dateOrNil(row.HatchDate),
		AcquiredDate:  dateOrNil(row.AcquiredDate),
		Status:        string(row.Status),
		SireID:        int32OrNil(row.SireID),
		DamID:         int32OrNil(row.DamID),
		Notes:         textOrEmpty(row.Notes),
		CreatedAt:     row.CreatedAt.Time,
		Traits:        traitsOut,
		CoverPhotoUrl: cover,
		Photos:        photosOut,
	}
	writeJSON(w, http.StatusOK, out)
}

// helpers

func dateOrNil(d pgtype.Date) *string {
	if !d.Valid {
		return nil
	}
	s := d.Time.Format("2006-01-02")
	return &s
}

func int32OrNil(v pgtype.Int4) *int32 {
	if !v.Valid {
		return nil
	}
	n := v.Int32
	return &n
}

func numericOrNil(n pgtype.Numeric) *string {
	if !n.Valid {
		return nil
	}
	bs, err := n.MarshalJSON()
	if err != nil {
		return nil
	}
	s := string(bs)
	// pgtype.Numeric marshals as a JSON string like "180" — already clean.
	// If it's quoted, strip quotes.
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return &s
}

// ---- write handlers ----

type geckoTraitInput struct {
	TraitID  int32  `json:"trait_id"`
	Zygosity string `json:"zygosity"`
}

type createGeckoReq struct {
	Name         string            `json:"name"`
	SpeciesID    int32             `json:"species_id"`
	Sex          string            `json:"sex"`
	HatchDate    string            `json:"hatch_date"`
	AcquiredDate string            `json:"acquired_date"`
	Status       string            `json:"status"`
	SireID       *int32            `json:"sire_id"`
	DamID        *int32            `json:"dam_id"`
	Notes        string            `json:"notes"`
	Traits       []geckoTraitInput `json:"traits"`
}

type updateGeckoReq = createGeckoReq

var (
	validSex    = map[string]db.Sex{"M": db.SexM, "F": db.SexF, "U": db.SexU}
	validStatus = map[string]db.GeckoStatus{
		"AVAILABLE": db.GeckoStatusAVAILABLE,
		"HOLD":      db.GeckoStatusHOLD,
		"BREEDING":  db.GeckoStatusBREEDING,
		"PERSONAL":  db.GeckoStatusPERSONAL,
		"SOLD":      db.GeckoStatusSOLD,
		"DECEASED":  db.GeckoStatusDECEASED,
	}
	validZygosity = map[string]db.Zygosity{
		"HOM":      db.ZygosityHOM,
		"HET":      db.ZygosityHET,
		"POSS_HET": db.ZygosityPOSSHET,
	}
)

func parseDate(s string) (pgtype.Date, error) {
	if s == "" {
		return pgtype.Date{Valid: false}, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}

func parseNumeric(s string) (pgtype.Numeric, error) {
	if s == "" {
		return pgtype.Numeric{Valid: false}, nil
	}
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return pgtype.Numeric{}, err
	}
	return n, nil
}

func pgInt4(p *int32) pgtype.Int4 {
	if p == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *p, Valid: true}
}

func pgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// generateCode returns the next unused code for the given species code + year.
// Format: ZG<SP>-<YYYY>-<NNN> (e.g. ZGLP-2026-007).
func (d *geckosDeps) generateCode(ctx context.Context, q *db.Queries, speciesCode string, year int) (string, error) {
	pattern := fmt.Sprintf("ZG%s-%d-%%", speciesCode, year)
	next, err := q.NextGeckoSequenceForSpeciesYear(ctx, pattern)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ZG%s-%d-%03d", speciesCode, year, next), nil
}

func (d *geckosDeps) createGecko(w http.ResponseWriter, r *http.Request) {
	var req createGeckoReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.SpeciesID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "species_id required"})
		return
	}

	sex, ok := validSex[req.Sex]
	if !ok {
		if req.Sex == "" {
			sex = db.SexU
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sex"})
			return
		}
	}

	var statusCol db.NullGeckoStatus
	if req.Status != "" {
		st, ok := validStatus[req.Status]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
		statusCol = db.NullGeckoStatus{GeckoStatus: st, Valid: true}
	}

	hatchDate, err := parseDate(req.HatchDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hatch_date"})
		return
	}
	acquiredDate, err := parseDate(req.AcquiredDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid acquired_date"})
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

	sp, err := qtx.GetSpeciesByID(ctx, req.SpeciesID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "species not found"})
		return
	}

	code, err := d.generateCode(ctx, qtx, string(sp.Code), time.Now().Year())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "code gen failed: " + err.Error()})
		return
	}

	gecko, err := qtx.CreateGecko(ctx, db.CreateGeckoParams{
		Code:         code,
		Name:         pgText(req.Name),
		SpeciesID:    req.SpeciesID,
		Sex:          sex,
		HatchDate:    hatchDate,
		AcquiredDate: acquiredDate,
		Column7:      statusCol,
		SireID:       pgInt4(req.SireID),
		DamID:        pgInt4(req.DamID),
		Notes:        pgText(req.Notes),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create failed: " + err.Error()})
		return
	}

	if err := applyTraits(ctx, qtx, gecko.ID, req.Traits); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}

	// Return full detail
	r2 := *r
	r2.URL = &url.URL{Path: "/api/geckos/" + strconv.Itoa(int(gecko.ID))}
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(gecko.ID)))
	r2 = *r2.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	d.getGecko(w, &r2)
}

func (d *geckosDeps) updateGecko(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id64, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req updateGeckoReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if req.SpeciesID == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "species_id required"})
		return
	}

	sex, ok := validSex[req.Sex]
	if !ok {
		if req.Sex == "" {
			sex = db.SexU
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid sex"})
			return
		}
	}

	status, ok := validStatus[req.Status]
	if !ok {
		if req.Status == "" {
			status = db.GeckoStatusAVAILABLE
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
	}

	hatchDate, err := parseDate(req.HatchDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid hatch_date"})
		return
	}
	acquiredDate, err := parseDate(req.AcquiredDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid acquired_date"})
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

	updated, err := qtx.UpdateGecko(ctx, db.UpdateGeckoParams{
		ID:           int32(id64),
		Name:         pgText(req.Name),
		SpeciesID:    req.SpeciesID,
		Sex:          sex,
		HatchDate:    hatchDate,
		AcquiredDate: acquiredDate,
		Status:       status,
		SireID:       pgInt4(req.SireID),
		DamID:        pgInt4(req.DamID),
		Notes:        pgText(req.Notes),
	})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found or update failed"})
		return
	}

	// Replace trait set
	if err := qtx.DeleteGenesForGecko(ctx, updated.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "clear genes failed"})
		return
	}
	if err := applyTraits(ctx, qtx, updated.ID, req.Traits); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(updated.ID)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	d.getGecko(w, r2)
}

func (d *geckosDeps) deleteGecko(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id64, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := d.q.DeleteGecko(r.Context(), int32(id64)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func applyTraits(ctx context.Context, q *db.Queries, geckoID int32, traits []geckoTraitInput) error {
	for _, t := range traits {
		zyg, ok := validZygosity[t.Zygosity]
		if !ok {
			return fmt.Errorf("invalid zygosity %q", t.Zygosity)
		}
		if _, err := q.CreateGeckoGene(ctx, db.CreateGeckoGeneParams{
			GeckoID:  geckoID,
			TraitID:  t.TraitID,
			Zygosity: zyg,
		}); err != nil {
			return fmt.Errorf("assign trait %d: %w", t.TraitID, err)
		}
	}
	return nil
}
