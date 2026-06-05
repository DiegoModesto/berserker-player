// Package nativeapi expõe a API REST nativa (/api/v1) descrita em openapi.yaml.
package nativeapi

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DiegoModesto/berserker-player/server/internal/artwork"
	"github.com/DiegoModesto/berserker-player/server/internal/auth"
	"github.com/DiegoModesto/berserker-player/server/internal/config"
	"github.com/DiegoModesto/berserker-player/server/internal/core"
	"github.com/DiegoModesto/berserker-player/server/internal/scanner"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	cfg     config.Config
	store   *core.Store
	authSvc *auth.Service
	scanner *scanner.Scanner
	art     *artwork.Resolver
	log     *slog.Logger
	version string
}

func NewServer(cfg config.Config, store *core.Store, authSvc *auth.Service, sc *scanner.Scanner, art *artwork.Resolver, log *slog.Logger, version string) *Server {
	return &Server{cfg: cfg, store: store, authSvc: authSvc, scanner: sc, art: art, log: log, version: version}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(s.requestLogger)
	if s.cfg.AllowedOrigins != "" {
		r.Use(s.cors)
	}

	r.Get("/healthz", s.handleHealthz)

	r.Route("/api/v1", func(r chi.Router) {
		// Públicos
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/refresh", s.handleRefresh)
		r.Get("/openapi.yaml", s.handleOpenAPI)

		// Autenticados (Bearer)
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Post("/auth/media-token", s.handleMediaToken)
			r.Get("/me", s.handleMe)

			// Admin
			r.Post("/admin/scan", s.handleTriggerScan)
			r.Get("/admin/scan/status", s.handleScanStatus)

			s.registerLibraryRoutes(r) // Fase 1: artists/albums/songs/search/playlists/annotations
		})

		// Mídia: auth por ?token= (media token), pois <audio>/AVPlayer não enviam Bearer.
		r.With(s.requireMediaToken).Get("/stream/{id}", s.handleStream)
		r.With(s.requireMediaToken).Get("/cover/{id}", s.handleCover)
	})

	// Servir o WebApp estático (origem única), se configurado.
	if s.cfg.WebappDir != "" {
		s.mountWebapp(r)
	}
	return r
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": s.version})
}

// mountWebapp serve o SPA com fallback para index.html (rotas client-side).
func (s *Server) mountWebapp(r chi.Router) {
	dir := s.cfg.WebappDir
	fs := http.FileServer(http.Dir(dir))
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// API e healthz têm rotas próprias; aqui tratamos só assets/SPA.
		p := filepath.Join(dir, filepath.Clean(req.URL.Path))
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			fs.ServeHTTP(w, req)
			return
		}
		if strings.HasPrefix(req.URL.Path, "/assets/") {
			http.NotFound(w, req)
			return
		}
		http.ServeFile(w, req, filepath.Join(dir, "index.html"))
	})
}

func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		s.log.Debug("http",
			"method", r.Method, "path", r.URL.Path,
			"status", ww.Status(), "bytes", ww.BytesWritten(),
			"dur", time.Since(start).Round(time.Millisecond).String())
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	origins := strings.Split(s.cfg.AllowedOrigins, ",")
	allowed := map[string]bool{}
	for _, o := range origins {
		allowed[strings.TrimSpace(o)] = true
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
