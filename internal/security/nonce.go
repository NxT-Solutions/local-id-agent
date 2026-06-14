package security

import (
	"fmt"
	"time"
)

type ChallengeFreshnessValidator struct {
	maxAge time.Duration
}

func NewChallengeFreshnessValidator(maxAgeSeconds int) *ChallengeFreshnessValidator {
	return &ChallengeFreshnessValidator{
		maxAge: time.Duration(maxAgeSeconds) * time.Second,
	}
}

func (v *ChallengeFreshnessValidator) ValidateTimestamp(signedAt time.Time, now time.Time) error {
	if signedAt.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	age := now.Sub(signedAt.UTC())
	if age < 0 {
		return fmt.Errorf("timestamp is in the future")
	}

	if age > v.maxAge {
		return fmt.Errorf("challenge timestamp is too old")
	}

	return nil
}
