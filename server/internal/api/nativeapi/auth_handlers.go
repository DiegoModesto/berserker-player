package nativeapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/auth"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
)

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken,omitempty"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// isWebClient decide se o refresh vai por cookie httpOnly (WebApp same-origin)
// ou no corpo (clientes nativos). Heurística: presença do header X-Client.
func isWebClient(r *http.Request) bool {
	return r.Header.Get("X-Client") == "webapp"
}

func (s *Server) issueTokens(w http.ResponseWriter, r *http.Request, u core.UserAuth) (tokenPair, error) {
	now := time.Now()
	access, exp, err := s.authSvc.AccessToken(u.ID, u.IsAdmin, now)
	if err != nil {
		return tokenPair{}, err
	}
	refreshRaw, refreshHash, err := auth.NewRefreshToken()
	if err != nil {
		return tokenPair{}, err
	}
	if _, err := s.store.SaveRefreshToken(u.ID, refreshHash, now.Add(s.authSvc.RefreshTTL())); err != nil {
		return tokenPair{}, err
	}
	tp := tokenPair{AccessToken: access, ExpiresAt: exp}
	if isWebClient(r) {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    refreshRaw,
			Path:     "/api/v1/auth",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteStrictMode,
			Expires:  now.Add(s.authSvc.RefreshTTL()),
		})
	} else {
		tp.RefreshToken = refreshRaw
	}
	return tp, nil
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	u, err := s.store.UserByUsername(req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}
	ok, err := auth.VerifyPassword(req.Password, u.PasswordHash)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "credenciais inválidas")
		return
	}
	tp, err := s.issueTokens(w, r, core.UserAuth{ID: u.ID, IsAdmin: u.IsAdmin})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao emitir tokens")
		return
	}
	writeJSON(w, http.StatusOK, tp)
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	raw := refreshFromRequest(r)
	if raw == "" {
		writeError(w, http.StatusUnauthorized, "refresh token ausente")
		return
	}
	rec, err := s.store.FindRefreshToken(auth.HashToken(raw))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh token inválido")
		return
	}
	if rec.Revoked {
		// Reuso de token revogado: revoga tudo do usuário (defesa contra roubo).
		_ = s.store.RevokeAllUserTokens(rec.UserID)
		writeError(w, http.StatusUnauthorized, "refresh token reutilizado")
		return
	}
	if time.Now().After(rec.ExpiresAt) {
		writeError(w, http.StatusUnauthorized, "refresh token expirado")
		return
	}
	// Rotação: revoga o atual e emite novo par.
	_ = s.store.RevokeRefreshToken(rec.ID)
	u, err := s.store.UserByID(rec.UserID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "usuário inexistente")
		return
	}
	tp, err := s.issueTokens(w, r, core.UserAuth{ID: u.ID, IsAdmin: u.IsAdmin})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao emitir tokens")
		return
	}
	writeJSON(w, http.StatusOK, tp)
}

func refreshFromRequest(r *http.Request) string {
	if c, err := r.Cookie("refreshToken"); err == nil && c.Value != "" {
		return c.Value
	}
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	return body.RefreshToken
}

func (s *Server) handleMediaToken(w http.ResponseWriter, r *http.Request) {
	tok, exp, err := s.authSvc.MediaToken(userID(r), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao emitir token de mídia")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "expiresAt": exp})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.store.UserByID(userID(r))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "usuário não encontrado")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
