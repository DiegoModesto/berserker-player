package core

import (
	"database/sql"
	"errors"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

func (s *Store) scanPlaylist(row interface{ Scan(...any) error }) (model.Playlist, error) {
	var p model.Playlist
	var created, updated string
	if err := row.Scan(&p.ID, &p.Name, &p.OwnerID, &p.SongCount, &p.Duration, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return p, ErrNotFound
		}
		return p, err
	}
	p.CreatedAt, p.UpdatedAt = parseTime(created), parseTime(updated)
	return p, nil
}

const playlistSelect = `
SELECT p.id, p.name, p.owner_id,
       (SELECT COUNT(*) FROM playlist_tracks pt WHERE pt.playlist_id = p.id),
       (SELECT COALESCE(SUM(m.duration),0) FROM playlist_tracks pt JOIN media_files m ON m.id = pt.media_id WHERE pt.playlist_id = p.id),
       p.created_at, p.updated_at
FROM playlists p`

func (s *Store) ListPlaylists(userID string) ([]model.Playlist, error) {
	rows, err := s.db.Query(playlistSelect+` WHERE p.owner_id = ? ORDER BY p.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Playlist{}
	for rows.Next() {
		p, err := s.scanPlaylist(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetPlaylist(userID, id string) (model.Playlist, error) {
	row := s.db.QueryRow(playlistSelect+` WHERE p.id = ? AND p.owner_id = ?`, id, userID)
	return s.scanPlaylist(row)
}

func (s *Store) CreatePlaylist(userID, name string, songIDs []string) (model.Playlist, error) {
	id := NewID()
	now := nowUTC().Format(rfc3339)
	if _, err := s.db.Exec(`INSERT INTO playlists(id, name, owner_id, created_at, updated_at) VALUES(?,?,?,?,?)`,
		id, name, userID, now, now); err != nil {
		return model.Playlist{}, err
	}
	if err := s.replaceTracks(id, songIDs); err != nil {
		return model.Playlist{}, err
	}
	return s.GetPlaylist(userID, id)
}

// UpdatePlaylist renomeia e/ou substitui a ordem completa de faixas.
func (s *Store) UpdatePlaylist(userID, id string, name *string, songIDs []string) (model.Playlist, error) {
	if _, err := s.GetPlaylist(userID, id); err != nil {
		return model.Playlist{}, err
	}
	if name != nil {
		if _, err := s.db.Exec(`UPDATE playlists SET name = ? WHERE id = ?`, *name, id); err != nil {
			return model.Playlist{}, err
		}
	}
	if songIDs != nil {
		if err := s.replaceTracks(id, songIDs); err != nil {
			return model.Playlist{}, err
		}
	}
	if _, err := s.db.Exec(`UPDATE playlists SET updated_at = ? WHERE id = ?`, nowUTC().Format(rfc3339), id); err != nil {
		return model.Playlist{}, err
	}
	return s.GetPlaylist(userID, id)
}

func (s *Store) DeletePlaylist(userID, id string) error {
	res, err := s.db.Exec(`DELETE FROM playlists WHERE id = ? AND owner_id = ?`, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) replaceTracks(playlistID string, songIDs []string) error {
	if _, err := s.db.Exec(`DELETE FROM playlist_tracks WHERE playlist_id = ?`, playlistID); err != nil {
		return err
	}
	for i, sid := range songIDs {
		if _, err := s.db.Exec(`INSERT INTO playlist_tracks(playlist_id, media_id, position) VALUES(?,?,?)`,
			playlistID, sid, i); err != nil {
			return err
		}
	}
	return nil
}

// PlaylistSongs devolve as faixas da playlist na ordem definida.
func (s *Store) PlaylistSongs(userID, playlistID string) ([]model.Song, error) {
	q := songSelect + `
        JOIN playlist_tracks pt ON pt.media_id = m.id
        WHERE pt.playlist_id = ? ORDER BY pt.position`
	rows, err := s.db.Query(q, userID, playlistID)
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
