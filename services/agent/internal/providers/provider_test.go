package providers

import (
	"context"
	"testing"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("mock", func(t *testing.T) {
		p, err := New(config.ProvidersConfig{Default: "mock"})
		require.NoError(t, err)
		assert.Equal(t, "mock", p.Name())
	})

	t.Run("pkcs11", func(t *testing.T) {
		p, err := New(config.ProvidersConfig{
			Default: "pkcs11",
			PKCS11:  config.PKCS11Config{Enabled: true, ModulePath: "/tmp/module"},
		})
		require.NoError(t, err)
		assert.Equal(t, "pkcs11", p.Name())
		status, err := p.Status(context.Background())
		assert.Nil(t, status)
		assert.ErrorIs(t, err, protocol.ErrNotImplemented)
	})

	t.Run("belgian_eid", func(t *testing.T) {
		p, err := New(config.ProvidersConfig{
			Default:    "belgian_eid",
			BelgianEID: config.BelgianEIDConfig{Enabled: true, PKCS11ModulePath: "/tmp/module"},
		})
		require.NoError(t, err)
		assert.Equal(t, "belgian_eid", p.Name())
		status, err := p.Status(context.Background())
		assert.Nil(t, status)
		assert.ErrorIs(t, err, protocol.ErrNotImplemented)
	})

	t.Run("unknown", func(t *testing.T) {
		p, err := New(config.ProvidersConfig{Default: "unknown"})
		assert.Nil(t, p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}
