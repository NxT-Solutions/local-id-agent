package pkcs11

import (
	"context"

	"github.com/rqc-icu/localid-agent/internal/config"
	"github.com/rqc-icu/localid-agent/internal/protocol"
)

type Provider struct {
	cfg config.PKCS11Config
}

func New(cfg config.PKCS11Config) (*Provider, error) {
	return &Provider{cfg: cfg}, nil
}

func (p *Provider) Name() string {
	return "pkcs11"
}

func (p *Provider) Status(ctx context.Context) (*protocol.Status, error) {
	return nil, protocol.ErrNotImplemented
}

func (p *Provider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	return nil, protocol.ErrNotImplemented
}
