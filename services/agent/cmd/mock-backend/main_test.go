package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
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
