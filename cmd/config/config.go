package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/caarlos0/env/v11"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Database
	DatabaseURL      string `json:"DATABASE_URL"         yaml:"DATABASE_URL"         env:"DATABASE_URL"`
	SuperTokensDBURL string `json:"SUPERTOKENS_DB_URL"   yaml:"SUPERTOKENS_DB_URL"   env:"SUPERTOKENS_DB_URL"`
	// Base Server
	ServerPort string `json:"SERVER_PORT" yaml:"SERVER_PORT" env:"SERVER_PORT"`
	PublicURL  string `json:"PUBLIC_URL"  yaml:"PUBLIC_URL"  env:"PUBLIC_URL"`
	// Auth
	SuperTokensCoreURL string `json:"SUPERTOKENS_CORE_URL"  yaml:"SUPERTOKENS_CORE_URL"  env:"SUPERTOKENS_CORE_URL"`
	SuperTokensAPIKey  string `json:"SUPER_TOKENS_API_KEY" yaml:"SUPER_TOKENS_API_KEY" env:"SUPER_TOKENS_API_KEY"`
}

type LoadOption func(*loadOptions)

type loadOptions struct {
	configFile string
}

func WithConfigFile(path string) LoadOption {
	return func(o *loadOptions) {
		o.configFile = path
	}
}

func Load(opts ...LoadOption) (Config, error) {
	options := &loadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	cfg := Config{}

	if options.configFile != "" {
		if err := loadFromFile(options.configFile, &cfg); err != nil {
			return Config{}, err
		}
	}

	// env vars take precedence over file values
	if err := env.Parse(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func loadFromFile(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	switch filepath.Ext(path) {
	case ".json":
		return json.NewDecoder(f).Decode(cfg)
	case ".yaml", ".yml":
		return yaml.NewDecoder(f).Decode(cfg)
	default:
		return fmt.Errorf("unsupported config file format: %s", filepath.Ext(path))
	}
}
