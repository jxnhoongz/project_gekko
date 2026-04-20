package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
	"github.com/jxnhoongz/project_gekko/backend/internal/db"
)

type authDeps struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	signer *auth.JWTSigner
}

// NewAuthRouter mounts /api/auth/* (login public, me protected).
func NewAuthRouter(pool *pgxpool.Pool, signer *auth.JWTSigner) http.Handler {
	d := &authDeps{pool: pool, q: db.New(pool), signer: signer}

	r := chi.NewRouter()
	r.Post("/api/auth/login", d.login)

	protected := chi.NewRouter()
	protected.Use(RequireAuth(signer))
	protected.Get("/api/auth/me", d.me)

	r.Mount("/", protected)
	return r
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminDTO struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type loginResp struct {
	Token string   `json:"token"`
	Admin adminDTO `json:"admin"`
}

func (d *authDeps) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	row, err := d.q.GetAdminByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "server error"})
		return
	}

	if !auth.VerifyPassword(row.PasswordHash, req.Password) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	tok, err := d.signer.Issue(int64(row.ID), row.Email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token issue failed"})
		return
	}

	writeJSON(w, http.StatusOK, loginResp{
		Token: tok,
		Admin: adminDTO{ID: int64(row.ID), Email: row.Email, Name: textOrEmpty(row.Name)},
	})
}

func (d *authDeps) me(w http.ResponseWriter, r *http.Request) {
	claims, ok := claimsFrom(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthenticated"})
		return
	}
	row, err := d.q.GetAdminByID(r.Context(), int32(claims.AdminID))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin gone"})
		return
	}
	writeJSON(w, http.StatusOK, adminDTO{ID: int64(row.ID), Email: row.Email, Name: textOrEmpty(row.Name)})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func textOrEmpty(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
