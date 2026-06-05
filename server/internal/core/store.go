// Package core implementa a lógica de negócio e o acesso a dados (sobre SQLite).
package core

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

// Store agrega o acesso a dados. Escritas são serializadas pelo SQLite (1 conexão).
type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) DB() *sql.DB { return s.db }

// NewID gera um identificador opaco de 16 bytes em hex.
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func nowUTC() time.Time { return time.Now().UTC() }

const rfc3339 = time.RFC3339Nano

func parseTime(s string) time.Time {
	t, _ := time.Parse(rfc3339, s)
	return t
}
