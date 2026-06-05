package nativeapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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

func serverWith(t *testing.T, cfg config.Config) *Server {
	t.Helper()
	tmp := t.TempDir()
	database, err := db.Open(filepath.Join(tmp, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	store := core.New(database)
	hash, _ := auth.HashPassword("pw")
	_, _ = store.CreateUser("admin", hash, true)
	authSvc := auth.NewService("secret", time.Minute, time.Minute, time.Hour)
	art := artwork.New(database, filepath.Join(tmp, "cache"))
	log := slog.New(slog.NewTextHandler(os.NewFile(0, os.DevNull), nil))
	sc := scanner.New(store, filepath.Join(tmp, "music"), "ffprobe", log)
	if cfg.MusicFolder == "" {
		cfg.MusicFolder = filepath.Join(tmp, "music")
	}
	return NewServer(cfg, store, authSvc, sc, art, log, "test")
}

func TestMountWebapp(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "assets"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>spa</html>"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644)

	srv := serverWith(t, config.Config{WebappDir: dir})
	router := srv.Router()

	// Rota de SPA desconhecida → index.html.
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/qualquer/rota", nil))
	if rr.Code != 200 || rr.Body.String() != "<html>spa</html>" {
		t.Fatalf("SPA fallback falhou: %d %q", rr.Code, rr.Body.String())
	}
	// Asset existente é servido.
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/assets/app.js", nil))
	if rr.Code != 200 {
		t.Fatalf("asset esperava 200, %d", rr.Code)
	}
	// Asset inexistente → 404 (não cai no index).
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest("GET", "/assets/nope.js", nil))
	if rr.Code != 404 {
		t.Fatalf("asset inexistente esperava 404, %d", rr.Code)
	}
}

func TestCORS(t *testing.T) {
	srv := serverWith(t, config.Config{AllowedOrigins: "http://app.local"})
	router := srv.Router()

	// Preflight OPTIONS de origem permitida.
	req := httptest.NewRequest("OPTIONS", "/api/v1/me", nil)
	req.Header.Set("Origin", "http://app.local")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("preflight esperava 204, %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "http://app.local" {
		t.Fatalf("CORS header ausente: %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestStreamTranscodeDisabled(t *testing.T) {
	srv := serverWith(t, config.Config{TranscodingEnabled: false})
	seedLibrary(t, srv)
	token := login(t, srv, "admin", "pw")
	mt := mediaToken(t, srv, token)

	var albums struct {
		Items []struct{ ID string } `json:"items"`
	}
	authGet(t, srv, token, "/api/v1/albums", &albums)
	var detail struct {
		Songs []struct{ ID string } `json:"songs"`
	}
	authGet(t, srv, token, "/api/v1/albums/"+albums.Items[0].ID, &detail)

	// format=opus mas transcode desabilitado → direct play (mp3), não ogg.
	rr := raw(t, srv, "GET", "/api/v1/stream/"+detail.Songs[0].ID+"?token="+mt+"&format=opus", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("esperava 200 direct play, %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct == "audio/ogg" {
		t.Fatalf("não deveria transcodificar (ct=%s)", ct)
	}
}

func TestCoverAndStreamErrors(t *testing.T) {
	srv := serverWith(t, config.Config{})
	token := login(t, srv, "admin", "pw")
	mt := mediaToken(t, srv, token)

	// Cover de id inexistente → 404.
	rr := raw(t, srv, "GET", "/api/v1/cover/none?token="+mt, "")
	if rr.Code != 404 {
		t.Fatalf("cover inexistente esperava 404, %d", rr.Code)
	}
	// Stream de faixa inexistente → 404.
	rr = raw(t, srv, "GET", "/api/v1/stream/none?token="+mt, "")
	if rr.Code != 404 {
		t.Fatalf("stream inexistente esperava 404, %d", rr.Code)
	}
}
