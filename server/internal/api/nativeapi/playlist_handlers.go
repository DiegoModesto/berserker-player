package nativeapi

import (
	"encoding/json"
	"net/http"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListPlaylists(w http.ResponseWriter, r *http.Request) {
	pls, err := s.store.ListPlaylists(userID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar playlists")
		return
	}
	writeJSON(w, http.StatusOK, pls)
}

type createPlaylistReq struct {
	Name    string   `json:"name"`
	SongIDs []string `json:"songIds"`
}

func (s *Server) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	var req createPlaylistReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "nome obrigatório")
		return
	}
	pl, err := s.store.CreatePlaylist(userID(r), req.Name, req.SongIDs)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao criar playlist")
		return
	}
	writeJSON(w, http.StatusCreated, pl)
}

func (s *Server) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	pl, err := s.store.GetPlaylist(userID(r), id)
	if err != nil {
		s.notFoundOr500(w, err)
		return
	}
	songs, err := s.store.PlaylistSongs(userID(r), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	writeJSON(w, http.StatusOK, struct {
		model.Playlist
		Songs []model.Song `json:"songs"`
	}{pl, songs})
}

type updatePlaylistReq struct {
	Name    *string  `json:"name"`
	SongIDs []string `json:"songIds"`
}

func (s *Server) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	var req updatePlaylistReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	pl, err := s.store.UpdatePlaylist(userID(r), chi.URLParam(r, "id"), req.Name, req.SongIDs)
	if err != nil {
		s.notFoundOr500(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pl)
}

func (s *Server) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeletePlaylist(userID(r), chi.URLParam(r, "id")); err != nil {
		s.notFoundOr500(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
