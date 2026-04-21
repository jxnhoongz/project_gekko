package http

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/config"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

const (
	maxUploadBytes = 10 * 1024 * 1024 // 10 MB per file
	maxFormBytes   = 12 * 1024 * 1024
)

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type mediaDeps struct {
	pool *pgxpool.Pool
	q    *db.Queries
	cfg  *config.Config
}

// MountMedia registers /api/geckos/{id}/media upload and /api/media/{id} delete.
func MountMedia(r chi.Router, pool *pgxpool.Pool, signer *auth.JWTSigner, cfg *config.Config) {
	d := &mediaDeps{pool: pool, q: db.New(pool), cfg: cfg}
	r.Group(func(pr chi.Router) {
		pr.Use(RequireAuth(signer))
		pr.Post("/api/geckos/{id}/media", d.upload)
		pr.Delete("/api/media/{id}", d.delete)
	})
}

func (d *mediaDeps) upload(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	geckoID, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid gecko id"})
		return
	}

	ctx := r.Context()
	// Ensure gecko exists (avoids dangling files on bad ids).
	if _, err := d.q.GetGeckoByID(ctx, int32(geckoID)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "gecko not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	if err := r.ParseMultipartForm(maxFormBytes); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "multipart parse failed: " + err.Error()})
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
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{
			"error": "unsupported type; allowed: jpg, png, webp, gif",
		})
		return
	}

	name, err := randomFilename(ext)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "rng failed"})
		return
	}

	relDir := filepath.Join("geckos", strconv.Itoa(geckoID))
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

	// Public URL (served by the static file handler).
	publicURL := "/uploads/" + filepath.ToSlash(filepath.Join(relDir, name))

	caption := strings.TrimSpace(r.FormValue("caption"))

	// Display order = current count, so new uploads land at the end.
	order, _ := d.q.CountMediaForGecko(ctx, pgtype.Int4{Int32: int32(geckoID), Valid: true})

	m, err := d.q.CreateMedia(ctx, db.CreateMediaParams{
		GeckoID:      pgtype.Int4{Int32: int32(geckoID), Valid: true},
		Url:          publicURL,
		Column3:      db.NullMediaType{MediaType: db.MediaTypeGALLERY, Valid: true},
		Caption:      pgText(caption),
		Column5:      int32(order),
	})
	if err != nil {
		_ = os.Remove(diskPath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db insert failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, mediaDTO{
		ID:           m.ID,
		Url:          m.Url,
		Type:         string(m.Type),
		Caption:      textOrEmpty(m.Caption),
		DisplayOrder: m.DisplayOrder,
	})
}

func (d *mediaDeps) delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id64, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	ctx := r.Context()
	row, err := d.q.GetMediaByID(ctx, int32(id64))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "lookup failed"})
		return
	}

	if err := d.q.DeleteMedia(ctx, row.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete row failed"})
		return
	}

	// Best-effort file cleanup. Only unlink if the URL points inside the
	// upload dir and resolves to an actual file under it — defensive against
	// an older row ever pointing at an external URL.
	if rel, ok := strings.CutPrefix(row.Url, "/uploads/"); ok {
		abs, err := filepath.Abs(filepath.Join(d.cfg.UploadDir, filepath.FromSlash(rel)))
		if err == nil {
			root, err2 := filepath.Abs(d.cfg.UploadDir)
			if err2 == nil && strings.HasPrefix(abs, root+string(filepath.Separator)) {
				_ = os.Remove(abs)
			}
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func randomFilename(ext string) (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d-%s%s", nowUnix(), hex.EncodeToString(b), ext), nil
}

func nowUnix() int64 {
	return timeNow().Unix()
}

// detectContentType reads the first 512 bytes and rewinds the reader, per
// http.DetectContentType's contract. Safe for multipart.File (seekable).
func detectContentType(r multipart.File) string {
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	_, _ = r.Seek(0, io.SeekStart)
	return http.DetectContentType(buf[:n])
}
