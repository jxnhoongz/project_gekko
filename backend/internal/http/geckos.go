package http

import (
	"net/http"
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
	ListPriceUsd  *string       `json:"list_price_usd"`
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

	total, err := d.q.CountGeckos(ctx)
	if err != nil {
		total = int64(len(rows))
	}

	out := make([]geckoDTO, 0, len(rows))
	for _, g := range rows {
		cover := d.loadCover(ctx, g.ID)
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
			ListPriceUsd:  numericOrNil(g.ListPriceUsd),
			Notes:         textOrEmpty(g.Notes),
			CreatedAt:     g.CreatedAt.Time,
			Traits:        genesByGecko[g.ID],
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
		ListPriceUsd:  numericOrNil(row.ListPriceUsd),
		Notes:         textOrEmpty(row.Notes),
		CreatedAt:     row.CreatedAt.Time,
		Traits:        traitsOut,
		CoverPhotoUrl: cover,
		Photos:        photosOut,
	}
	writeJSON(w, http.StatusOK, out)
}

func (d *geckosDeps) loadCover(ctx any, id int32) *string {
	rctx, ok := ctx.(interface {
		Done() <-chan struct{}
	})
	_ = rctx
	_ = ok
	return nil // left as placeholder; cover is fetched in getGecko only
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
