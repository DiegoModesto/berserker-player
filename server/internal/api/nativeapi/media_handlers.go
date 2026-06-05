package nativeapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/DiegoModesto/berserker-player/server/internal/artwork"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/stream"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	path, _, err := s.store.GetSongPath(id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, http.StatusNotFound, "faixa não encontrada")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	// Fase 0: direct play (Range/206). Transcodificação entra na fase avançada.
	if err := stream.ServeFile(w, r, path); err != nil {
		writeError(w, http.StatusInternalServerError, "falha ao servir arquivo")
	}
}

func (s *Server) handleCover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	size := 0
	if v := r.URL.Query().Get("size"); v != "" {
		size, _ = strconv.Atoi(v)
	}
	data, ct, err := s.art.Cover(id, size)
	if err != nil {
		if errors.Is(err, artwork.ErrNoCover) {
			writeError(w, http.StatusNotFound, "sem capa")
			return
		}
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}
