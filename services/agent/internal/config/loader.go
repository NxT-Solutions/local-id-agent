package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	if c.Server.Host == "0.0.0.0" && !c.Server.AllowRemoteBind {
		return fmt.Errorf("server.host 0.0.0.0 requires server.allow_remote_bind: true")
	}

	if len(c.Security.AllowedOrigins) == 0 {
		return fmt.Errorf("security.allowed_origins must not be empty")
	}

	if len(c.Security.AllowedBackends) == 0 {
		return fmt.Errorf("security.allowed_backends must not be empty")
	}

	if c.Security.ChallengeMaxAgeSeconds <= 0 {
		c.Security.ChallengeMaxAgeSeconds = 60
	}

	for _, origin := range c.Security.AllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("wildcard origins are not allowed")
		}
	}

	if c.Security.Production {
		for _, origin := range c.Security.AllowedOrigins {
			if strings.Contains(origin, "*") {
				return fmt.Errorf("wildcard origins are not allowed in production mode")
			}
		}
	}

	if c.Providers.Default == "" {
		c.Providers.Default = "mock"
	}

	switch c.Providers.Default {
	case "mock":
		if !c.Providers.Mock.Enabled {
			return fmt.Errorf("default provider mock is not enabled")
		}
	case "pkcs11":
		if !c.Providers.PKCS11.Enabled {
			return fmt.Errorf("default provider pkcs11 is not enabled")
		}
	case "belgian_eid":
		if !c.Providers.BelgianEID.Enabled {
			return fmt.Errorf("default provider belgian_eid is not enabled")
		}
	default:
		return fmt.Errorf("unknown default provider: %s", c.Providers.Default)
	}

	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}

	switch strings.ToLower(c.Logging.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("logging.level must be one of debug, info, warn, error")
	}

	return nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
