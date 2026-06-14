package belgian_eid

import (
	"context"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
)

type Provider struct {
	cfg config.BelgianEIDConfig
}

func New(cfg config.BelgianEIDConfig) (*Provider, error) {
	return &Provider{cfg: cfg}, nil
}

func (p *Provider) Name() string {
	return "belgian_eid"
}

func (p *Provider) Status(ctx context.Context) (*protocol.Status, error) {
	return nil, protocol.ErrNotImplemented
}

func (p *Provider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	return nil, protocol.ErrNotImplemented
}
