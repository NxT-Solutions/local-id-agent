package belgian_eid

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/rqc-icu/localid-agent/services/agent/internal/providers/pkcs11"
)

const moduleOverrideEnv = "LOCALID_BEID_PKCS11_MODULE"

type pkcs11Provider interface {
	Status(ctx context.Context) (*protocol.Status, error)
	SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error)
}

type Provider struct {
	cfg config.BelgianEIDConfig

	newPKCS11 func(config.PKCS11Config) (pkcs11Provider, error)
}

func New(cfg config.BelgianEIDConfig) (*Provider, error) {
	return &Provider{
		cfg: cfg,
		newPKCS11: func(pkcs11Cfg config.PKCS11Config) (pkcs11Provider, error) {
			return pkcs11.New(pkcs11Cfg)
		},
	}, nil
}

func (p *Provider) Name() string {
	return "belgian_eid"
}

func (p *Provider) Status(ctx context.Context) (*protocol.Status, error) {
	delegate, err := p.newDelegate()
	if err != nil {
		return nil, err
	}

	status, err := delegate.Status(ctx)
	if err != nil {
		return nil, err
	}

	status.Provider = p.Name()
	return status, nil
}

func (p *Provider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	delegate, err := p.newDelegate()
	if err != nil {
		return nil, err
	}

	resp, err := delegate.SignChallenge(ctx, req)
	if err != nil {
		return nil, err
	}

	resp.Provider = p.Name()
	return resp, nil
}

func (p *Provider) newDelegate() (pkcs11Provider, error) {
	delegate, err := p.newPKCS11(config.PKCS11Config{
		Enabled:    true,
		ModulePath: p.effectiveModulePath(),
		PINPrompt:  "terminal",
	})
	if err != nil {
		return nil, fmt.Errorf("initialize PKCS#11 provider: %w", err)
	}
	return delegate, nil
}

func (p *Provider) effectiveModulePath() string {
	if override := strings.TrimSpace(os.Getenv(moduleOverrideEnv)); override != "" {
		return override
	}

	if configured := strings.TrimSpace(p.cfg.PKCS11ModulePath); configured != "" {
		return configured
	}

	return "auto"
}
