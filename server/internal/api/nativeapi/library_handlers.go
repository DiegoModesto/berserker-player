package nativeapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/model"
	"github.com/go-chi/chi/v5"
)

// registerLibraryRoutes registra biblioteca, playlists e anotações (Fase 1).
func (s *Server) registerLibraryRoutes(r chi.Router) {
	r.Get("/artists", s.handleListArtists)
	r.Get("/artists/{id}", s.handleGetArtist)
	r.Get("/albums", s.handleListAlbums)
	r.Get("/albums/{id}", s.handleGetAlbum)
	r.Get("/songs/{id}", s.handleGetSong)
	r.Get("/search", s.handleSearch)

	r.Get("/playlists", s.handleListPlaylists)
	r.Post("/playlists", s.handleCreatePlaylist)
	r.Get("/playlists/{id}", s.handleGetPlaylist)
	r.Put("/playlists/{id}", s.handleUpdatePlaylist)
	r.Delete("/playlists/{id}", s.handleDeletePlaylist)

	r.Post("/star", s.handleStar)
	r.Delete("/star", s.handleUnstar)
	r.Post("/rating", s.handleRating)
	r.Post("/scrobble", s.handleScrobble)
}

func pageParams(r *http.Request, defLimit int) (offset, limit int) {
	offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = defLimit
	}
	if limit > 200 {
		limit = 200
	}
	return offset, limit
}

func (s *Server) handleListArtists(w http.ResponseWriter, r *http.Request) {
	offset, limit := pageParams(r, 50)
	q := r.URL.Query()
	page, err := s.store.ListArtists(userID(r), q.Get("sort"), q.Get("order"), offset, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar artistas")
		return
	}
	if page.Items == nil {
		page.Items = []model.Artist{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	artist, err := s.store.GetArtist(userID(r), id)
	if err != nil {
		s.notFoundOr500(w, err)
		return
	}
	albums, err := s.store.AlbumsByArtist(userID(r), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	if albums == nil {
		albums = []model.Album{}
	}
	writeJSON(w, http.StatusOK, struct {
		model.Artist
		Albums []model.Album `json:"albums"`
	}{artist, albums})
}

func (s *Server) handleListAlbums(w http.ResponseWriter, r *http.Request) {
	offset, limit := pageParams(r, 50)
	q := r.URL.Query()
	page, err := s.store.ListAlbums(userID(r), core.AlbumQuery{
		Filter: q.Get("filter"), Genre: q.Get("genre"), ArtistID: q.Get("artistId"),
		Sort: q.Get("sort"), Order: q.Get("order"), Offset: offset, Limit: limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro ao listar álbuns")
		return
	}
	if page.Items == nil {
		page.Items = []model.Album{}
	}
	writeJSON(w, http.StatusOK, page)
}

func (s *Server) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	album, err := s.store.GetAlbum(userID(r), id)
	if err != nil {
		s.notFoundOr500(w, err)
		return
	}
	songs, err := s.store.SongsByAlbum(userID(r), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro interno")
		return
	}
	if songs == nil {
		songs = []model.Song{}
	}
	writeJSON(w, http.StatusOK, struct {
		model.Album
		Songs []model.Song `json:"songs"`
	}{album, songs})
}

func (s *Server) handleGetSong(w http.ResponseWriter, r *http.Request) {
	song, err := s.store.GetSong(userID(r), chi.URLParam(r, "id"))
	if err != nil {
		s.notFoundOr500(w, err)
		return
	}
	writeJSON(w, http.StatusOK, song)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, "parâmetro q obrigatório")
		return
	}
	limit := 20
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 {
		limit = v
		if limit > 50 {
			limit = 50
		}
	}
	res, err := s.store.Search(userID(r), q, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "erro na busca")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) notFoundOr500(w http.ResponseWriter, err error) {
	if errors.Is(err, core.ErrNotFound) {
		writeError(w, http.StatusNotFound, "não encontrado")
		return
	}
	writeError(w, http.StatusInternalServerError, "erro interno")
}
