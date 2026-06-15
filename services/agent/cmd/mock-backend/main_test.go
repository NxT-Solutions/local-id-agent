package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	mockprovider "github.com/rqc-icu/localid-agent/services/agent/internal/providers/mock"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallengeCache(t *testing.T) {
	c := newChallengeCache()
	c.put("abc")
	require.NoError(t, c.consume("abc"))

	err := c.consume("abc")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	c.items["expired"] = challengeEntry{expiresAt: time.Now().UTC().Add(-time.Second)}
	err = c.consume("expired")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestHandleChallenge(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		restore := randRead
		t.Cleanup(func() { randRead = restore })
		randRead = rand.Read

		s := &server{challenges: newChallengeCache()}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/localid/challenge", bytes.NewReader([]byte("{}")))
		s.handleChallenge(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var payload challengeResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
		assert.NotEmpty(t, payload.Challenge)
	})

	t.Run("random read error", func(t *testing.T) {
		restore := randRead
		t.Cleanup(func() { randRead = restore })
		randRead = func(b []byte) (int, error) { return 0, errors.New("rng unavailable") }

		s := &server{challenges: newChallengeCache()}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/localid/challenge", bytes.NewReader([]byte("{}")))
		s.handleChallenge(rec, req)

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "failed to generate challenge")
	})
}

func validVerifyRequest(t *testing.T, challenge string) verifyRequest {
	t.Helper()

	p, err := mockprovider.New()
	require.NoError(t, err)

	signReq := protocol.SignChallengeRequest{
		Challenge: challenge,
		Backend:   expectedBackend,
		Origin:    allowedOrigins[0],
		Purpose:   "login",
	}
	resp, err := p.SignChallenge(context.Background(), signReq)
	require.NoError(t, err)

	return verifyRequest{
		Challenge:   challenge,
		Backend:     expectedBackend,
		Origin:      allowedOrigins[0],
		Purpose:     "login",
		Provider:    resp.Provider,
		Algorithm:   resp.Algorithm,
		Signature:   resp.Signature,
		Certificate: resp.Certificate,
		SignedAt:    resp.SignedAt,
	}
}

func mustDoVerify(t *testing.T, s *server, reqBody verifyRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/localid/verify", bytes.NewReader(body))
	s.handleVerify(rec, req)
	return rec
}

func TestHandleVerify(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/localid/verify", bytes.NewReader([]byte("{")))
		s.handleVerify(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("challenge missing", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		rec := mustDoVerify(t, s, verifyRequest{Challenge: "YWJj"})
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unsupported algorithm", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Algorithm = "HS256"
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid purpose", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Purpose = "sign"
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("backend not allowed", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Backend = "http://evil.example.com"
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("origin not allowed", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Origin = "http://evil.example.com"
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid signedAt", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.SignedAt = "bad"
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("stale challenge timestamp", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(1)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.SignedAt = time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339)
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid challenge payload", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		req.Challenge = "bad$challenge"
		s.challenges.put(req.Challenge)
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("invalid certificate", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Certificate = "invalid"
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid signature base64url", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Signature = "bad==="
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("signature verification failed", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		req.Signature = base64.RawURLEncoding.EncodeToString([]byte("wrong"))
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("success", func(t *testing.T) {
		s := &server{challenges: newChallengeCache(), freshness: security.NewChallengeFreshnessValidator(60)}
		req := validVerifyRequest(t, "YWJj")
		s.challenges.put(req.Challenge)
		rec := mustDoVerify(t, s, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"success":true`)
	})
}

func TestWriteHelpers(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]string{"ok": "yes"})
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	errRec := httptest.NewRecorder()
	writeError(errRec, http.StatusForbidden, "nope")
	assert.Equal(t, http.StatusForbidden, errRec.Code)
	assert.Contains(t, errRec.Body.String(), `"error":"nope"`)
}

func TestRunAndMain(t *testing.T) {
	t.Run("run returns listen error", func(t *testing.T) {
		restore := listenAndServe
		t.Cleanup(func() { listenAndServe = restore })
		listenAndServe = func(addr string, handler http.Handler) error {
			return errors.New("listen failed")
		}

		err := run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "listen failed")
	})

	t.Run("main success path", func(t *testing.T) {
		restoreRun := runMain
		restoreFatal := logFatal
		t.Cleanup(func() {
			runMain = restoreRun
			logFatal = restoreFatal
		})

		fatalCalled := false
		runMain = func() error { return nil }
		logFatal = func(v ...any) { fatalCalled = true }

		main()
		assert.False(t, fatalCalled)
	})

	t.Run("main error path", func(t *testing.T) {
		restoreRun := runMain
		restoreFatal := logFatal
		t.Cleanup(func() {
			runMain = restoreRun
			logFatal = restoreFatal
		})

		fatalCalled := false
		runMain = func() error { return errors.New("boom") }
		logFatal = func(v ...any) { fatalCalled = true }

		main()
		assert.True(t, fatalCalled)
	})
}

func TestParseCertificate(t *testing.T) {
	_, err := parseCertificate("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate is required")

	_, err = parseCertificate("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "valid base64")

	_, err = parseCertificate(base64.StdEncoding.EncodeToString([]byte("not cert")))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid certificate")

	validReq := validVerifyRequest(t, "YWJj")
	cert, err := parseCertificate(validReq.Certificate)
	require.NoError(t, err)
	assert.NotNil(t, cert.PublicKey)
}

func TestVerifyProofSignature(t *testing.T) {
	payload := []byte("canonical-payload")

	t.Run("RS256 success", func(t *testing.T) {
		cert, privateKey := mustCreateRSACert(t)
		hash := sha256.Sum256(payload)
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
		require.NoError(t, err)
		require.NoError(t, verifyProofSignature("RS256", cert, payload, signature))
	})

	t.Run("RS256 wrong public key type", func(t *testing.T) {
		cert, _ := mustCreateECDSACert(t, elliptic.P256())
		err := verifyProofSignature("RS256", cert, payload, []byte("sig"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not RSA")
	})

	t.Run("RS256 invalid signature", func(t *testing.T) {
		cert, _ := mustCreateRSACert(t)
		err := verifyProofSignature("RS256", cert, payload, []byte("invalid-signature"))
		require.Error(t, err)
	})

	t.Run("ES256 success", func(t *testing.T) {
		cert, privateKey := mustCreateECDSACert(t, elliptic.P256())
		hash := sha256.Sum256(payload)
		r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
		require.NoError(t, err)
		size := (privateKey.Curve.Params().BitSize + 7) / 8
		signature := append(padBigInt(r, size), padBigInt(s, size)...)
		require.NoError(t, verifyProofSignature("ES256", cert, payload, signature))
	})

	t.Run("ES256 invalid signature", func(t *testing.T) {
		cert, _ := mustCreateECDSACert(t, elliptic.P256())
		err := verifyProofSignature("ES256", cert, payload, []byte("bad-signature"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ES256 signature")
	})

	t.Run("ES256 wrong public key type", func(t *testing.T) {
		cert, privateKey := mustCreateRSACert(t)
		hash := sha256.Sum256(payload)
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
		require.NoError(t, err)
		err = verifyProofSignature("ES256", cert, payload, signature)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not ECDSA")
	})

	t.Run("ES384 success", func(t *testing.T) {
		cert, privateKey := mustCreateECDSACert(t, elliptic.P384())
		hash := sha512.Sum384(payload)
		r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
		require.NoError(t, err)
		size := (privateKey.Curve.Params().BitSize + 7) / 8
		signature := append(padBigInt(r, size), padBigInt(s, size)...)
		require.NoError(t, verifyProofSignature("ES384", cert, payload, signature))
	})

	t.Run("ES384 wrong public key type", func(t *testing.T) {
		cert, privateKey := mustCreateRSACert(t)
		hash := sha512.Sum384(payload)
		signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA384, hash[:])
		require.NoError(t, err)
		err = verifyProofSignature("ES384", cert, payload, signature)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not ECDSA")
	})

	t.Run("ES384 invalid signature", func(t *testing.T) {
		cert, _ := mustCreateECDSACert(t, elliptic.P384())
		err := verifyProofSignature("ES384", cert, payload, []byte("bad-signature"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid ES384 signature")
	})

	t.Run("unsupported algorithm", func(t *testing.T) {
		cert, _ := mustCreateRSACert(t)
		err := verifyProofSignature("HS256", cert, payload, []byte("sig"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported algorithm")
	})
}

func TestVerifyRawECDSA(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	hash := sha256.Sum256([]byte("payload"))
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	require.NoError(t, err)

	size := (privateKey.Curve.Params().BitSize + 7) / 8
	validSignature := append(padBigInt(r, size), padBigInt(s, size)...)
	assert.True(t, verifyRawECDSA(&privateKey.PublicKey, hash[:], validSignature))
	assert.False(t, verifyRawECDSA(&privateKey.PublicKey, hash[:], []byte("short")))
}

func mustCreateRSACert(t *testing.T) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "RSA Test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	return cert, privateKey
}

func mustCreateECDSACert(t *testing.T, curve elliptic.Curve) (*x509.Certificate, *ecdsa.PrivateKey) {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ECDSA Test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)
	return cert, privateKey
}

func padBigInt(value *big.Int, size int) []byte {
	bytes := value.Bytes()
	if len(bytes) >= size {
		return bytes
	}
	padded := make([]byte, size)
	copy(padded[size-len(bytes):], bytes)
	return padded
}
