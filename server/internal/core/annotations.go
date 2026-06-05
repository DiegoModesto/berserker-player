package core

import (
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

// ensureAnnotation garante a existência da linha (user, type, item).
func (s *Store) ensureAnnotation(userID, itemID string, t model.ItemType) error {
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO annotations(user_id, item_id, item_type) VALUES(?,?,?)`,
		userID, itemID, string(t))
	return err
}

// Star marca um item como favorito.
func (s *Store) Star(userID, itemID string, t model.ItemType) error {
	if err := s.ensureAnnotation(userID, itemID, t); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE annotations SET starred_at = ? WHERE user_id=? AND item_type=? AND item_id=?`,
		nowUTC().Format(rfc3339), userID, string(t), itemID)
	return err
}

// Unstar remove o favorito.
func (s *Store) Unstar(userID, itemID string, t model.ItemType) error {
	_, err := s.db.Exec(
		`UPDATE annotations SET starred_at = NULL WHERE user_id=? AND item_type=? AND item_id=?`,
		userID, string(t), itemID)
	return err
}

// SetRating define a nota (0-5) de um item.
func (s *Store) SetRating(userID, itemID string, t model.ItemType, rating int) error {
	if err := s.ensureAnnotation(userID, itemID, t); err != nil {
		return err
	}
	_, err := s.db.Exec(
		`UPDATE annotations SET rating = ? WHERE user_id=? AND item_type=? AND item_id=?`,
		rating, userID, string(t), itemID)
	return err
}

// Scrobble registra uma reprodução (submission), deduplicando repetições
// próximas (janela de 20s) para evitar contagem dupla entre clientes/recargas.
func (s *Store) Scrobble(userID, songID string, playedAt time.Time) error {
	if err := s.ensureAnnotation(userID, songID, model.ItemSong); err != nil {
		return err
	}
	var last string
	_ = s.db.QueryRow(
		`SELECT COALESCE(last_played,'') FROM annotations WHERE user_id=? AND item_type='song' AND item_id=?`,
		userID, songID).Scan(&last)
	if last != "" {
		if t := parseTime(last); playedAt.Sub(t) < 20*time.Second && playedAt.After(t) {
			return nil // dedup
		}
	}
	if _, err := s.db.Exec(
		`UPDATE annotations SET play_count = play_count + 1, last_played = ? WHERE user_id=? AND item_type='song' AND item_id=?`,
		playedAt.Format(rfc3339), userID, songID); err != nil {
		return err
	}
	// Incrementa play_count do álbum (agregado objetivo simples).
	_, err := s.db.Exec(
		`UPDATE albums SET play_count = play_count + 1 WHERE id = (SELECT album_id FROM media_files WHERE id = ?)`,
		songID)
	return err
}
