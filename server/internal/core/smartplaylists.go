package core

import (
	"encoding/json"
	"strings"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

// SmartRules descreve as regras de uma smart playlist (subconjunto pragmático).
type SmartRules struct {
	Genre   string `json:"genre,omitempty"`   // album.genre exato
	Artist  string `json:"artist,omitempty"`  // substring do nome do artista
	MinYear int    `json:"minYear,omitempty"` // ano >= (0 ignora)
	MaxYear int    `json:"maxYear,omitempty"` // ano <= (0 ignora)
	Starred bool   `json:"starred,omitempty"` // apenas faixas favoritadas pelo usuário
	Sort    string `json:"sort,omitempty"`    // recentlyAdded|playCount|random|title
	Limit   int    `json:"limit,omitempty"`   // máx. de faixas (default 50, teto 500)
}

// CreateSmartPlaylist cria uma playlist dinâmica a partir de regras.
func (s *Store) CreateSmartPlaylist(userID, name string, rules SmartRules) (model.Playlist, error) {
	raw, err := json.Marshal(rules)
	if err != nil {
		return model.Playlist{}, err
	}
	id := NewID()
	now := nowUTC().Format(rfc3339)
	if _, err := s.db.Exec(
		`INSERT INTO playlists(id, name, owner_id, created_at, updated_at, is_smart, rules) VALUES(?,?,?,?,?,1,?)`,
		id, name, userID, now, now, string(raw)); err != nil {
		return model.Playlist{}, err
	}
	return s.GetPlaylist(userID, id)
}

// IsSmart informa se a playlist é dinâmica e devolve suas regras.
func (s *Store) IsSmart(playlistID string) (bool, SmartRules) {
	var isSmart int
	var raw string
	if err := s.db.QueryRow(`SELECT is_smart, rules FROM playlists WHERE id = ?`, playlistID).
		Scan(&isSmart, &raw); err != nil {
		return false, SmartRules{}
	}
	if isSmart != 1 {
		return false, SmartRules{}
	}
	var r SmartRules
	_ = json.Unmarshal([]byte(raw), &r)
	return true, r
}

// EvaluateSmart resolve as faixas de uma smart playlist conforme suas regras.
func (s *Store) EvaluateSmart(userID string, r SmartRules) ([]model.Song, error) {
	where := []string{}
	args := []any{userID}
	if r.Genre != "" {
		where = append(where, "al.genre = ?")
		args = append(args, r.Genre)
	}
	if r.Artist != "" {
		where = append(where, "ar.name LIKE ?")
		args = append(args, "%"+r.Artist+"%")
	}
	if r.MinYear > 0 {
		where = append(where, "al.year >= ?")
		args = append(args, r.MinYear)
	}
	if r.MaxYear > 0 {
		where = append(where, "al.year <= ?")
		args = append(args, r.MaxYear)
	}
	if r.Starred {
		where = append(where, "an.starred_at IS NOT NULL")
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	order := " ORDER BY ar.name, al.name, m.disc, m.track"
	switch r.Sort {
	case "recentlyAdded":
		order = " ORDER BY m.created_at DESC"
	case "playCount":
		order = " ORDER BY COALESCE(an.play_count,0) DESC"
	case "random":
		order = " ORDER BY RANDOM()"
	case "title":
		order = " ORDER BY m.title"
	}

	limit := r.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	args = append(args, limit)

	q := songSelect + whereSQL + order + " LIMIT ?"
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Song{}
	for rows.Next() {
		so, err := s.scanSong(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, so)
	}
	return out, rows.Err()
}
