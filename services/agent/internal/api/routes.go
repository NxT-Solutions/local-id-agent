package api

import (
	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes(r chi.Router) {
	r.Get("/health", s.handleHealth)
	r.Get("/status", s.handleStatus)
	r.With(s.requireJSONContentType).Post("/sign-challenge", s.handleSignChallenge)

	r.Group(func(r chi.Router) {
		r.Use(s.requireDevMode)
		r.Get("/openapi.json", s.handleOpenAPIJSON)
		r.Get("/openapi.yaml", s.handleOpenAPIYAML)
	})
}
