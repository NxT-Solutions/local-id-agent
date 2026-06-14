package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/rqc-icu/localid-agent/services/agent/internal/config"
	"github.com/rqc-icu/localid-agent/services/agent/internal/providers"
	mockprovider "github.com/rqc-icu/localid-agent/services/agent/internal/providers/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunErrors(t *testing.T) {
	t.Run("flag parse error", func(t *testing.T) {
		err := run([]string{"-bad-flag"})
		require.Error(t, err)
	})

	t.Run("missing config", func(t *testing.T) {
		err := run(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing required --config flag")
	})

	t.Run("load config error", func(t *testing.T) {
		restoreLoad := loadConfig
		t.Cleanup(func() { loadConfig = restoreLoad })
		loadConfig = func(path string) (*config.Config, error) {
			return nil, errors.New("load failed")
		}

		err := run([]string{"--config", "config.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load config")
	})

	t.Run("provider create error", func(t *testing.T) {
		restoreLoad := loadConfig
		restoreProvider := newProvider
		t.Cleanup(func() {
			loadConfig = restoreLoad
			newProvider = restoreProvider
		})

		loadConfig = func(path string) (*config.Config, error) {
			return &config.Config{}, nil
		}
		newProvider = func(cfg config.ProvidersConfig) (providers.Provider, error) {
			return nil, errors.New("provider failed")
		}

		err := run([]string{"--config", "config.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create provider")
	})

	t.Run("server run error", func(t *testing.T) {
		restoreLoad := loadConfig
		restoreProvider := newProvider
		restoreNotify := notifyContext
		t.Cleanup(func() {
			loadConfig = restoreLoad
			newProvider = restoreProvider
			notifyContext = restoreNotify
		})

		p, err := mockprovider.New()
		require.NoError(t, err)

		loadConfig = func(path string) (*config.Config, error) {
			return &config.Config{
				Server: config.ServerConfig{
					Host: "127.0.0.1",
					Port: -1,
				},
				Security: config.SecurityConfig{
					AllowedOrigins:  []string{"http://localhost:5173"},
					AllowedBackends: []string{"http://localhost:8000"},
				},
			}, nil
		}
		newProvider = func(cfg config.ProvidersConfig) (providers.Provider, error) { return p, nil }
		notifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
			return context.WithCancel(parent)
		}

		err = run([]string{"--config", "config.json"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "server error")
	})
}

func TestRunSuccess(t *testing.T) {
	restoreLoad := loadConfig
	restoreProvider := newProvider
	restoreNotify := notifyContext
	restoreSetup := setupLogging
	t.Cleanup(func() {
		loadConfig = restoreLoad
		newProvider = restoreProvider
		notifyContext = restoreNotify
		setupLogging = restoreSetup
	})

	p, err := mockprovider.New()
	require.NoError(t, err)

	loadConfig = func(path string) (*config.Config, error) {
		return &config.Config{
			Server: config.ServerConfig{
				Host: "127.0.0.1",
				Port: 0,
			},
			Security: config.SecurityConfig{
				AllowedOrigins:  []string{"http://localhost:5173"},
				AllowedBackends: []string{"http://localhost:8000"},
			},
			Logging: config.LoggingConfig{Level: "info"},
		}, nil
	}
	newProvider = func(cfg config.ProvidersConfig) (providers.Provider, error) { return p, nil }
	setupLogging = func(level string) *slog.Logger { return slog.Default() }
	notifyContext = func(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
		ctx, cancel := context.WithCancel(parent)
		cancel()
		return ctx, func() {}
	}

	require.NoError(t, run([]string{"--config", "config.json"}))
}

func TestMain(t *testing.T) {
	t.Run("no exit on success", func(t *testing.T) {
		restoreRun := runMain
		restoreExit := exitMain
		t.Cleanup(func() {
			runMain = restoreRun
			exitMain = restoreExit
		})

		exitCode := 0
		runMain = func(args []string) error { return nil }
		exitMain = func(code int) { exitCode = code }

		main()
		assert.Equal(t, 0, exitCode)
	})

	t.Run("exit on error", func(t *testing.T) {
		restoreRun := runMain
		restoreExit := exitMain
		t.Cleanup(func() {
			runMain = restoreRun
			exitMain = restoreExit
		})

		exitCode := 0
		runMain = func(args []string) error { return errors.New("boom") }
		exitMain = func(code int) { exitCode = code }

		main()
		assert.Equal(t, 1, exitCode)
	})
}
