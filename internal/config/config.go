package config

type Config struct {
	Server    ServerConfig    `json:"server"`
	Security  SecurityConfig  `json:"security"`
	Providers ProvidersConfig `json:"providers"`
	Logging   LoggingConfig   `json:"logging"`
}

type ServerConfig struct {
	Host             string `json:"host"`
	Port             int    `json:"port"`
	AllowRemoteBind  bool   `json:"allow_remote_bind"`
}

type SecurityConfig struct {
	AllowedOrigins          []string `json:"allowed_origins"`
	AllowedBackends         []string `json:"allowed_backends"`
	ChallengeMaxAgeSeconds  int      `json:"challenge_max_age_seconds"`
	Production              bool     `json:"production"`
}

type ProvidersConfig struct {
	Default    string            `json:"default"`
	Mock       MockConfig        `json:"mock"`
	PKCS11     PKCS11Config      `json:"pkcs11"`
	BelgianEID BelgianEIDConfig  `json:"belgian_eid"`
}

type MockConfig struct {
	Enabled bool `json:"enabled"`
}

type PKCS11Config struct {
	Enabled          bool   `json:"enabled"`
	ModulePath       string `json:"module_path"`
	TokenLabel       string `json:"token_label"`
	CertificateLabel string `json:"certificate_label"`
	PINPrompt        string `json:"pin_prompt"`
}

type BelgianEIDConfig struct {
	Enabled          bool   `json:"enabled"`
	PKCS11ModulePath string `json:"pkcs11_module_path"`
}

type LoggingConfig struct {
	Level string `json:"level"`
}
