package nativeapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

type itemRef struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func (i itemRef) valid() bool {
	switch model.ItemType(i.Type) {
	case model.ItemArtist, model.ItemAlbum, model.ItemSong:
		return i.ID != ""
	}
	return false
}

func (s *Server) handleStar(w http.ResponseWriter, r *http.Request) {
	var ref itemRef
	if err := json.NewDecoder(r.Body).Decode(&ref); err != nil || !ref.valid() {
		writeError(w, http.StatusBadRequest, "id/type inválidos")
		return
	}
	if err := s.store.Star(userID(r), ref.ID, model.ItemType(ref.Type)); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao favoritar")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUnstar(w http.ResponseWriter, r *http.Request) {
	var ref itemRef
	if err := json.NewDecoder(r.Body).Decode(&ref); err != nil || !ref.valid() {
		writeError(w, http.StatusBadRequest, "id/type inválidos")
		return
	}
	if err := s.store.Unstar(userID(r), ref.ID, model.ItemType(ref.Type)); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao desfavoritar")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type ratingReq struct {
	itemRef
	Rating int `json:"rating"`
}

func (s *Server) handleRating(w http.ResponseWriter, r *http.Request) {
	var req ratingReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !req.valid() || req.Rating < 0 || req.Rating > 5 {
		writeError(w, http.StatusBadRequest, "parâmetros inválidos")
		return
	}
	if err := s.store.SetRating(userID(r), req.ID, model.ItemType(req.Type), req.Rating); err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao definir rating")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type scrobbleReq struct {
	SongID   string    `json:"songId"`
	Event    string    `json:"event"`
	PlayedAt time.Time `json:"playedAt"`
}

func (s *Server) handleScrobble(w http.ResponseWriter, r *http.Request) {
	var req scrobbleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SongID == "" {
		writeError(w, http.StatusBadRequest, "songId obrigatório")
		return
	}
	// 'nowplaying' não contabiliza; só 'submission' incrementa play_count.
	if req.Event == "submission" {
		at := req.PlayedAt
		if at.IsZero() {
			at = time.Now().UTC()
		}
		if err := s.store.Scrobble(userID(r), req.SongID, at); err != nil {
			writeError(w, http.StatusInternalServerError, "erro ao registrar scrobble")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
