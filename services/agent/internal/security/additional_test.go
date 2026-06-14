package security

import (
	"errors"
	"testing"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalCanonical(t *testing.T) {
	payload, err := marshalCanonical(map[string]string{
		"timestamp": "2026-06-14T12:00:00Z",
		"purpose":   "login",
		"origin":    "http://localhost:5173",
		"challenge": "YWJj",
		"backend":   "http://localhost:8000",
	})
	require.NoError(t, err)
	assert.Equal(t, `{"backend":"http://localhost:8000","challenge":"YWJj","origin":"http://localhost:5173","purpose":"login","timestamp":"2026-06-14T12:00:00Z"}`, string(payload))
}

func TestMarshalCanonicalError(t *testing.T) {
	restore := canonicalMarshal
	t.Cleanup(func() { canonicalMarshal = restore })
	canonicalMarshal = func(v any) ([]byte, error) { return nil, errors.New("marshal fail") }

	_, err := marshalCanonical(map[string]string{"backend": "x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal canonical payload")
}

func TestChallengeFreshnessValidator_AdditionalEdges(t *testing.T) {
	v := NewChallengeFreshnessValidator(60)
	now := time.Date(2026, 6, 14, 12, 1, 0, 0, time.UTC)

	err := v.ValidateTimestamp(time.Time{}, now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp is required")

	err = v.ValidateTimestamp(now.Add(10*time.Second), now)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "future")

	// boundary condition: exactly max age old is accepted
	err = v.ValidateTimestamp(now.Add(-60*time.Second), now)
	assert.NoError(t, err)
}

func TestBuildCanonicalPayload_FieldValidation(t *testing.T) {
	ts := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	base := protocol.SignChallengeRequest{
		Challenge: "YWJj",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	}

	_, err := BuildCanonicalPayload(protocol.SignChallengeRequest{
		Challenge: "",
		Backend:   base.Backend,
		Origin:    base.Origin,
		Purpose:   base.Purpose,
	}, ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "challenge is required")

	_, err = BuildCanonicalPayload(protocol.SignChallengeRequest{
		Challenge: base.Challenge,
		Backend:   "",
		Origin:    base.Origin,
		Purpose:   base.Purpose,
	}, ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend is required")

	_, err = BuildCanonicalPayload(protocol.SignChallengeRequest{
		Challenge: base.Challenge,
		Backend:   base.Backend,
		Origin:    "",
		Purpose:   base.Purpose,
	}, ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "origin is required")

	_, err = BuildCanonicalPayload(protocol.SignChallengeRequest{
		Challenge: base.Challenge,
		Backend:   base.Backend,
		Origin:    base.Origin,
		Purpose:   "",
	}, ts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "purpose is required")
}

func TestValidateChallengeAndHelpers(t *testing.T) {
	assert.NoError(t, ValidateChallenge("abc-XYZ_0123"))

	err := ValidateChallenge("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required")

	err = ValidateChallenge("abc=")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base64url")

	assert.Equal(t, "http://localhost:8000", stringsTrimCheck("http://localhost:8000///"))
	assert.Equal(t, "", stringsTrimCheck(""))
	assert.True(t, isValidBase64URL("abc-_012"))
	assert.False(t, isValidBase64URL("abc$"))
}

func TestBackendAndOriginValidatorExtraBranches(t *testing.T) {
	backendValidator := NewBackendValidator(config.SecurityConfig{
		AllowedBackends: []string{"http://localhost:8000"},
	})
	err := backendValidator.ValidateBackend("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend is required")

	originValidator := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
		Production:     true,
	})
	err = originValidator.ValidateOrigin("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "origin is required")

	err = originValidator.ValidateOriginHeaderAndBody("http://localhost:5173", "http://evil.example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	err = originValidator.ValidateOriginHeaderAndBody("http://evil.example.com", "http://localhost:5173")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")

	assert.True(t, containsWildcard("*"))
	assert.True(t, containsWildcard("https://*.example.com"))
	assert.False(t, containsWildcard("https://app.example.com"))
	assert.True(t, containsChar("abc", 'b'))
	assert.False(t, containsChar("abc", 'z'))
}
