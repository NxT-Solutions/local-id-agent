package belgian_eid

import (
	"context"
	"testing"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	p, err := New(config.BelgianEIDConfig{
		Enabled:          true,
		PKCS11ModulePath: "/tmp/beid-pkcs11.so",
	})
	require.NoError(t, err)
	assert.Equal(t, "belgian_eid", p.Name())
	assert.Equal(t, "/tmp/beid-pkcs11.so", p.cfg.PKCS11ModulePath)

	status, err := p.Status(context.Background())
	assert.Nil(t, status)
	assert.ErrorIs(t, err, protocol.ErrNotImplemented)

	resp, err := p.SignChallenge(context.Background(), protocol.SignChallengeRequest{})
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, protocol.ErrNotImplemented)
}
