package nativeapi

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const (
	ctxUserID  ctxKey = "userID"
	ctxIsAdmin ctxKey = "isAdmin"
)

// requireAuth valida o JWT de acesso (header Authorization: Bearer).
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "token ausente")
			return
		}
		claims, err := s.authSvc.ParseAccess(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token inválido ou expirado")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxIsAdmin, claims.IsAdmin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireMediaToken valida o token de mídia assinado (?token=).
func (s *Server) requireMediaToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := r.URL.Query().Get("token")
		if tok == "" {
			writeError(w, http.StatusUnauthorized, "token de mídia ausente")
			return
		}
		claims, err := s.authSvc.ParseMedia(tok)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "token de mídia inválido ou expirado")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxUserID).(string); ok {
		return v
	}
	return ""
}

func isAdmin(r *http.Request) bool {
	if v, ok := r.Context().Value(ctxIsAdmin).(bool); ok {
		return v
	}
	return false
}
