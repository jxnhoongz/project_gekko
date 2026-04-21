package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountPublic mounts the storefront-facing endpoints (no auth). One endpoint
// is wrapped in a per-IP rate limiter (waitlist signup). The list + detail
// endpoints are left open — they're trivially cacheable and low-risk.
func MountPublic(r chi.Router, pool *pgxpool.Pool) {
	d := &publicDeps{pool: pool, q: db.New(pool)}
	waitlistLimiter := NewIPRateLimiter(5, time.Hour)

	r.Get("/api/public/geckos", d.listAvailable)
	r.Get("/api/public/geckos/{code}", d.getByCode)
	r.With(waitlistLimiter.Middleware).Post("/api/public/waitlist", d.createWaitlist)
}

type publicDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// ---- public DTOs (strict subset; no notes/sire/dam/status/acquired_date) ----

type publicGeckoDTO struct {
	Code          string  `json:"code"`
	Name          string  `json:"name"`
	SpeciesCode   string  `json:"species_code"`
	SpeciesName   string  `json:"species_name"`
	Morph         string  `json:"morph"`
	Sex           string  `json:"sex"`
	HatchDate     *string `json:"hatch_date"`
	ListPriceUsd  *string `json:"list_price_usd"`
	CoverPhotoUrl *string `json:"cover_photo_url"`
}

type publicGeckoListResp struct {
	Geckos []publicGeckoDTO `json:"geckos"`
	Total  int              `json:"total"`
}

type publicMediaDTO struct {
	Url          string `json:"url"`
	Caption      string `json:"caption"`
	DisplayOrder int32  `json:"display_order"`
}

type publicGeckoDetailDTO struct {
	publicGeckoDTO
	Photos []publicMediaDTO `json:"photos"`
}

// ---- list ----

func (d *publicDeps) listAvailable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := d.q.ListAvailableGeckos(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}

	ids := make([]int32, 0, len(rows))
	for _, g := range rows {
		ids = append(ids, g.ID)
	}

	// Preload traits + covers in one round trip each.
	genes, err := d.q.ListPublicGenesByGeckoIDs(ctx, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list genes failed"})
		return
	}
	genesByGecko := map[int32][]db.ListPublicGenesByGeckoIDsRow{}
	for _, g := range genes {
		genesByGecko[g.GeckoID] = append(genesByGecko[g.GeckoID], g)
	}

	covers, err := d.q.ListPublicMediaByGeckoIDs(ctx, ids)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list covers failed"})
		return
	}
	coverByGecko := map[int32]string{}
	for _, c := range covers {
		coverByGecko[c.GeckoID.Int32] = c.Url
	}

	out := make([]publicGeckoDTO, 0, len(rows))
	for _, g := range rows {
		out = append(out, publicGeckoDTO{
			Code:          g.Code,
			Name:          textOrEmpty(g.Name),
			SpeciesCode:   string(g.SpeciesCode),
			SpeciesName:   g.SpeciesCommonName,
			Morph:         composePublicMorph(genesByGecko[g.ID]),
			Sex:           string(g.Sex),
			HatchDate:     dateOrNil(g.HatchDate),
			ListPriceUsd:  numericOrNil(g.ListPriceUsd),
			CoverPhotoUrl: coverPtr(coverByGecko, g.ID),
		})
	}

	writeJSON(w, http.StatusOK, publicGeckoListResp{Geckos: out, Total: len(out)})
}

// ---- detail ----

func (d *publicDeps) getByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "code required"})
		return
	}

	ctx := r.Context()
	row, err := d.q.GetAvailableGeckoByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}

	genes, err := d.q.ListPublicGenesByGeckoIDs(ctx, []int32{row.ID})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list genes failed"})
		return
	}

	photos, err := d.q.ListMediaForGecko(ctx, pgtype.Int4{Int32: row.ID, Valid: true})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list media failed"})
		return
	}
	photosOut := make([]publicMediaDTO, 0, len(photos))
	for _, p := range photos {
		photosOut = append(photosOut, publicMediaDTO{
			Url:          p.Url,
			Caption:      textOrEmpty(p.Caption),
			DisplayOrder: p.DisplayOrder,
		})
	}

	var cover *string
	if len(photosOut) > 0 {
		cover = &photosOut[0].Url
	}

	writeJSON(w, http.StatusOK, publicGeckoDetailDTO{
		publicGeckoDTO: publicGeckoDTO{
			Code:          row.Code,
			Name:          textOrEmpty(row.Name),
			SpeciesCode:   string(row.SpeciesCode),
			SpeciesName:   row.SpeciesCommonName,
			Morph:         composePublicMorph(genes),
			Sex:           string(row.Sex),
			HatchDate:     dateOrNil(row.HatchDate),
			ListPriceUsd:  numericOrNil(row.ListPriceUsd),
			CoverPhotoUrl: cover,
		},
		Photos: photosOut,
	})
}

// ---- waitlist ----

type publicWaitlistReq struct {
	Email        string `json:"email"`
	Telegram     string `json:"telegram"`
	Phone        string `json:"phone"`
	InterestedIn string `json:"interested_in"`
	Notes        string `json:"notes"`
}

type publicWaitlistResp struct {
	ID           *int32 `json:"id,omitempty"`
	Deduplicated bool   `json:"deduplicated,omitempty"`
}

var emailRE = regexp.MustCompile(`^.+@.+\..+$`)

func (d *publicDeps) createWaitlist(w http.ResponseWriter, r *http.Request) {
	var req publicWaitlistReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Telegram = strings.TrimSpace(req.Telegram)
	req.Phone = strings.TrimSpace(req.Phone)
	req.InterestedIn = strings.TrimSpace(req.InterestedIn)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.Email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "email is required"})
		return
	}
	if !emailRE.MatchString(req.Email) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid email"})
		return
	}
	if len(req.Email) > 255 || len(req.Telegram) > 100 || len(req.Phone) > 32 ||
		len(req.InterestedIn) > 100 || len(req.Notes) > 2000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "field too long"})
		return
	}

	row, err := d.q.CreateWaitlistEntry(r.Context(), db.CreateWaitlistEntryParams{
		Email:        req.Email,
		Telegram:     pgText(req.Telegram),
		Phone:        pgText(req.Phone),
		InterestedIn: pgText(req.InterestedIn),
		Column5:      "storefront",
		Notes:        pgText(req.Notes),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			writeJSON(w, http.StatusOK, publicWaitlistResp{Deduplicated: true})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}

	id := row.ID
	writeJSON(w, http.StatusCreated, publicWaitlistResp{ID: &id})
}

// ---- helpers ----

// composePublicMorph mirrors admin's morphFromTraits in Go so the public
// storefront stays free of genetics business logic.
func composePublicMorph(rows []db.ListPublicGenesByGeckoIDsRow) string {
	if len(rows) == 0 {
		return "Normal"
	}
	var hom, het, poss []string
	for _, r := range rows {
		switch r.Zygosity {
		case db.ZygosityHOM:
			hom = append(hom, r.TraitName)
		case db.ZygosityHET:
			het = append(het, r.TraitName)
		case db.ZygosityPOSSHET:
			poss = append(poss, r.TraitName)
		}
	}
	parts := []string{}
	if len(hom) > 0 {
		parts = append(parts, strings.Join(hom, " "))
	}
	if len(het) > 0 {
		prefixed := make([]string, len(het))
		for i, n := range het {
			prefixed[i] = "het " + n
		}
		parts = append(parts, strings.Join(prefixed, " "))
	}
	if len(poss) > 0 {
		prefixed := make([]string, len(poss))
		for i, n := range poss {
			prefixed[i] = "poss. het " + n
		}
		parts = append(parts, strings.Join(prefixed, " "))
	}
	if len(parts) == 0 {
		return "Normal"
	}
	return strings.Join(parts, ", ")
}

func coverPtr(m map[int32]string, id int32) *string {
	if u, ok := m[id]; ok {
		return &u
	}
	return nil
}
