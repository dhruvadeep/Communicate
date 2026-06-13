package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

var (
	cfg     *Config
	loadErr error
	once    sync.Once
)

// Load reads the environment once on first use and reuses the config afterwards.
func Load() (*Config, error) {
	once.Do(func() {
		if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
			loadErr = fmt.Errorf("load .env: %w", err)
			return
		}

		cfg = &Config{
			Host:     os.Getenv("POSTGRES_HOST"),
			Port:     os.Getenv("POSTGRES_PORT"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			DB:       os.Getenv("POSTGRES_DB"),
		}

		loadErr = validate(cfg)
	})

	return cfg, loadErr
}

func validate(cfg *Config) error {
	if cfg.Host == "" {
		return fmt.Errorf("missing required env: POSTGRES_HOST")
	}
	if cfg.Port == "" {
		return fmt.Errorf("missing required env: POSTGRES_PORT")
	}
	if cfg.User == "" {
		return fmt.Errorf("missing required env: POSTGRES_USER")
	}
	if cfg.Password == "" {
		return fmt.Errorf("missing required env: POSTGRES_PASSWORD")
	}
	if cfg.DB == "" {
		return fmt.Errorf("missing required env: POSTGRES_DB")
	}

	return nil
}

func (cfg *Config) PostgresURL() string {
	return (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   cfg.DB,
	}).String()
}
