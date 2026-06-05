// Command berserker é o servidor BerserkerPlayer.
package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/api/nativeapi"
	"github.com/DiegoModesto/berserker-player/server/internal/artwork"
	"github.com/DiegoModesto/berserker-player/server/internal/auth"
	"github.com/DiegoModesto/berserker-player/server/internal/config"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/db"
	"github.com/DiegoModesto/berserker-player/server/internal/scanner"
)

var version = "0.1.0-dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "erro fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return err
	}

	log := newLogger(cfg.LogLevel)

	if err := os.MkdirAll(cfg.DataFolder, 0o755); err != nil {
		return fmt.Errorf("data folder: %w", err)
	}

	// Secret JWT: usa config/env ou gera e persiste em data/.
	if cfg.JWTSecret == "" {
		cfg.JWTSecret, err = ensureSecret(filepath.Join(cfg.DataFolder, "jwt.secret"))
		if err != nil {
			return err
		}
	}

	httpSrv, database, err := buildServer(cfg, log)
	if err != nil {
		return err
	}
	defer database.Close()

	// Shutdown gracioso.
	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor ouvindo", "addr", httpSrv.Addr, "version", version)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-sig:
		log.Info("encerrando…")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return httpSrv.Shutdown(ctx)
	}
}

// buildServer faz toda a montagem (db, store, serviços, seed, scans) e devolve
// o *http.Server pronto (sem escutar) e a conexão. Separado de run() para teste.
func buildServer(cfg config.Config, log *slog.Logger) (*http.Server, *sql.DB, error) {
	database, err := db.Open(filepath.Join(cfg.DataFolder, "berserker.db"))
	if err != nil {
		return nil, nil, fmt.Errorf("db: %w", err)
	}

	store := core.New(database)
	authSvc := auth.NewService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.MediaTokenTTL, cfg.RefreshTokenTTL)
	art := artwork.New(database, filepath.Join(cfg.DataFolder, "cache", "artwork"))
	sc := scanner.New(store, cfg.MusicFolder, cfg.FFprobePath, log)

	if err := seedAdmin(store, cfg, log); err != nil {
		database.Close()
		return nil, nil, err
	}

	if cfg.ScanOnStart {
		go func() {
			log.Info("iniciando scan", "music", cfg.MusicFolder)
			if _, err := sc.Scan(context.Background()); err != nil {
				log.Error("scan falhou", "err", err)
			}
		}()
	}
	if cfg.Watch {
		go func() {
			if err := sc.Watch(context.Background(), 2*time.Second); err != nil {
				log.Error("watcher falhou", "err", err)
			}
		}()
	}
	if cfg.ScanInterval > 0 {
		go func() {
			t := time.NewTicker(cfg.ScanInterval)
			defer t.Stop()
			for range t.C {
				if _, err := sc.Scan(context.Background()); err != nil {
					log.Error("scan periódico falhou", "err", err)
				}
			}
		}()
	}

	srv := nativeapi.NewServer(cfg, store, authSvc, sc, art, log, version)
	httpSrv := &http.Server{
		Addr:              announceAddr(cfg.Port),
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return httpSrv, database, nil
}

func announceAddr(port int) string { return fmt.Sprintf(":%d", port) }

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func ensureSecret(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil && len(b) >= 32 {
		return string(b), nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return "", err
	}
	return secret, nil
}

func seedAdmin(store *core.Store, cfg config.Config, log *slog.Logger) error {
	n, err := store.CountUsers()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	password := cfg.AdminPassword
	generated := false
	if password == "" {
		buf := make([]byte, 9)
		_, _ = rand.Read(buf)
		password = hex.EncodeToString(buf)
		generated = true
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	if _, err := store.CreateUser(cfg.AdminUser, hash, true); err != nil {
		return err
	}
	if generated {
		log.Warn("admin criado com senha gerada — troque-a", "user", cfg.AdminUser, "password", password)
	} else {
		log.Info("admin criado", "user", cfg.AdminUser)
	}
	return nil
}
