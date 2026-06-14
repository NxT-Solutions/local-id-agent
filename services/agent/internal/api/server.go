package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/providers"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
)

const Version = "0.1.0"

const shutdownTimeout = 10 * time.Second

const maxBodySize = 64 << 10 // 64 KB

var shutdownServer = func(server *http.Server, ctx context.Context) error {
	return server.Shutdown(ctx)
}

type Server struct {
	cfg              *config.Config
	provider         providers.Provider
	logger           *slog.Logger
	httpServer       *http.Server
	originValidator  *security.OriginValidator
	backendValidator *security.BackendValidator
}

func NewServer(cfg *config.Config, provider providers.Provider, logger *slog.Logger) *Server {
	return &Server{
		cfg:              cfg,
		provider:         provider,
		logger:           logger,
		originValidator:  security.NewOriginValidator(cfg.Security),
		backendValidator: security.NewBackendValidator(cfg.Security),
	}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(s.loggingMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(s.corsMiddleware)
	r.Use(s.maxBodyMiddleware)

	s.registerRoutes(r)

	s.httpServer = &http.Server{
		Addr:    s.cfg.Addr(),
		Handler: r,
	}

	return r
}

func (s *Server) Run(ctx context.Context) error {
	s.Handler()

	s.logger.Info("starting server", "addr", s.cfg.Addr(), "provider", s.provider.Name())

	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		s.logger.Info("shutting down server")
		if err := shutdownServer(s.httpServer, shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Error("failed to encode json response", "error", err)
	}
}
