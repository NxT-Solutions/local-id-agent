package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"github.com/rqc-icu/localid-agent/internal/protocol"
)

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins:   s.cfg.Security.AllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type", "Origin"},
		AllowCredentials: false,
	})
	return c.Handler(next)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}

func (s *Server) maxBodyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if !isJSONContentType(contentType) {
			protocol.WriteUnsupportedMediaType(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isJSONContentType(contentType string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return mediaType == "application/json"
}

func jsonDecode(r *http.Request, dst any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if err == io.EOF {
			return fmt.Errorf("request body is required")
		}
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errPayloadTooLarge
		}
		return fmt.Errorf("invalid json body")
	}

	if decoder.More() {
		return fmt.Errorf("invalid json body")
	}

	return nil
}

var errPayloadTooLarge = errors.New("payload too large")

func writeDecodeError(w http.ResponseWriter, err error) {
	if errors.Is(err, errPayloadTooLarge) {
		protocol.WritePayloadTooLarge(w)
		return
	}
	protocol.WriteBadRequest(w, err.Error())
}
