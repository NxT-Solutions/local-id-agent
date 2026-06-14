package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseConfig() Config {
	return Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 17443,
		},
		Security: SecurityConfig{
			AllowedOrigins:  []string{"http://localhost:5173"},
			AllowedBackends: []string{"http://localhost:8000"},
		},
		Providers: ProvidersConfig{
			Default: "mock",
			Mock: MockConfig{
				Enabled: true,
			},
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}

func writeConfigFile(t *testing.T, cfg string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	return path
}

func TestLoad(t *testing.T) {
	path := writeConfigFile(t, `{
		"server": {"host":"127.0.0.1","port":17443},
		"security": {
			"allowed_origins":["http://localhost:5173"],
			"allowed_backends":["http://localhost:8000"]
		},
		"providers": {"default":"mock","mock":{"enabled":true}},
		"logging": {"level":"info"}
	}`)

	cfg, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:17443", cfg.Addr())
}

func TestLoadErrors(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read config")

	path := writeConfigFile(t, `{`)
	_, err = Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config")

	invalidPath := writeConfigFile(t, `{
		"server": {"host":"127.0.0.1","port":17443},
		"security": {
			"allowed_origins":["http://localhost:5173"],
			"allowed_backends":["http://localhost:8000"]
		},
		"providers": {"default":"mock","mock":{"enabled":false}},
		"logging": {"level":"info"}
	}`)
	_, err = Load(invalidPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "default provider mock is not enabled")
}

func TestValidateSuccessWithDefaults(t *testing.T) {
	cfg := baseConfig()
	cfg.Server.Host = ""
	cfg.Security.ChallengeMaxAgeSeconds = 0
	cfg.Providers.Default = ""
	cfg.Logging.Level = ""

	err := cfg.Validate()
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 60, cfg.Security.ChallengeMaxAgeSeconds)
	assert.Equal(t, "mock", cfg.Providers.Default)
	assert.Equal(t, "info", cfg.Logging.Level)
}

func TestValidateErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "invalid port",
			cfg: func() Config {
				c := baseConfig()
				c.Server.Port = 0
				return c
			}(),
			want: "server.port must be between 1 and 65535",
		},
		{
			name: "remote bind not allowed",
			cfg: func() Config {
				c := baseConfig()
				c.Server.Host = "0.0.0.0"
				return c
			}(),
			want: "requires server.allow_remote_bind: true",
		},
		{
			name: "missing origins",
			cfg: func() Config {
				c := baseConfig()
				c.Security.AllowedOrigins = nil
				return c
			}(),
			want: "allowed_origins must not be empty",
		},
		{
			name: "missing backends",
			cfg: func() Config {
				c := baseConfig()
				c.Security.AllowedBackends = nil
				return c
			}(),
			want: "allowed_backends must not be empty",
		},
		{
			name: "wildcard origin rejected",
			cfg: func() Config {
				c := baseConfig()
				c.Security.AllowedOrigins = []string{"*"}
				return c
			}(),
			want: "wildcard origins are not allowed",
		},
		{
			name: "production wildcard rejected",
			cfg: func() Config {
				c := baseConfig()
				c.Security.Production = true
				c.Security.AllowedOrigins = []string{"https://*.example.com"}
				return c
			}(),
			want: "wildcard origins are not allowed in production mode",
		},
		{
			name: "mock provider not enabled",
			cfg: func() Config {
				c := baseConfig()
				c.Providers.Mock.Enabled = false
				return c
			}(),
			want: "default provider mock is not enabled",
		},
		{
			name: "pkcs11 provider not enabled",
			cfg: func() Config {
				c := baseConfig()
				c.Providers.Default = "pkcs11"
				return c
			}(),
			want: "default provider pkcs11 is not enabled",
		},
		{
			name: "belgian eid provider not enabled",
			cfg: func() Config {
				c := baseConfig()
				c.Providers.Default = "belgian_eid"
				return c
			}(),
			want: "default provider belgian_eid is not enabled",
		},
		{
			name: "unknown provider",
			cfg: func() Config {
				c := baseConfig()
				c.Providers.Default = "unknown"
				return c
			}(),
			want: "unknown default provider",
		},
		{
			name: "invalid log level",
			cfg: func() Config {
				c := baseConfig()
				c.Logging.Level = "trace"
				return c
			}(),
			want: "logging.level must be one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}
