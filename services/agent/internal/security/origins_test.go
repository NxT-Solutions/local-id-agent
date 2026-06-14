package security

import (
	"testing"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestOriginValidator_AllowedOrigin(t *testing.T) {
	v := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	err := v.ValidateOrigin("http://localhost:5173")
	assert.NoError(t, err)
}

func TestOriginValidator_UnknownOrigin(t *testing.T) {
	v := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	err := v.ValidateOrigin("http://evil.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestOriginValidator_WildcardRejected(t *testing.T) {
	v := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	err := v.ValidateOrigin("*")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard")
}

func TestOriginValidator_HeaderBodyMismatch(t *testing.T) {
	v := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"https://app.example.com",
		},
	})

	err := v.ValidateOriginHeaderAndBody("http://localhost:5173", "https://app.example.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must match")
}

func TestOriginValidator_HeaderBodyMatch(t *testing.T) {
	v := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{"http://localhost:5173"},
	})

	err := v.ValidateOriginHeaderAndBody("http://localhost:5173", "http://localhost:5173")
	assert.NoError(t, err)
}

func TestOriginValidator_ProductionRejectsWildcardPattern(t *testing.T) {
	v := NewOriginValidator(config.SecurityConfig{
		AllowedOrigins: []string{"https://*.example.com"},
		Production:     true,
	})

	err := v.ValidateOrigin("https://*.example.com")
	assert.Error(t, err)
}
