package core

import (
	"database/sql"
	"errors"
	"time"
)

// SaveRefreshToken persiste o hash de um refresh token emitido.
func (s *Store) SaveRefreshToken(userID, tokenHash string, expiresAt time.Time) (string, error) {
	id := NewID()
	_, err := s.db.Exec(
		`INSERT INTO refresh_tokens(id, user_id, token_hash, expires_at, revoked, created_at) VALUES(?,?,?,?,0,?)`,
		id, userID, tokenHash, expiresAt.Format(rfc3339), nowUTC().Format(rfc3339))
	return id, err
}

type RefreshRecord struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
}

// FindRefreshToken localiza um refresh token pelo hash.
func (s *Store) FindRefreshToken(tokenHash string) (RefreshRecord, error) {
	var r RefreshRecord
	var exp string
	var revoked int
	err := s.db.QueryRow(
		`SELECT id, user_id, expires_at, revoked FROM refresh_tokens WHERE token_hash = ?`, tokenHash).
		Scan(&r.ID, &r.UserID, &exp, &revoked)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r, ErrNotFound
		}
		return r, err
	}
	r.ExpiresAt = parseTime(exp)
	r.Revoked = revoked == 1
	return r, nil
}

// RevokeRefreshToken marca um token como revogado (usado na rotação).
func (s *Store) RevokeRefreshToken(id string) error {
	_, err := s.db.Exec(`UPDATE refresh_tokens SET revoked = 1 WHERE id = ?`, id)
	return err
}

// RevokeAllUserTokens revoga todos os refresh tokens de um usuário
// (defesa contra reuso/roubo detectado).
func (s *Store) RevokeAllUserTokens(userID string) error {
	_, err := s.db.Exec(`UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ?`, userID)
	return err
}
