package security

import (
	"fmt"
	"slices"
	"time"

	"github.com/rqc-icu/localid-agent/internal/config"
	"github.com/rqc-icu/localid-agent/internal/protocol"
)

type BackendValidator struct {
	allowedBackends []string
}

func NewBackendValidator(cfg config.SecurityConfig) *BackendValidator {
	return &BackendValidator{
		allowedBackends: cfg.AllowedBackends,
	}
}

func (v *BackendValidator) ValidateBackend(backend string) error {
	if backend == "" {
		return fmt.Errorf("backend is required")
	}

	if backend != stringsTrimCheck(backend) {
		return fmt.Errorf("backend must not have trailing slashes")
	}

	if !slices.Contains(v.allowedBackends, backend) {
		return fmt.Errorf("backend is not allowed")
	}

	return nil
}

func stringsTrimCheck(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// BuildCanonicalPayload constructs the compact JSON payload signed by providers.
//
// Canonicalization rules (Laravel M5 must mirror):
//  1. Keys sorted alphabetically: backend, challenge, origin, purpose, timestamp
//  2. Compact JSON (no extra whitespace)
//  3. timestamp is agent UTC time at sign moment (RFC3339)
//  4. challenge must be non-empty valid base64url
func BuildCanonicalPayload(req protocol.SignChallengeRequest, timestamp time.Time) ([]byte, error) {
	if err := ValidateChallenge(req.Challenge); err != nil {
		return nil, err
	}

	if req.Backend == "" {
		return nil, fmt.Errorf("backend is required")
	}

	if req.Origin == "" {
		return nil, fmt.Errorf("origin is required")
	}

	if req.Purpose == "" {
		return nil, fmt.Errorf("purpose is required")
	}

	payload := map[string]string{
		"backend":   req.Backend,
		"challenge": req.Challenge,
		"origin":    req.Origin,
		"purpose":   req.Purpose,
		"timestamp": timestamp.UTC().Format(time.RFC3339),
	}

	return marshalCanonical(payload)
}

func ValidateChallenge(challenge string) error {
	if challenge == "" {
		return fmt.Errorf("challenge is required")
	}

	if !isValidBase64URL(challenge) {
		return fmt.Errorf("challenge must be valid base64url")
	}

	return nil
}

func isValidBase64URL(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'A' && c <= 'Z',
			c >= 'a' && c <= 'z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
			continue
		default:
			return false
		}
	}
	return true
}
