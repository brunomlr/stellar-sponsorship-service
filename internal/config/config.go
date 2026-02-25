package config

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sethvargo/go-envconfig"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/network"
)

type Config struct {
	StellarNetwork         string   `env:"STELLAR_NETWORK,required"`
	SigningSecretKey        string   `env:"SIGNING_SECRET_KEY,required"`
	MasterFundingPublicKey string   `env:"MASTER_FUNDING_PUBLIC_KEY,required"`
	DatabaseURL            string   `env:"DATABASE_URL,required"`
	GoogleClientID         string   `env:"GOOGLE_CLIENT_ID,required"`
	GoogleAllowedDomain    string   `env:"GOOGLE_ALLOWED_DOMAIN,required"`
	GoogleAllowedEmails    []string `env:"GOOGLE_ALLOWED_EMAILS,required"`
	Port                   int      `env:"PORT,default=8080"`
	HorizonURL             string   `env:"HORIZON_URL"`
	LogLevel               string   `env:"LOG_LEVEL,default=info"`
	CORSOrigins            []string `env:"CORS_ORIGINS"`

	// HTTP server timeouts
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT,default=15s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT,default=30s"`
	IdleTimeout  time.Duration `env:"HTTP_IDLE_TIMEOUT,default=60s"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if c.StellarNetwork != "testnet" && c.StellarNetwork != "mainnet" {
		return fmt.Errorf("STELLAR_NETWORK must be 'testnet' or 'mainnet', got %q", c.StellarNetwork)
	}

	if !strings.HasPrefix(c.SigningSecretKey, "S") {
		return fmt.Errorf("SIGNING_SECRET_KEY must be a valid Stellar secret key (starts with 'S')")
	}
	if _, err := keypair.ParseFull(c.SigningSecretKey); err != nil {
		return fmt.Errorf("SIGNING_SECRET_KEY is not a valid Stellar secret key: %w", err)
	}

	if _, err := keypair.ParseAddress(c.MasterFundingPublicKey); err != nil {
		return fmt.Errorf("MASTER_FUNDING_PUBLIC_KEY is not a valid Stellar public key: %w", err)
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got %d", c.Port)
	}

	return nil
}

func (c *Config) NetworkPassphrase() string {
	if c.StellarNetwork == "mainnet" {
		return network.PublicNetworkPassphrase
	}
	return network.TestNetworkPassphrase
}

func (c *Config) DefaultHorizonURL() string {
	if c.HorizonURL != "" {
		return c.HorizonURL
	}
	if c.StellarNetwork == "mainnet" {
		return "https://horizon.stellar.org"
	}
	return "https://horizon-testnet.stellar.org"
}
