package security

import (
	"fmt"
	"slices"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
)

type OriginValidator struct {
	allowedOrigins []string
	production     bool
}

func NewOriginValidator(cfg config.SecurityConfig) *OriginValidator {
	return &OriginValidator{
		allowedOrigins: cfg.AllowedOrigins,
		production:     cfg.Production,
	}
}

func (v *OriginValidator) ValidateOrigin(origin string) error {
	if origin == "" {
		return fmt.Errorf("origin is required")
	}

	if origin == "*" {
		return fmt.Errorf("wildcard origins are not allowed")
	}

	if v.production && containsWildcard(origin) {
		return fmt.Errorf("wildcard origins are not allowed in production mode")
	}

	if !slices.Contains(v.allowedOrigins, origin) {
		return fmt.Errorf("origin is not allowed")
	}

	return nil
}

func (v *OriginValidator) ValidateOriginHeaderAndBody(headerOrigin, bodyOrigin string) error {
	if err := v.ValidateOrigin(headerOrigin); err != nil {
		return err
	}

	if err := v.ValidateOrigin(bodyOrigin); err != nil {
		return err
	}

	if headerOrigin != bodyOrigin {
		return fmt.Errorf("origin header and body must match")
	}

	return nil
}

func containsWildcard(origin string) bool {
	return origin == "*" || containsChar(origin, '*')
}

func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}
