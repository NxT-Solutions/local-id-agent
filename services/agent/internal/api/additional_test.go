package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/services/agent/internal/providers"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type writeFailResponseWriter struct {
	headers http.Header
	status  int
}

func (w *writeFailResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *writeFailResponseWriter) WriteHeader(statusCode int)  { w.status = statusCode }
func (w *writeFailResponseWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }

type stubProvider struct {
	statusResp *protocol.Status
	statusErr  error
	signResp   *protocol.SignChallengeResponse
	signErr    error
}

func (p *stubProvider) Name() string { return "stub" }
func (p *stubProvider) Status(ctx context.Context) (*protocol.Status, error) {
	return p.statusResp, p.statusErr
}
func (p *stubProvider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	return p.signResp, p.signErr
}

func newServerWithProvider(t *testing.T, provider providers.Provider) *Server {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "127.0.0.1",
			Port: 17443,
		},
		Security: config.SecurityConfig{
			AllowedOrigins:  []string{"http://localhost:5173"},
			AllowedBackends: []string{"http://localhost:8000"},
		},
	}
	return &Server{
		cfg:              cfg,
		provider:         provider,
		logger:           slog.Default(),
		originValidator:  security.NewOriginValidator(cfg.Security),
		backendValidator: security.NewBackendValidator(cfg.Security),
	}
}

func TestHandleStatus(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{
			statusResp: &protocol.Status{Provider: "stub", Ready: true, CardPresent: true},
		})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		resp, err := http.Get(server.URL + "/status")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("provider error", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{statusErr: errors.New("boom")})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		resp, err := http.Get(server.URL + "/status")
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestHandleSignChallengeDecodeAndValidationErrors(t *testing.T) {
	t.Run("missing body", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", http.NoBody)
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("invalid json", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewBufferString("{"))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("multiple json objects", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewBufferString(`{} {}`))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("too large payload", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		oversized := map[string]string{
			"challenge": string(bytes.Repeat([]byte("a"), maxBodySize)),
			"backend":   "http://localhost:8000",
			"purpose":   "login",
			"origin":    "http://localhost:5173",
		}
		body, err := json.Marshal(oversized)
		require.NoError(t, err)

		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)
	})

	t.Run("header body origin mismatch", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		payload := []byte(`{"challenge":"YWJj","backend":"http://localhost:8000","purpose":"login","origin":"http://evil.example.com"}`)
		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("missing purpose", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		payload := []byte(`{"challenge":"YWJj","backend":"http://localhost:8000","origin":"http://localhost:5173"}`)
		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("unsupported purpose", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		payload := []byte(`{"challenge":"YWJj","backend":"http://localhost:8000","purpose":"sign","origin":"http://localhost:5173"}`)
		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("provider sign error", func(t *testing.T) {
		s := newServerWithProvider(t, &stubProvider{
			signErr: errors.New("sign failure"),
		})
		server := httptest.NewServer(s.Handler())
		defer server.Close()

		payload := []byte(`{"challenge":"YWJj","backend":"http://localhost:8000","purpose":"login","origin":"http://localhost:5173"}`)
		req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
		require.NoError(t, err)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHandleOpenAPIYAMLAndContentTypeHelpers(t *testing.T) {
	s := testServer(t)
	s.cfg.Server.DevMode = true
	server := httptest.NewServer(s.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/openapi.yaml")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/yaml", resp.Header.Get("Content-Type"))

	assert.True(t, isJSONContentType("application/json"))
	assert.True(t, isJSONContentType("Application/Json; charset=UTF-8"))
	assert.False(t, isJSONContentType("text/plain"))
}

func TestHandleOpenAPIWriterErrors(t *testing.T) {
	s := testServer(t)
	w := &writeFailResponseWriter{}

	s.handleOpenAPIJSON(w, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	assert.Equal(t, http.StatusInternalServerError, w.status)

	w = &writeFailResponseWriter{}
	s.handleOpenAPIYAML(w, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	assert.Equal(t, http.StatusInternalServerError, w.status)
}

func TestWriteJSONWithUnencodablePayload(t *testing.T) {
	s := testServer(t)
	rec := httptest.NewRecorder()
	s.writeJSON(rec, http.StatusOK, map[string]any{"bad": make(chan int)})

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRunReturnsServerError(t *testing.T) {
	s := newServerWithProvider(t, &stubProvider{
		statusResp: &protocol.Status{Provider: "stub", Ready: true},
	})
	s.cfg.Server.Host = "127.0.0.1"
	s.cfg.Server.Port = -1

	err := s.Run(context.Background())
	require.Error(t, err)
}

func TestRunGracefulShutdown(t *testing.T) {
	s := newServerWithProvider(t, &stubProvider{
		statusResp: &protocol.Status{Provider: "stub", Ready: true},
	})
	s.cfg.Server.Port = 0

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}
}

func TestRunShutdownError(t *testing.T) {
	restore := shutdownServer
	t.Cleanup(func() { shutdownServer = restore })
	shutdownServer = func(server *http.Server, ctx context.Context) error {
		return errors.New("shutdown failed")
	}

	s := newServerWithProvider(t, &stubProvider{
		statusResp: &protocol.Status{Provider: "stub", Ready: true},
	})
	s.cfg.Server.Port = 0

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- s.Run(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "shutdown failed")
	case <-time.After(3 * time.Second):
		t.Fatal("server did not return in time")
	}
}
