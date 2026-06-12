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
	SuperTokensAPIKey  string `json:"SUPERTOKENS_API_KEY" yaml:"SUPERTOKENS_API_KEY" env:"SUPERTOKENS_API_KEY"`
	// Hero carousel IDs
	HeroBuildID  string `json:"HERO_BUILD_ID"  yaml:"HERO_BUILD_ID"  env:"HERO_BUILD_ID"`
	HeroGarageID string `json:"HERO_GARAGE_ID" yaml:"HERO_GARAGE_ID" env:"HERO_GARAGE_ID"`
	HeroClubID   string `json:"HERO_CLUB_ID"   yaml:"HERO_CLUB_ID"   env:"HERO_CLUB_ID"`
	// Object Storage
	R2AccountID                    string `json:"R2_ACCOUNT_ID"                     yaml:"R2_ACCOUNT_ID"                     env:"R2_ACCOUNT_ID"`
	R2AccessKeyID                  string `json:"R2_ACCESS_KEY_ID"                  yaml:"R2_ACCESS_KEY_ID"                  env:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey              string `json:"R2_SECRET_ACCESS_KEY"              yaml:"R2_SECRET_ACCESS_KEY"              env:"R2_SECRET_ACCESS_KEY"`
	R2AccountPhotosBucketName      string `json:"R2_ACCOUNT_PHOTOS_BUCKET_NAME"     yaml:"R2_ACCOUNT_PHOTOS_BUCKET_NAME"     env:"R2_ACCOUNT_PHOTOS_BUCKET_NAME"`
	R2CarPhotosBucketName          string `json:"R2_CAR_PHOTOS_BUCKET_NAME"         yaml:"R2_CAR_PHOTOS_BUCKET_NAME"         env:"R2_CAR_PHOTOS_BUCKET_NAME"`
	R2AccountPhotosBucketPublicURL string `json:"R2_ACCOUNT_PHOTOS_BUCKET_PUBLIC_URL" yaml:"R2_ACCOUNT_PHOTOS_BUCKET_PUBLIC_URL" env:"R2_ACCOUNT_PHOTOS_BUCKET_PUBLIC_URL"`
	R2CarPhotosBucketPublicURL     string `json:"R2_CAR_PHOTOS_BUCKET_PUBLIC_URL"   yaml:"R2_CAR_PHOTOS_BUCKET_PUBLIC_URL"   env:"R2_CAR_PHOTOS_BUCKET_PUBLIC_URL"`
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
