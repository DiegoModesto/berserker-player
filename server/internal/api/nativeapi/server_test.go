package nativeapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/artwork"
	"github.com/DiegoModesto/berserker-player/server/internal/auth"
	"github.com/DiegoModesto/berserker-player/server/internal/config"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/db"
	"github.com/DiegoModesto/berserker-player/server/internal/scanner"
)

func newTestServer(t *testing.T) (*Server, *core.Store) {
	t.Helper()
	tmp := t.TempDir()
	database, err := db.Open(filepath.Join(tmp, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	store := core.New(database)
	hash, _ := auth.HashPassword("pw")
	if _, err := store.CreateUser("admin", hash, true); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{}
	authSvc := auth.NewService("secret", time.Minute, time.Minute, time.Hour)
	art := artwork.New(database, filepath.Join(tmp, "cache"))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	sc := scanner.New(store, filepath.Join(tmp, "music"), "ffprobe", log)
	return NewServer(cfg, store, authSvc, sc, art, log, "test"), store
}

func TestHealthz(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
}

func login(t *testing.T, srv *Server, user, pass string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": user, "password": pass})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("login status %d: %s", rr.Code, rr.Body.String())
	}
	var tp tokenPair
	_ = json.Unmarshal(rr.Body.Bytes(), &tp)
	return tp.AccessToken
}

func TestLoginAndMe(t *testing.T) {
	srv, _ := newTestServer(t)
	token := login(t, srv, "admin", "pw")
	if token == "" {
		t.Fatal("access token vazio")
	}

	req := httptest.NewRequest("GET", "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("me status %d", rr.Code)
	}
	var u core.UserAuth
	var raw map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &raw)
	_ = u
	if raw["username"] != "admin" || raw["isAdmin"] != true {
		t.Fatalf("me inesperado: %v", raw)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "x"})
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body)))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401, obtive %d", rr.Code)
	}
}

func TestMediaTokenAndStreamAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	token := login(t, srv, "admin", "pw")

	// Emite media token.
	req := httptest.NewRequest("POST", "/api/v1/auth/media-token", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("media-token status %d", rr.Code)
	}

	// Stream sem token -> 401.
	rr2 := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr2, httptest.NewRequest("GET", "/api/v1/stream/xyz", nil))
	if rr2.Code != http.StatusUnauthorized {
		t.Fatalf("stream sem token deveria ser 401, obtive %d", rr2.Code)
	}
}

func TestProtectedRequiresAuth(t *testing.T) {
	srv, _ := newTestServer(t)
	rr := httptest.NewRecorder()
	srv.Router().ServeHTTP(rr, httptest.NewRequest("GET", "/api/v1/me", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("esperava 401, obtive %d", rr.Code)
	}
}
