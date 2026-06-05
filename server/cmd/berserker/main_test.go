package main

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/auth"
	"github.com/DiegoModesto/berserker-player/server/internal/config"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/db"
)

func TestEnsureSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), "jwt.secret")
	s1, err := ensureSecret(path)
	if err != nil || len(s1) < 32 {
		t.Fatalf("ensureSecret: %q err=%v", s1, err)
	}
	// Persistente: segunda chamada devolve o mesmo segredo.
	s2, _ := ensureSecret(path)
	if s1 != s2 {
		t.Fatal("segredo deveria persistir entre chamadas")
	}
}

func TestNewLoggerAndAddr(t *testing.T) {
	for _, lvl := range []string{"debug", "info", "warn", "error", "qualquer"} {
		if newLogger(lvl) == nil {
			t.Fatalf("logger nil para %q", lvl)
		}
	}
	if announceAddr(4533) != ":4533" {
		t.Fatalf("announceAddr errado: %s", announceAddr(4533))
	}
}

func TestSeedAdmin(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	store := core.New(database)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	cfg := config.Config{AdminUser: "admin", AdminPassword: "secret"}
	if err := seedAdmin(store, cfg, log); err != nil {
		t.Fatal(err)
	}
	u, err := store.UserByUsername("admin")
	if err != nil || !u.IsAdmin {
		t.Fatalf("admin não criado: %+v err=%v", u, err)
	}
	ok, _ := auth.VerifyPassword("secret", u.PasswordHash)
	if !ok {
		t.Fatal("senha do admin não confere")
	}
	// Idempotente: não cria segundo usuário.
	if err := seedAdmin(store, cfg, log); err != nil {
		t.Fatal(err)
	}
	if n, _ := store.CountUsers(); n != 1 {
		t.Fatalf("esperava 1 usuário, %d", n)
	}
}

func TestRunGracefulShutdown(t *testing.T) {
	tmp := t.TempDir()
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"berserker",
		"--port", "0", // porta efêmera
		"--data", tmp,
		"--music", filepath.Join(tmp, "music"),
		"--admin-password", "pw",
		"--log-level", "error",
	}

	done := make(chan error, 1)
	go func() { done <- run() }()
	time.Sleep(700 * time.Millisecond) // deixa subir

	// SIGTERM → shutdown gracioso (run() intercepta o sinal).
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGTERM)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run() retornou erro: %v", err)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("run() não encerrou após SIGTERM")
	}
}

func TestBuildServer(t *testing.T) {
	tmp := t.TempDir()
	cfg, _ := config.Load(nil)
	cfg.DataFolder = tmp
	cfg.MusicFolder = filepath.Join(tmp, "music")
	cfg.JWTSecret = "test-secret-test-secret-test-secret"
	cfg.AdminPassword = "pw"
	cfg.ScanOnStart = false // evita scan em background no teste
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	srv, database, err := buildServer(cfg, log)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	// O handler montado responde ao healthz.
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, httptest.NewRequest("GET", "/healthz", nil))
	if rr.Code != 200 {
		t.Fatalf("healthz via buildServer: %d", rr.Code)
	}
}

func TestSeedAdminGeneratedPassword(t *testing.T) {
	database, _ := db.Open(filepath.Join(t.TempDir(), "t.db"))
	defer database.Close()
	store := core.New(database)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	// Sem senha → gera uma.
	if err := seedAdmin(store, config.Config{AdminUser: "admin"}, log); err != nil {
		t.Fatal(err)
	}
	if n, _ := store.CountUsers(); n != 1 {
		t.Fatalf("esperava 1 usuário")
	}
}
