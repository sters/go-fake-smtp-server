package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	// SMTP Server Configuration
	SMTPAddr              string        `env:"SMTP_ADDR"                envDefault:"127.0.0.1:10025"`
	SMTPHostname          string        `env:"SMTP_HOSTNAME"            envDefault:"fakeserver"`
	SMTPReadTimeout       time.Duration `env:"SMTP_READ_TIMEOUT"        envDefault:"10s"`
	SMTPWriteTimeout      time.Duration `env:"SMTP_WRITE_TIMEOUT"       envDefault:"10s"`
	SMTPMaxMessageBytes   int64         `env:"SMTP_MAX_MESSAGE_BYTES"   envDefault:"1048576"` // 1MB
	SMTPMaxRecipients     int           `env:"SMTP_MAX_RECIPIENTS"      envDefault:"50"`
	SMTPAllowInsecureAuth bool          `env:"SMTP_ALLOW_INSECURE_AUTH" envDefault:"true"`

	// HTTP Server Configuration
	ViewAddr              string        `env:"VIEW_ADDR"                envDefault:"127.0.0.1:11080"`
	ViewReadHeaderTimeout time.Duration `env:"VIEW_READ_HEADER_TIMEOUT" envDefault:"10s"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return cfg, nil
}
