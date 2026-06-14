package mock

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Status(t *testing.T) {
	p, err := New()
	require.NoError(t, err)

	status, err := p.Status(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "mock", status.Provider)
	assert.True(t, status.Ready)
	assert.True(t, status.CardPresent)
}

func TestProvider_SignChallenge_DeterministicVerification(t *testing.T) {
	p, err := New()
	require.NoError(t, err)

	req := protocol.SignChallengeRequest{
		Challenge: "YWJj",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	}

	resp1, err := p.SignChallenge(context.Background(), req)
	require.NoError(t, err)

	resp2, err := p.SignChallenge(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "mock", resp1.Provider)
	assert.Equal(t, "RS256", resp1.Algorithm)
	assert.Equal(t, req.Challenge, resp1.Challenge)
	assert.NotEmpty(t, resp1.Signature)
	assert.NotEmpty(t, resp1.Certificate)
	assert.NotEmpty(t, resp1.SignedAt)

	// Signatures differ because timestamp changes, but each must verify.
	for _, resp := range []*protocol.SignChallengeResponse{resp1, resp2} {
		verifySignature(t, p, req, resp)
	}
}

func TestProvider_SignChallenge_CanonicalPayloadMatches(t *testing.T) {
	p, err := New()
	require.NoError(t, err)

	req := protocol.SignChallengeRequest{
		Challenge: "YWJj",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	}

	resp, err := p.SignChallenge(context.Background(), req)
	require.NoError(t, err)

	signedAt, err := time.Parse(time.RFC3339, resp.SignedAt)
	require.NoError(t, err)

	payload, err := security.BuildCanonicalPayload(req, signedAt)
	require.NoError(t, err)

	sig, err := base64.RawURLEncoding.DecodeString(resp.Signature)
	require.NoError(t, err)

	hash := sha256.Sum256(payload)
	err = rsa.VerifyPKCS1v15(p.PublicKey(), crypto.SHA256, hash[:], sig)
	assert.NoError(t, err)
}

func verifySignature(t *testing.T, p *Provider, req protocol.SignChallengeRequest, resp *protocol.SignChallengeResponse) {
	t.Helper()

	signedAt, err := time.Parse(time.RFC3339, resp.SignedAt)
	require.NoError(t, err)

	payload, err := security.BuildCanonicalPayload(req, signedAt)
	require.NoError(t, err)

	sig, err := base64.RawURLEncoding.DecodeString(resp.Signature)
	require.NoError(t, err)

	hash := sha256.Sum256(payload)
	err = rsa.VerifyPKCS1v15(p.PublicKey(), crypto.SHA256, hash[:], sig)
	assert.NoError(t, err)
}
