package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/config"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

func MountTraits(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner, cfg *config.Config) {
	d := &traitsDeps{pool: pool, q: db.New(pool), cfg: cfg}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Get("/api/traits/{id}", d.get)
		pr.Patch("/api/traits/{id}", d.update)
		pr.Post("/api/traits/{id}/photo", d.uploadPhoto)
	})
}

type traitsDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
	cfg  *config.Config
}

type updateTraitReq struct {
	TraitCode       string `json:"trait_code"`
	Description     string `json:"description"`
	Notes           string `json:"notes"`
	InheritanceType string `json:"inheritance_type"`
	SuperFormName   string `json:"super_form_name"`
}

func (d *traitsDeps) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	t, err := d.q.GetTraitByID(r.Context(), int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}
	writeJSON(w, http.StatusOK, getTraitToDTO(t))
}

func (d *traitsDeps) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req updateTraitReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	// Validate inheritance_type
	if _, ok := validInheritanceType[req.InheritanceType]; !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid inheritance_type"})
		return
	}

	ctx := r.Context()
	// Fetch current to preserve example_photo_url
	cur, err := d.q.GetTraitByID(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}

	t, err := d.q.UpdateTrait(ctx, db.UpdateTraitParams{
		ID:              int32(id),
		TraitCode:       pgText(req.TraitCode),
		Description:     pgText(req.Description),
		Notes:           pgText(req.Notes),
		InheritanceType: db.InheritanceType(req.InheritanceType),
		SuperFormName:   pgText(req.SuperFormName),
		ExamplePhotoUrl: cur.ExamplePhotoUrl, // preserve existing photo
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "update failed"})
		return
	}
	writeJSON(w, http.StatusOK, updateTraitToDTO(t))
}

func (d *traitsDeps) uploadPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	ctx := r.Context()
	// Ensure trait exists.
	if _, err := d.q.GetTraitByID(ctx, int32(id)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseMultipartForm(maxFormBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "multipart parse failed"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing file field"})
		return
	}
	defer file.Close()

	if header.Size > maxUploadBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "file too large (max 10 MB)"})
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(file)
	}
	ext, ok := allowedContentTypes[strings.ToLower(contentType)]
	if !ok {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "unsupported type; allowed: jpg, png, webp, gif"})
		return
	}

	name, err := randomFilename(ext)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "rng failed"})
		return
	}

	relDir := filepath.Join("traits", strconv.Itoa(id))
	absDir := filepath.Join(d.cfg.UploadDir, relDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mkdir failed"})
		return
	}

	diskPath := filepath.Join(absDir, name)
	out, err := os.OpenFile(diskPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create file failed"})
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		_ = os.Remove(diskPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "write failed"})
		return
	}

	publicURL := "/uploads/" + filepath.ToSlash(filepath.Join(relDir, name))

	// Fetch current to preserve other fields.
	cur, err := d.q.GetTraitByID(ctx, int32(id))
	if err != nil {
		_ = os.Remove(diskPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "fetch failed"})
		return
	}
	_, err = d.q.UpdateTrait(ctx, db.UpdateTraitParams{
		ID:              int32(id),
		TraitCode:       cur.TraitCode,
		Description:     cur.Description,
		Notes:           cur.Notes,
		InheritanceType: cur.InheritanceType,
		SuperFormName:   cur.SuperFormName,
		ExamplePhotoUrl: pgText(publicURL),
	})
	if err != nil {
		_ = os.Remove(diskPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db update failed"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": publicURL})
}

// validInheritanceType is the allowed set for PATCH validation.
var validInheritanceType = map[string]struct{}{
	"RECESSIVE":   {},
	"CO_DOMINANT": {},
	"DOMINANT":    {},
	"POLYGENIC":   {},
}

// getTraitToDTO converts a GetTraitByIDRow to traitDTO.
func getTraitToDTO(t db.GetTraitByIDRow) traitDTO {
	return traitDTO{
		ID:              t.ID,
		SpeciesID:       t.SpeciesID,
		TraitName:       t.TraitName,
		TraitCode:       textOrEmpty(t.TraitCode),
		Description:     textOrEmpty(t.Description),
		IsDominant:      t.IsDominant,
		InheritanceType: string(t.InheritanceType),
		SuperFormName:   textOrEmpty(t.SuperFormName),
		ExamplePhotoUrl: textOrEmpty(t.ExamplePhotoUrl),
		Notes:           textOrEmpty(t.Notes),
	}
}

// updateTraitToDTO converts an UpdateTraitRow to traitDTO.
func updateTraitToDTO(t db.UpdateTraitRow) traitDTO {
	return traitDTO{
		ID:              t.ID,
		SpeciesID:       t.SpeciesID,
		TraitName:       t.TraitName,
		TraitCode:       textOrEmpty(t.TraitCode),
		Description:     textOrEmpty(t.Description),
		IsDominant:      t.IsDominant,
		InheritanceType: string(t.InheritanceType),
		SuperFormName:   textOrEmpty(t.SuperFormName),
		ExamplePhotoUrl: textOrEmpty(t.ExamplePhotoUrl),
		Notes:           textOrEmpty(t.Notes),
	}
}
