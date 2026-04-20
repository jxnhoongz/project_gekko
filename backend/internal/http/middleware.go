package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/jxnhoongz/project_gekko/backend/internal/auth"
)

type ctxKey string

const ctxAdminKey ctxKey = "admin"

func RequireAuth(signer *auth.JWTSigner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			tok := strings.TrimPrefix(h, "Bearer ")
			claims, err := signer.Verify(tok)
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxAdminKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func claimsFrom(r *http.Request) (*auth.Claims, bool) {
	c, ok := r.Context().Value(ctxAdminKey).(*auth.Claims)
	return c, ok
}
