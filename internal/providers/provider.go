package providers

import (
	"context"
	"fmt"

	"github.com/rqc-icu/localid-agent/internal/config"
	"github.com/rqc-icu/localid-agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/internal/providers/belgian_eid"
	"github.com/rqc-icu/localid-agent/internal/providers/mock"
	"github.com/rqc-icu/localid-agent/internal/providers/pkcs11"
)

type Provider interface {
	Name() string
	Status(ctx context.Context) (*protocol.Status, error)
	SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error)
}

func New(cfg config.ProvidersConfig) (Provider, error) {
	switch cfg.Default {
	case "mock":
		return mock.New()
	case "pkcs11":
		return pkcs11.New(cfg.PKCS11)
	case "belgian_eid":
		return belgian_eid.New(cfg.BelgianEID)
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Default)
	}
}
