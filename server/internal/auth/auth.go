// Package auth cuida de hashing de senha (argon2id), emissão/validação de JWT
// (access e media tokens) e geração de refresh tokens opacos.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidToken = errors.New("token inválido")
	ErrWrongKind    = errors.New("tipo de token inesperado")
)

// ---- Senhas (argon2id) ----

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLen     uint32
	keyLen      uint32
}

var defaultArgon = argonParams{memory: 64 * 1024, iterations: 1, parallelism: 4, saltLen: 16, keyLen: 32}

// HashPassword retorna um hash argon2id codificado (formato PHC).
func HashPassword(password string) (string, error) {
	p := defaultArgon
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.memory, p.iterations, p.parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyPassword compara a senha com o hash codificado (constante no tempo).
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("hash inválido")
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, err
	}
	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// ---- JWT (access + media) ----

type Service struct {
	secret     []byte
	accessTTL  time.Duration
	mediaTTL   time.Duration
	refreshTTL time.Duration
}

func NewService(secret string, accessTTL, mediaTTL, refreshTTL time.Duration) *Service {
	return &Service{secret: []byte(secret), accessTTL: accessTTL, mediaTTL: mediaTTL, refreshTTL: refreshTTL}
}

type Claims struct {
	jwt.RegisteredClaims
	Kind    string `json:"knd"`
	IsAdmin bool   `json:"adm,omitempty"`
}

func (s *Service) sign(userID, kind string, isAdmin bool, ttl time.Duration, now time.Time) (string, time.Time, error) {
	exp := now.Add(ttl)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
		Kind:    kind,
		IsAdmin: isAdmin,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	return signed, exp, err
}

// AccessToken emite um JWT de acesso de vida curta.
func (s *Service) AccessToken(userID string, isAdmin bool, now time.Time) (string, time.Time, error) {
	return s.sign(userID, "access", isAdmin, s.accessTTL, now)
}

// MediaToken emite um token de vida curta para /stream e /cover.
func (s *Service) MediaToken(userID string, now time.Time) (string, time.Time, error) {
	return s.sign(userID, "media", false, s.mediaTTL, now)
}

func (s *Service) parse(token, kind string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	if claims.Kind != kind {
		return nil, ErrWrongKind
	}
	return claims, nil
}

func (s *Service) ParseAccess(token string) (*Claims, error) { return s.parse(token, "access") }
func (s *Service) ParseMedia(token string) (*Claims, error)  { return s.parse(token, "media") }

func (s *Service) RefreshTTL() time.Duration { return s.refreshTTL }

// ---- Refresh tokens (opacos) ----

// NewRefreshToken gera um token opaco (retornado ao cliente) e seu hash (guardado no DB).
func NewRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashToken(token), nil
}

// HashToken retorna o sha256 hex de um token opaco.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
