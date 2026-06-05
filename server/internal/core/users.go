package core

import (
	"database/sql"
	"errors"

	"github.com/DiegoModesto/berserker-player/server/internal/model"
)

var ErrNotFound = errors.New("não encontrado")

// UserAuth é o subconjunto de identidade usado na emissão de tokens.
type UserAuth struct {
	ID      string
	IsAdmin bool
}

func (s *Store) CreateUser(username, passwordHash string, isAdmin bool) (model.User, error) {
	u := model.User{
		ID:           NewID(),
		Username:     username,
		PasswordHash: passwordHash,
		IsAdmin:      isAdmin,
		CreatedAt:    nowUTC(),
	}
	_, err := s.db.Exec(
		`INSERT INTO users(id, username, password_hash, is_admin, created_at) VALUES(?,?,?,?,?)`,
		u.ID, u.Username, u.PasswordHash, boolToInt(isAdmin), u.CreatedAt.Format(rfc3339))
	return u, err
}

func (s *Store) UserByUsername(username string) (model.User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE username = ?`, username))
}

func (s *Store) UserByID(id string) (model.User, error) {
	return s.scanUser(s.db.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE id = ?`, id))
}

func (s *Store) CountUsers() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (s *Store) scanUser(row *sql.Row) (model.User, error) {
	var u model.User
	var isAdmin int
	var createdAt string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return u, ErrNotFound
		}
		return u, err
	}
	u.IsAdmin = isAdmin == 1
	u.CreatedAt = parseTime(createdAt)
	return u, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
