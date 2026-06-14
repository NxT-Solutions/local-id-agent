package mock

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
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
	assert.Equal(t, "mock", p.Name())

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

func TestProvider_SignChallenge_ValidationError(t *testing.T) {
	p, err := New()
	require.NoError(t, err)

	_, err = p.SignChallenge(context.Background(), protocol.SignChallengeRequest{
		Challenge: "",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "challenge is required")
}

func TestProvider_SignChallenge_SigningError(t *testing.T) {
	p, err := New()
	require.NoError(t, err)
	p.privateKey = &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: big.NewInt(0),
			E: 0,
		},
	}

	_, err = p.SignChallenge(context.Background(), protocol.SignChallengeRequest{
		Challenge: "YWJj",
		Backend:   "http://localhost:8000",
		Origin:    "http://localhost:5173",
		Purpose:   "login",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sign payload")
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

func TestNewErrors(t *testing.T) {
	t.Run("invalid private key pem", func(t *testing.T) {
		restore := privateKeyPEM
		t.Cleanup(func() { privateKeyPEM = restore })
		privateKeyPEM = "invalid"

		p, err := New()
		assert.Nil(t, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode private key pem")
	})

	t.Run("invalid private key parse", func(t *testing.T) {
		restore := privateKeyPEM
		t.Cleanup(func() { privateKeyPEM = restore })
		privateKeyPEM = "-----BEGIN PRIVATE KEY-----\naW52YWxpZA==\n-----END PRIVATE KEY-----"

		p, err := New()
		assert.Nil(t, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parse private key")
	})

	t.Run("invalid certificate pem", func(t *testing.T) {
		restore := certificatePEM
		t.Cleanup(func() { certificatePEM = restore })
		certificatePEM = "invalid"

		p, err := New()
		assert.Nil(t, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode certificate pem")
	})

	t.Run("private key not rsa", func(t *testing.T) {
		restore := privateKeyPEM
		t.Cleanup(func() { privateKeyPEM = restore })

		ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
		der, err := x509.MarshalPKCS8PrivateKey(ecKey)
		require.NoError(t, err)
		privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: der,
		}))

		p, err := New()
		assert.Nil(t, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not rsa")
	})

	t.Run("pkcs1 private key fallback", func(t *testing.T) {
		restore := privateKeyPEM
		t.Cleanup(func() { privateKeyPEM = restore })

		block, _ := pem.Decode([]byte(restore))
		require.NotNil(t, block)
		parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		require.NoError(t, err)
		rsaKey, ok := parsed.(*rsa.PrivateKey)
		require.True(t, ok)

		privateKeyPEM = string(pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(rsaKey),
		}))

		p, err := New()
		require.NoError(t, err)
		assert.NotNil(t, p)
	})
}
