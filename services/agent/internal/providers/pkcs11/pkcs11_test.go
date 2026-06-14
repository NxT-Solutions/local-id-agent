package pkcs11

import (
	"context"
	"testing"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	p, err := New(config.PKCS11Config{
		Enabled:          true,
		ModulePath:       "/tmp/pkcs11.so",
		TokenLabel:       "token",
		CertificateLabel: "cert",
		PINPrompt:        "pin",
	})
	require.NoError(t, err)
	assert.Equal(t, "pkcs11", p.Name())
	assert.Equal(t, "/tmp/pkcs11.so", p.cfg.ModulePath)

	status, err := p.Status(context.Background())
	assert.Nil(t, status)
	assert.ErrorIs(t, err, protocol.ErrNotImplemented)

	resp, err := p.SignChallenge(context.Background(), protocol.SignChallengeRequest{})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, protocol.ErrNotImplemented)
}
