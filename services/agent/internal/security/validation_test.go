package security

import (
	"testing"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendValidator_AllowedBackend(t *testing.T) {
	v := NewBackendValidator(config.SecurityConfig{
		AllowedBackends: []string{"http://localhost:8000"},
	})

	err := v.ValidateBackend("http://localhost:8000")
	assert.NoError(t, err)
}

func TestBackendValidator_UnknownBackend(t *testing.T) {
	v := NewBackendValidator(config.SecurityConfig{
		AllowedBackends: []string{"http://localhost:8000"},
	})

	err := v.ValidateBackend("https://evil.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestBackendValidator_TrailingSlashRejected(t *testing.T) {
	v := NewBackendValidator(config.SecurityConfig{
		AllowedBackends: []string{"http://localhost:8000"},
	})

	err := v.ValidateBackend("http://localhost:8000/")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "trailing slashes")
}

func TestBuildCanonicalPayload_ByteStability(t *testing.T) {
	req := protocol.SignChallengeRequest{
		Challenge: "YWJj",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	}
	ts := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)

	payload, err := BuildCanonicalPayload(req, ts)
	require.NoError(t, err)

	expected := `{"backend":"http://localhost:8000","challenge":"YWJj","origin":"http://localhost:5173","purpose":"login","timestamp":"2026-06-14T12:00:00Z"}`
	assert.Equal(t, expected, string(payload))
}

func TestBuildCanonicalPayload_InvalidChallenge(t *testing.T) {
	req := protocol.SignChallengeRequest{
		Challenge: "not valid!!!",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	}

	_, err := BuildCanonicalPayload(req, time.Now().UTC())
	assert.Error(t, err)
}

func TestChallengeFreshnessValidator(t *testing.T) {
	v := NewChallengeFreshnessValidator(60)
	now := time.Date(2026, 6, 14, 12, 1, 0, 0, time.UTC)

	err := v.ValidateTimestamp(time.Date(2026, 6, 14, 12, 0, 30, 0, time.UTC), now)
	assert.NoError(t, err)

	err = v.ValidateTimestamp(time.Date(2026, 6, 14, 11, 0, 0, 0, time.UTC), now)
	assert.Error(t, err)
}
