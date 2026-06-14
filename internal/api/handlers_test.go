package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rqc-icu/localid-agent/internal/config"
	"github.com/rqc-icu/localid-agent/internal/logging"
	"github.com/rqc-icu/localid-agent/internal/providers/mock"
	"github.com/rqc-icu/localid-agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T) *Server {
	t.Helper()

	provider, err := mock.New()
	require.NoError(t, err)

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

	logger := logging.Setup("error")
	return NewServer(cfg, provider, logger)
}

func TestHandleSignChallenge_HappyPath(t *testing.T) {
	s := testServer(t)
	server := httptest.NewServer(s.Handler())
	defer server.Close()

	body := map[string]string{
		"challenge": "YWJj",
		"backend":   "http://localhost:8000",
		"purpose":   "login",
		"origin":    "http://localhost:5173",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result protocol.SignChallengeResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Equal(t, "mock", result.Provider)
	assert.Equal(t, "RS256", result.Algorithm)
	assert.Equal(t, "YWJj", result.Challenge)
	assert.NotEmpty(t, result.Signature)
}

func TestHandleSignChallenge_UnknownOrigin(t *testing.T) {
	s := testServer(t)
	server := httptest.NewServer(s.Handler())
	defer server.Close()

	body := map[string]string{
		"challenge": "YWJj",
		"backend":   "http://localhost:8000",
		"purpose":   "login",
		"origin":    "http://evil.example.com",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Origin", "http://evil.example.com")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestHandleSignChallenge_UnknownBackend(t *testing.T) {
	s := testServer(t)
	server := httptest.NewServer(s.Handler())
	defer server.Close()

	body := map[string]string{
		"challenge": "YWJj",
		"backend":   "https://evil.example.com",
		"purpose":   "login",
		"origin":    "http://localhost:5173",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestHandleSignChallenge_WrongContentType(t *testing.T) {
	s := testServer(t)
	server := httptest.NewServer(s.Handler())
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/sign-challenge", bytes.NewReader([]byte("not json")))
	require.NoError(t, err)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
}

func TestHandleHealth(t *testing.T) {
	s := testServer(t)
	server := httptest.NewServer(s.Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result protocol.HealthResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.True(t, result.OK)
	assert.Equal(t, "LocalID Agent", result.Name)
	assert.Equal(t, Version, result.Version)
}
