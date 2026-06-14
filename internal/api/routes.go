package api

import (
	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes(r chi.Router) {
	r.Get("/health", s.handleHealth)
	r.Get("/status", s.handleStatus)
	r.With(s.requireJSONContentType).Post("/sign-challenge", s.handleSignChallenge)
}
