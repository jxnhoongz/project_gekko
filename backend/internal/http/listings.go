package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

// MountListings registers admin-only /api/listings CRUD.
func MountListings(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner) {
	d := &listingsDeps{pool: pool, q: db.New(pool)}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/listings", d.list)
		pr.Post("/api/listings", d.create)
		pr.Get("/api/listings/{id}", d.get)
		pr.Patch("/api/listings/{id}", d.update)
		pr.Delete("/api/listings/{id}", d.delete)
	})
}

type listingsDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

// ---- DTOs ----

type listingGeckoRefDTO struct {
	GeckoID     int32  `json:"gecko_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	SpeciesCode string `json:"species_code"`
}

type listingComponentRefDTO struct {
	ComponentListingID int32  `json:"component_listing_id"`
	Title              string `json:"title"`
	Type               string `json:"type"`
	PriceUsd           string `json:"price_usd"`
	Quantity           int32  `json:"quantity"`
}

type listingDTO struct {
	ID             int32                    `json:"id"`
	Sku            string                   `json:"sku"`
	Type           string                   `json:"type"`
	Title          string                   `json:"title"`
	Description    string                   `json:"description"`
	PriceUsd       string                   `json:"price_usd"`
	DepositUsd     *string                  `json:"deposit_usd"`
	Status         string                   `json:"status"`
	CoverPhotoUrl  string                   `json:"cover_photo_url"`
	ListedAt       *time.Time               `json:"listed_at"`
	SoldAt         *time.Time               `json:"sold_at"`
	ArchivedAt     *time.Time               `json:"archived_at"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
	GeckoCount     int32                    `json:"gecko_count"`
	ComponentCount int32                    `json:"component_count"`
	Geckos         []listingGeckoRefDTO     `json:"geckos,omitempty"`
	Components     []listingComponentRefDTO `json:"components,omitempty"`
}

type listingsListResp struct {
	Listings []listingDTO `json:"listings"`
	Total    int          `json:"total"`
}

// ---- requests ----

type listingComponentInput struct {
	ComponentListingID int32 `json:"component_listing_id"`
	Quantity           int32 `json:"quantity"`
}

type listingGeckoInput struct {
	GeckoID int32 `json:"gecko_id"`
}

type createListingReq struct {
	Sku           string                  `json:"sku"`
	Type          string                  `json:"type"`
	Title         string                  `json:"title"`
	Description   string                  `json:"description"`
	PriceUsd      string                  `json:"price_usd"`
	DepositUsd    string                  `json:"deposit_usd"`
	Status        string                  `json:"status"`
	CoverPhotoUrl string                  `json:"cover_photo_url"`
	Geckos        []listingGeckoInput     `json:"geckos"`
	Components    []listingComponentInput `json:"components"`
}

// updateListingReq is identical to createListingReq except `type` is ignored
// server-side (immutable after create).
type updateListingReq = createListingReq

var (
	validListingType = map[string]db.ListingType{
		"GECKO":   db.ListingTypeGECKO,
		"PACKAGE": db.ListingTypePACKAGE,
		"SUPPLY":  db.ListingTypeSUPPLY,
	}
	validListingStatus = map[string]db.ListingStatus{
		"DRAFT":    db.ListingStatusDRAFT,
		"LISTED":   db.ListingStatusLISTED,
		"RESERVED": db.ListingStatusRESERVED,
		"SOLD":     db.ListingStatusSOLD,
		"ARCHIVED": db.ListingStatusARCHIVED,
	}
)

// ---- handlers ----

func (d *listingsDeps) list(w http.ResponseWriter, r *http.Request) {
	// TODO: wire ?type= and ?status= query filters server-side once the
	// admin UI grows past MVP. Current caller count is small so the
	// frontend filters client-side.
	rows, err := d.q.ListListings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list failed"})
		return
	}

	out := make([]listingDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, listingDTO{
			ID:             row.ID,
			Sku:            textOrEmpty(row.Sku),
			Type:           string(row.Type),
			Title:          row.Title,
			Description:    textOrEmpty(row.Description),
			PriceUsd:       numericString(row.PriceUsd),
			DepositUsd:     numericOrNil(row.DepositUsd),
			Status:         string(row.Status),
			CoverPhotoUrl:  textOrEmpty(row.CoverPhotoUrl),
			ListedAt:       timestampPtr(row.ListedAt),
			SoldAt:         timestampPtr(row.SoldAt),
			ArchivedAt:     timestampPtr(row.ArchivedAt),
			CreatedAt:      row.CreatedAt.Time,
			UpdatedAt:      row.UpdatedAt.Time,
			GeckoCount:     row.GeckoCount,
			ComponentCount: row.ComponentCount,
		})
	}
	writeJSON(w, http.StatusOK, listingsListResp{Listings: out, Total: len(out)})
}

func (d *listingsDeps) get(w http.ResponseWriter, r *http.Request) {
	id := parseInt32Path(r, "id")
	if id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ctx := r.Context()
	row, err := d.q.GetListing(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}

	dto := listingDTO{
		ID:            row.ID,
		Sku:           textOrEmpty(row.Sku),
		Type:          string(row.Type),
		Title:         row.Title,
		Description:   textOrEmpty(row.Description),
		PriceUsd:      numericString(row.PriceUsd),
		DepositUsd:    numericOrNil(row.DepositUsd),
		Status:        string(row.Status),
		CoverPhotoUrl: textOrEmpty(row.CoverPhotoUrl),
		ListedAt:      timestampPtr(row.ListedAt),
		SoldAt:        timestampPtr(row.SoldAt),
		ArchivedAt:    timestampPtr(row.ArchivedAt),
		CreatedAt:     row.CreatedAt.Time,
		UpdatedAt:     row.UpdatedAt.Time,
	}

	// Attach junctions per type.
	switch row.Type {
	case db.ListingTypeGECKO:
		gs, err := d.q.ListGeckosForListing(ctx, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list geckos failed"})
			return
		}
		dto.Geckos = make([]listingGeckoRefDTO, 0, len(gs))
		for _, g := range gs {
			dto.Geckos = append(dto.Geckos, listingGeckoRefDTO{
				GeckoID:     g.GeckoID,
				Code:        g.Code,
				Name:        textOrEmpty(g.Name),
				SpeciesCode: string(g.SpeciesCode),
			})
		}
		dto.GeckoCount = int32(len(dto.Geckos))
	case db.ListingTypePACKAGE:
		cs, err := d.q.ListComponentsForListing(ctx, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list components failed"})
			return
		}
		dto.Components = make([]listingComponentRefDTO, 0, len(cs))
		for _, c := range cs {
			dto.Components = append(dto.Components, listingComponentRefDTO{
				ComponentListingID: c.ComponentListingID,
				Title:              c.Title,
				Type:               string(c.Type),
				PriceUsd:           numericString(c.PriceUsd),
				Quantity:           c.Quantity,
			})
		}
		dto.ComponentCount = int32(len(dto.Components))
	}

	writeJSON(w, http.StatusOK, dto)
}

func (d *listingsDeps) create(w http.ResponseWriter, r *http.Request) {
	var req createListingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	req.Sku = strings.TrimSpace(req.Sku)
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.PriceUsd = strings.TrimSpace(req.PriceUsd)
	req.DepositUsd = strings.TrimSpace(req.DepositUsd)
	req.CoverPhotoUrl = strings.TrimSpace(req.CoverPhotoUrl)

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	if len(req.Title) > 200 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title too long"})
		return
	}
	if req.PriceUsd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price_usd is required"})
		return
	}
	lt, ok := validListingType[req.Type]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid type"})
		return
	}

	// Per-type validation
	switch lt {
	case db.ListingTypeGECKO:
		if len(req.Geckos) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecko listing needs at least one gecko"})
			return
		}
	case db.ListingTypePACKAGE:
		if len(req.Components) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "package listing needs at least one component"})
			return
		}
	case db.ListingTypeSUPPLY:
		if req.Sku == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku is required for supply listings"})
			return
		}
	}

	if strings.HasPrefix(strings.TrimSpace(req.PriceUsd), "-") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price_usd must be non-negative"})
		return
	}
	price, err := parseNumeric(req.PriceUsd)
	if err != nil || !price.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid price_usd"})
		return
	}
	if strings.HasPrefix(strings.TrimSpace(req.DepositUsd), "-") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "deposit_usd must be non-negative"})
		return
	}
	deposit, err := parseNumeric(req.DepositUsd)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid deposit_usd"})
		return
	}

	var statusCol db.NullListingStatus
	if req.Status != "" {
		s, ok := validListingStatus[req.Status]
		if !ok {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
		statusCol = db.NullListingStatus{ListingStatus: s, Valid: true}
	}

	ctx := r.Context()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	listing, err := qtx.CreateListing(ctx, db.CreateListingParams{
		Sku:           pgText(req.Sku),
		Type:          lt,
		Title:         req.Title,
		Description:   pgText(req.Description),
		PriceUsd:      price,
		DepositUsd:    deposit,
		Column7:       statusCol,
		CoverPhotoUrl: pgText(req.CoverPhotoUrl),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "sku already in use"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create failed: " + err.Error()})
		return
	}

	if lt == db.ListingTypeGECKO {
		for _, g := range req.Geckos {
			if err := qtx.AttachGeckoToListing(ctx, db.AttachGeckoToListingParams{
				ListingID: listing.ID, GeckoID: g.GeckoID,
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attach gecko failed: " + err.Error()})
				return
			}
		}
	}
	if lt == db.ListingTypePACKAGE {
		for _, c := range req.Components {
			if c.Quantity <= 0 {
				c.Quantity = 1
			}
			if c.ComponentListingID == listing.ID {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "package cannot contain itself"})
				return
			}
			if err := qtx.SetListingComponent(ctx, db.SetListingComponentParams{
				ListingID:          listing.ID,
				ComponentListingID: c.ComponentListingID,
				Quantity:           c.Quantity,
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "attach component failed: " + err.Error()})
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}

	// Reuse get() for consistent response.
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(listing.ID)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	w.Header().Del("Content-Type")
	wr := &statusRecorder{ResponseWriter: w, status: http.StatusCreated}
	d.get(wr, r2)
}

// statusRecorder wraps ResponseWriter to coerce the status code to 201 on
// create even though the inner get() writes 200.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if s.wrote {
		return
	}
	s.wrote = true
	// Prefer the caller-provided 201 over the inner handler's 200.
	if code == http.StatusOK {
		code = s.status
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.WriteHeader(s.status)
	}
	return s.ResponseWriter.Write(b)
}

func (d *listingsDeps) update(w http.ResponseWriter, r *http.Request) {
	id := parseInt32Path(r, "id")
	if id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	var req updateListingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	req.Sku = strings.TrimSpace(req.Sku)
	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)
	req.PriceUsd = strings.TrimSpace(req.PriceUsd)
	req.DepositUsd = strings.TrimSpace(req.DepositUsd)
	req.CoverPhotoUrl = strings.TrimSpace(req.CoverPhotoUrl)

	ctx := r.Context()
	existing, err := d.q.GetListing(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}

	if req.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}
	if req.PriceUsd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price_usd is required"})
		return
	}

	// Per-type validation (using the existing row's type — type is immutable).
	switch existing.Type {
	case db.ListingTypeGECKO:
		if len(req.Geckos) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gecko listing needs at least one gecko"})
			return
		}
	case db.ListingTypePACKAGE:
		if len(req.Components) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "package listing needs at least one component"})
			return
		}
	case db.ListingTypeSUPPLY:
		if req.Sku == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sku is required for supply listings"})
			return
		}
	}

	if strings.HasPrefix(strings.TrimSpace(req.PriceUsd), "-") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "price_usd must be non-negative"})
		return
	}
	price, err := parseNumeric(req.PriceUsd)
	if err != nil || !price.Valid {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid price_usd"})
		return
	}
	if strings.HasPrefix(strings.TrimSpace(req.DepositUsd), "-") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "deposit_usd must be non-negative"})
		return
	}
	deposit, err := parseNumeric(req.DepositUsd)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid deposit_usd"})
		return
	}
	status, ok := validListingStatus[req.Status]
	if !ok {
		if req.Status == "" {
			status = existing.Status
		} else {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
	}

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin failed"})
		return
	}
	defer tx.Rollback(ctx)
	qtx := d.q.WithTx(tx)

	if _, err := qtx.UpdateListing(ctx, db.UpdateListingParams{
		ID:            id,
		Sku:           pgText(req.Sku),
		Title:         req.Title,
		Description:   pgText(req.Description),
		PriceUsd:      price,
		DepositUsd:    deposit,
		Status:        status,
		CoverPhotoUrl: pgText(req.CoverPhotoUrl),
	}); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "sku already in use"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed: " + err.Error()})
		return
	}

	// Replace junctions atomically.
	if existing.Type == db.ListingTypeGECKO {
		if err := qtx.DetachGeckosForListing(ctx, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "detach failed"})
			return
		}
		for _, g := range req.Geckos {
			if err := qtx.AttachGeckoToListing(ctx, db.AttachGeckoToListingParams{
				ListingID: id, GeckoID: g.GeckoID,
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reattach gecko failed"})
				return
			}
		}
	}
	if existing.Type == db.ListingTypePACKAGE {
		if err := qtx.DeleteComponentsForListing(ctx, id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "detach components failed"})
			return
		}
		for _, c := range req.Components {
			if c.Quantity <= 0 {
				c.Quantity = 1
			}
			if c.ComponentListingID == id {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "package cannot contain itself"})
				return
			}
			if err := qtx.SetListingComponent(ctx, db.SetListingComponentParams{
				ListingID:          id,
				ComponentListingID: c.ComponentListingID,
				Quantity:           c.Quantity,
			}); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "reattach component failed"})
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "commit failed"})
		return
	}

	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add("id", strconv.Itoa(int(id)))
	r2 := r.WithContext(context.WithValue(ctx, chi.RouteCtxKey, chiCtx))
	d.get(w, r2)
}

func (d *listingsDeps) delete(w http.ResponseWriter, r *http.Request) {
	id := parseInt32Path(r, "id")
	if id == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := d.q.DeleteListing(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- helpers ----

func parseInt32Path(r *http.Request, key string) int32 {
	s := chi.URLParam(r, key)
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0
	}
	return int32(n)
}

// numericString renders a pgtype.Numeric as a plain string (empty when NULL).
func numericString(n pgtype.Numeric) string {
	s := numericOrNil(n)
	if s == nil {
		return ""
	}
	return *s
}

// timestampPtr converts a nullable pgtype.Timestamp to *time.Time.
func timestampPtr(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	tt := t.Time
	return &tt
}
