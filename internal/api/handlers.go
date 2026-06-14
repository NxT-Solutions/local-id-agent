package api

import (
	"net/http"

	"github.com/rqc-icu/localid-agent/internal/protocol"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, protocol.HealthResponse{
		OK:      true,
		Name:    "LocalID Agent",
		Version: Version,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.provider.Status(r.Context())
	if err != nil {
		s.logger.Error("provider status failed", "error", err)
		protocol.WriteInternalError(w)
		return
	}

	s.writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleSignChallenge(w http.ResponseWriter, r *http.Request) {
	originHeader := r.Header.Get("Origin")
	if err := s.originValidator.ValidateOrigin(originHeader); err != nil {
		protocol.WriteForbidden(w, err.Error())
		return
	}

	var req protocol.SignChallengeRequest
	if err := jsonDecode(r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := s.originValidator.ValidateOriginHeaderAndBody(originHeader, req.Origin); err != nil {
		protocol.WriteForbidden(w, err.Error())
		return
	}

	if err := s.backendValidator.ValidateBackend(req.Backend); err != nil {
		protocol.WriteForbidden(w, err.Error())
		return
	}

	if req.Purpose == "" {
		protocol.WriteBadRequest(w, "purpose is required")
		return
	}

	if req.Purpose != "login" {
		protocol.WriteForbidden(w, "purpose is not allowed")
		return
	}

	resp, err := s.provider.SignChallenge(r.Context(), req)
	if err != nil {
		s.logger.Error("sign challenge failed", "error", err)
		protocol.WriteBadRequest(w, err.Error())
		return
	}

	s.writeJSON(w, http.StatusOK, resp)
}
