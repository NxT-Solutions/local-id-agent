package belgian_eid

import (
	"context"
	"errors"
	"testing"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePKCS11Provider struct {
	statusResp *protocol.Status
	statusErr  error
	signResp   *protocol.SignChallengeResponse
	signErr    error
}

func (p *fakePKCS11Provider) Status(ctx context.Context) (*protocol.Status, error) {
	return p.statusResp, p.statusErr
}

func (p *fakePKCS11Provider) SignChallenge(ctx context.Context, req protocol.SignChallengeRequest) (*protocol.SignChallengeResponse, error) {
	return p.signResp, p.signErr
}

func TestProviderBasics(t *testing.T) {
	p, err := New(config.BelgianEIDConfig{
		Enabled:          true,
		PKCS11ModulePath: "/tmp/beid-pkcs11.so",
	})
	require.NoError(t, err)
	assert.Equal(t, "belgian_eid", p.Name())
	assert.Equal(t, "/tmp/beid-pkcs11.so", p.cfg.PKCS11ModulePath)
	assert.NotNil(t, p.newPKCS11)

	_, err = p.newPKCS11(config.PKCS11Config{ModulePath: "/definitely/missing/module.so"})
	require.NoError(t, err)
}

func TestProviderStatusAndSign(t *testing.T) {
	t.Run("status delegates and rewrites provider", func(t *testing.T) {
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		p.newPKCS11 = func(config.PKCS11Config) (pkcs11Provider, error) {
			return &fakePKCS11Provider{
				statusResp: &protocol.Status{
					Provider:    "pkcs11",
					Ready:       true,
					CardPresent: true,
				},
			}, nil
		}

		status, err := p.Status(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "belgian_eid", status.Provider)
		assert.True(t, status.Ready)
		assert.True(t, status.CardPresent)
	})

	t.Run("status returns delegate error", func(t *testing.T) {
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		p.newPKCS11 = func(config.PKCS11Config) (pkcs11Provider, error) {
			return &fakePKCS11Provider{statusErr: errors.New("boom")}, nil
		}

		status, err := p.Status(context.Background())
		assert.Nil(t, status)
		require.Error(t, err)
	})

	t.Run("sign delegates and rewrites provider", func(t *testing.T) {
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		p.newPKCS11 = func(config.PKCS11Config) (pkcs11Provider, error) {
			return &fakePKCS11Provider{
				signResp: &protocol.SignChallengeResponse{
					Provider:  "pkcs11",
					Algorithm: "RS256",
					Challenge: "YWJj",
				},
			}, nil
		}

		resp, err := p.SignChallenge(context.Background(), protocol.SignChallengeRequest{Challenge: "YWJj"})
		require.NoError(t, err)
		assert.Equal(t, "belgian_eid", resp.Provider)
		assert.Equal(t, "RS256", resp.Algorithm)
	})

	t.Run("sign returns delegate error", func(t *testing.T) {
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		p.newPKCS11 = func(config.PKCS11Config) (pkcs11Provider, error) {
			return &fakePKCS11Provider{signErr: errors.New("no card")}, nil
		}

		resp, err := p.SignChallenge(context.Background(), protocol.SignChallengeRequest{})
		assert.Nil(t, resp)
		require.Error(t, err)
	})

	t.Run("delegate creation failure", func(t *testing.T) {
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		p.newPKCS11 = func(config.PKCS11Config) (pkcs11Provider, error) {
			return nil, errors.New("init failed")
		}

		status, err := p.Status(context.Background())
		assert.Nil(t, status)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "initialize PKCS#11 provider")
	})

	t.Run("sign handles delegate creation failure", func(t *testing.T) {
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		p.newPKCS11 = func(config.PKCS11Config) (pkcs11Provider, error) {
			return nil, errors.New("init failed")
		}

		resp, err := p.SignChallenge(context.Background(), protocol.SignChallengeRequest{})
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "initialize PKCS#11 provider")
	})
}

func TestEffectiveModulePath(t *testing.T) {
	t.Run("uses env override", func(t *testing.T) {
		t.Setenv(moduleOverrideEnv, "/override/module.so")
		p, err := New(config.BelgianEIDConfig{PKCS11ModulePath: "/config/module.so"})
		require.NoError(t, err)
		assert.Equal(t, "/override/module.so", p.effectiveModulePath())
	})

	t.Run("uses config path", func(t *testing.T) {
		t.Setenv(moduleOverrideEnv, "")
		p, err := New(config.BelgianEIDConfig{PKCS11ModulePath: "/config/module.so"})
		require.NoError(t, err)
		assert.Equal(t, "/config/module.so", p.effectiveModulePath())
	})

	t.Run("defaults to auto", func(t *testing.T) {
		t.Setenv(moduleOverrideEnv, "")
		p, err := New(config.BelgianEIDConfig{})
		require.NoError(t, err)
		assert.Equal(t, "auto", p.effectiveModulePath())
	})
}
