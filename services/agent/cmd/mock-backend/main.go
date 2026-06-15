package main

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/services/agent/internal/security"
	"github.com/rs/cors"
)

const (
	listenAddr      = ":8000"
	expectedBackend = "http://localhost:8000"
	challengeTTL    = 60 * time.Second
	challengeBytes  = 32
)

var allowedOrigins = []string{"http://localhost:5173", "http://localhost:5174"}

var (
	randRead       = rand.Read
	listenAndServe = http.ListenAndServe
	runMain        = run
	logFatal       = log.Fatal
)

type challengeEntry struct {
	expiresAt time.Time
}

type challengeCache struct {
	mu    sync.Mutex
	items map[string]challengeEntry
}

func newChallengeCache() *challengeCache {
	return &challengeCache{items: make(map[string]challengeEntry)}
}

func (c *challengeCache) put(challenge string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[challenge] = challengeEntry{expiresAt: time.Now().UTC().Add(challengeTTL)}
}

func (c *challengeCache) consume(challenge string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.items[challenge]
	if !ok {
		return fmt.Errorf("challenge not found or already used")
	}

	delete(c.items, challenge)

	if time.Now().UTC().After(entry.expiresAt) {
		return fmt.Errorf("challenge expired")
	}

	return nil
}

type challengeResponse struct {
	Challenge string `json:"challenge"`
}

type verifyRequest struct {
	Challenge   string `json:"challenge"`
	Backend     string `json:"backend"`
	Origin      string `json:"origin"`
	Purpose     string `json:"purpose"`
	Provider    string `json:"provider"`
	Algorithm   string `json:"algorithm"`
	Signature   string `json:"signature"`
	Certificate string `json:"certificate"`
	SignedAt    string `json:"signedAt"`
}

type verifyUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type verifyResponse struct {
	Success bool       `json:"success"`
	User    verifyUser `json:"user"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type server struct {
	challenges *challengeCache
	freshness  *security.ChallengeFreshnessValidator
}

func run() error {
	s := &server{
		challenges: newChallengeCache(),
		freshness:  security.NewChallengeFreshnessValidator(60),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /localid/challenge", s.handleChallenge)
	mux.HandleFunc("POST /localid/verify", s.handleVerify)

	handler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: false,
	}).Handler(mux)

	log.Printf("mock backend listening on %s", listenAddr)
	return listenAndServe(listenAddr, handler)
}

func main() {
	if err := runMain(); err != nil {
		logFatal(err)
	}
}

func (s *server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, challengeBytes)
	if _, err := randRead(b); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate challenge")
		return
	}

	challenge := base64.RawURLEncoding.EncodeToString(b)
	s.challenges.put(challenge)

	writeJSON(w, http.StatusOK, challengeResponse{Challenge: challenge})
}

func (s *server) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := s.challenges.consume(req.Challenge); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	if !slices.Contains([]string{"RS256", "ES256", "ES384"}, req.Algorithm) {
		writeError(w, http.StatusForbidden, "unsupported algorithm")
		return
	}

	if req.Purpose != "login" {
		writeError(w, http.StatusForbidden, "purpose is not allowed")
		return
	}

	if req.Backend != expectedBackend {
		writeError(w, http.StatusForbidden, "backend is not allowed")
		return
	}

	if !slices.Contains(allowedOrigins, req.Origin) {
		writeError(w, http.StatusForbidden, "origin is not allowed")
		return
	}

	signedAt, err := time.Parse(time.RFC3339, req.SignedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, "signedAt must be RFC3339")
		return
	}

	if err := s.freshness.ValidateTimestamp(signedAt, time.Now().UTC()); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	signReq := protocol.SignChallengeRequest{
		Challenge: req.Challenge,
		Backend:   req.Backend,
		Origin:    req.Origin,
		Purpose:   req.Purpose,
	}

	payload, err := security.BuildCanonicalPayload(signReq, signedAt)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cert, err := parseCertificate(req.Certificate)
	if err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	signature, err := base64.RawURLEncoding.DecodeString(req.Signature)
	if err != nil {
		writeError(w, http.StatusBadRequest, "signature must be valid base64url")
		return
	}

	if err := verifyProofSignature(req.Algorithm, cert, payload, signature); err != nil {
		writeError(w, http.StatusForbidden, "signature verification failed")
		return
	}

	writeJSON(w, http.StatusOK, verifyResponse{
		Success: true,
		User: verifyUser{
			ID:   "mock-user",
			Name: "Mock Dev User",
		},
	})
}

func parseCertificate(certB64 string) (*x509.Certificate, error) {
	if certB64 == "" {
		return nil, fmt.Errorf("certificate is required")
	}

	der, err := base64.StdEncoding.DecodeString(certB64)
	if err != nil {
		return nil, fmt.Errorf("certificate must be valid base64")
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("invalid certificate")
	}

	return cert, nil
}

func verifyProofSignature(algorithm string, cert *x509.Certificate, payload, signature []byte) error {
	switch algorithm {
	case "RS256":
		publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("certificate public key is not RSA")
		}
		hash := sha256.Sum256(payload)
		return rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature)
	case "ES256":
		publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("certificate public key is not ECDSA")
		}
		hash := sha256.Sum256(payload)
		if verifyRawECDSA(publicKey, hash[:], signature) {
			return nil
		}
		return fmt.Errorf("invalid ES256 signature")
	case "ES384":
		publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("certificate public key is not ECDSA")
		}
		hash := sha512.Sum384(payload)
		if verifyRawECDSA(publicKey, hash[:], signature) {
			return nil
		}
		return fmt.Errorf("invalid ES384 signature")
	default:
		return fmt.Errorf("unsupported algorithm")
	}
}

func verifyRawECDSA(publicKey *ecdsa.PublicKey, hash, signature []byte) bool {
	size := (publicKey.Curve.Params().BitSize + 7) / 8
	if len(signature) != 2*size {
		return false
	}

	r := new(big.Int).SetBytes(signature[:size])
	s := new(big.Int).SetBytes(signature[size:])
	return ecdsa.Verify(publicKey, hash, r, s)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
