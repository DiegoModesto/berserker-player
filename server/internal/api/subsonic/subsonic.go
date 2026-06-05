// Package subsonic implementa um subconjunto da API Subsonic/OpenSubsonic
// (/rest/*) reusando o mesmo core. Apenas serializa para o formato Subsonic.
//
// Auth: aceita `u` + `p` (senha em texto puro ou "enc:"+hex), verificada contra
// o hash argon2id. O método token+salt (t,s) NÃO é suportado, pois exigiria senha
// recuperável (ver decisão em Plans/01-server-plan.md §8).
package subsonic

import (
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/DiegoModesto/berserker-player/server/internal/artwork"
	"github.com/DiegoModesto/berserker-player/server/internal/auth"
	"github.com/DiegoModesto/berserker-player/server/internal/config"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/model"
	"github.com/DiegoModesto/berserker-player/server/internal/stream"
	"github.com/go-chi/chi/v5"
)

const apiVersion = "1.16.1"

type Server struct {
	cfg        config.Config
	store      *core.Store
	art        *artwork.Resolver
	transcoder *stream.Transcoder
	log        *slog.Logger
}

func NewServer(cfg config.Config, store *core.Store, art *artwork.Resolver, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: store, art: art, transcoder: stream.NewTranscoder(cfg.FFmpegPath), log: log}
}

func (s *Server) Mount(r chi.Router) {
	r.Route("/rest", func(r chi.Router) {
		r.Use(s.authMiddleware)
		r.HandleFunc("/ping", s.ping)
		r.HandleFunc("/ping.view", s.ping)
		r.HandleFunc("/getLicense", s.getLicense)
		r.HandleFunc("/getLicense.view", s.getLicense)
		r.HandleFunc("/getArtists", s.getArtists)
		r.HandleFunc("/getArtists.view", s.getArtists)
		r.HandleFunc("/getArtist", s.getArtist)
		r.HandleFunc("/getArtist.view", s.getArtist)
		r.HandleFunc("/getAlbum", s.getAlbum)
		r.HandleFunc("/getAlbum.view", s.getAlbum)
		r.HandleFunc("/getAlbumList2", s.getAlbumList2)
		r.HandleFunc("/getAlbumList2.view", s.getAlbumList2)
		r.HandleFunc("/search3", s.search3)
		r.HandleFunc("/search3.view", s.search3)
		r.HandleFunc("/stream", s.stream)
		r.HandleFunc("/stream.view", s.stream)
		r.HandleFunc("/getCoverArt", s.getCoverArt)
		r.HandleFunc("/getCoverArt.view", s.getCoverArt)
		r.HandleFunc("/scrobble", s.scrobble)
		r.HandleFunc("/scrobble.view", s.scrobble)
		r.HandleFunc("/star", s.starHandler(true))
		r.HandleFunc("/star.view", s.starHandler(true))
		r.HandleFunc("/unstar", s.starHandler(false))
		r.HandleFunc("/unstar.view", s.starHandler(false))
	})
}

// ---- Auth ----

type ctxKey string

const ctxUserID ctxKey = "subsonicUserID"

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		username := q.Get("u")
		if q.Get("t") != "" && q.Get("s") != "" {
			s.writeError(w, r, 41, "Autenticação por token (t/s) não suportada; use a senha (p).")
			return
		}
		password := decodePassword(q.Get("p"))
		u, err := s.store.UserByUsername(username)
		if err != nil {
			s.writeError(w, r, 40, "Usuário ou senha inválidos.")
			return
		}
		ok, _ := auth.VerifyPassword(password, u.PasswordHash)
		if !ok {
			s.writeError(w, r, 40, "Usuário ou senha inválidos.")
			return
		}
		ctx := r.Context()
		ctx = withUserID(ctx, u.ID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func decodePassword(p string) string {
	if strings.HasPrefix(p, "enc:") {
		if b, err := hex.DecodeString(strings.TrimPrefix(p, "enc:")); err == nil {
			return string(b)
		}
	}
	return p
}

// ---- Response envelope ----

type response struct {
	Inner map[string]any `json:"subsonic-response"`
}

func (s *Server) base() map[string]any {
	return map[string]any{
		"status": "ok", "version": apiVersion, "type": "berserker", "serverVersion": "0.1.0",
	}
}

func (s *Server) write(w http.ResponseWriter, r *http.Request, payload map[string]any) {
	body := s.base()
	for k, v := range payload {
		body[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{Inner: body})
	_ = r
}

func (s *Server) writeError(w http.ResponseWriter, r *http.Request, code int, msg string) {
	body := s.base()
	body["status"] = "failed"
	body["error"] = map[string]any{"code": code, "message": msg}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{Inner: body})
	_ = r
}

// ---- Handlers ----

func (s *Server) ping(w http.ResponseWriter, r *http.Request) { s.write(w, r, nil) }

func (s *Server) getLicense(w http.ResponseWriter, r *http.Request) {
	s.write(w, r, map[string]any{"license": map[string]any{"valid": true}})
}

func (s *Server) getArtists(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	page, err := s.store.ListArtists(uid, "name", "asc", 0, 5000)
	if err != nil {
		s.writeError(w, r, 0, "erro interno")
		return
	}
	artists := make([]map[string]any, 0, len(page.Items))
	for _, a := range page.Items {
		artists = append(artists, map[string]any{
			"id": a.ID, "name": a.Name, "albumCount": a.AlbumCount,
		})
	}
	index := []map[string]any{{"name": "*", "artist": artists}}
	s.write(w, r, map[string]any{"artists": map[string]any{"index": index}})
}

func (s *Server) getArtist(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	id := r.URL.Query().Get("id")
	artist, err := s.store.GetArtist(uid, id)
	if err != nil {
		s.writeError(w, r, 70, "Artista não encontrado.")
		return
	}
	albums, _ := s.store.AlbumsByArtist(uid, id)
	out := make([]map[string]any, 0, len(albums))
	for _, al := range albums {
		out = append(out, s.albumJSON(al))
	}
	s.write(w, r, map[string]any{"artist": map[string]any{
		"id": artist.ID, "name": artist.Name, "albumCount": artist.AlbumCount, "album": out,
	}})
}

func (s *Server) getAlbum(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	id := r.URL.Query().Get("id")
	album, err := s.store.GetAlbum(uid, id)
	if err != nil {
		s.writeError(w, r, 70, "Álbum não encontrado.")
		return
	}
	songs, _ := s.store.SongsByAlbum(uid, id)
	aj := s.albumJSON(album)
	children := make([]map[string]any, 0, len(songs))
	for _, so := range songs {
		children = append(children, s.songJSON(so))
	}
	aj["song"] = children
	s.write(w, r, map[string]any{"album": aj})
}

func (s *Server) getAlbumList2(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	q := r.URL.Query()
	size, _ := strconv.Atoi(q.Get("size"))
	if size <= 0 {
		size = 10
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	filter := "all"
	switch q.Get("type") {
	case "newest":
		filter = "recent"
	case "frequent":
		filter = "frequent"
	case "random":
		filter = "random"
	case "starred":
		filter = "starred"
	}
	page, _ := s.store.ListAlbums(uid, core.AlbumQuery{Filter: filter, Offset: offset, Limit: size})
	out := make([]map[string]any, 0, len(page.Items))
	for _, al := range page.Items {
		out = append(out, s.albumJSON(al))
	}
	s.write(w, r, map[string]any{"albumList2": map[string]any{"album": out}})
}

func (s *Server) search3(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	res, _ := s.store.Search(uid, r.URL.Query().Get("query"), 20)
	artists := make([]map[string]any, 0)
	for _, a := range res.Artists {
		artists = append(artists, map[string]any{"id": a.ID, "name": a.Name, "albumCount": a.AlbumCount})
	}
	albums := make([]map[string]any, 0)
	for _, a := range res.Albums {
		albums = append(albums, s.albumJSON(a))
	}
	songs := make([]map[string]any, 0)
	for _, so := range res.Songs {
		songs = append(songs, s.songJSON(so))
	}
	s.write(w, r, map[string]any{"searchResult3": map[string]any{
		"artist": artists, "album": albums, "song": songs,
	}})
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	path, _, err := s.store.GetSongPath(id)
	if err != nil {
		s.writeError(w, r, 70, "Faixa não encontrada.")
		return
	}
	format := r.URL.Query().Get("format")
	if s.cfg.TranscodingEnabled && format != "" && format != "raw" && stream.SupportedFormat(format) {
		maxBitRate, _ := strconv.Atoi(r.URL.Query().Get("maxBitRate"))
		_ = s.transcoder.Stream(r.Context(), w, path, format, maxBitRate, 0)
		return
	}
	_ = stream.ServeFile(w, r, path)
}

func (s *Server) getCoverArt(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	size := 0
	if v := r.URL.Query().Get("size"); v != "" {
		size, _ = strconv.Atoi(v)
	}
	data, ct, err := s.art.Cover(id, size)
	if err != nil {
		s.writeError(w, r, 70, "Sem capa.")
		return
	}
	w.Header().Set("Content-Type", ct)
	_, _ = w.Write(data)
}

func (s *Server) scrobble(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	id := r.URL.Query().Get("id")
	submission := r.URL.Query().Get("submission") != "false"
	if submission {
		_ = s.store.Scrobble(uid, id, nowUTC())
	}
	s.write(w, r, nil)
}

func (s *Server) starHandler(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid := userID(r)
		id := r.URL.Query().Get("id")
		if id == "" {
			s.write(w, r, nil)
			return
		}
		// Heurística de tipo: tenta álbum, depois artista, senão faixa.
		t := model.ItemSong
		if _, err := s.store.GetAlbum(uid, id); err == nil {
			t = model.ItemAlbum
		} else if _, err := s.store.GetArtist(uid, id); err == nil {
			t = model.ItemArtist
		}
		if on {
			_ = s.store.Star(uid, id, t)
		} else {
			_ = s.store.Unstar(uid, id, t)
		}
		s.write(w, r, nil)
	}
}

// ---- Mapeamento de entidades ----

func (s *Server) albumJSON(a model.Album) map[string]any {
	return map[string]any{
		"id": a.ID, "name": a.Name, "artist": a.ArtistName, "artistId": a.ArtistID,
		"coverArt": a.ID, "songCount": a.SongCount, "duration": a.Duration,
		"year": a.Year, "starred": starredStr(a.Starred),
	}
}

func (s *Server) songJSON(so model.Song) map[string]any {
	return map[string]any{
		"id": so.ID, "parent": so.AlbumID, "title": so.Title, "album": so.AlbumName,
		"artist": so.ArtistName, "artistId": so.ArtistID, "albumId": so.AlbumID,
		"track": so.Track, "duration": so.Duration, "suffix": so.Suffix,
		"coverArt": so.AlbumID, "isDir": false, "type": "music",
	}
}

func starredStr(b bool) any {
	if b {
		return true
	}
	return nil
}
