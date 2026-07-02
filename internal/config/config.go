package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Postgres
	Host     string
	Port     string
	User     string
	Password string
	DB       string

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	SMTPFromName string

	// Server
	ServerPort string

	// Cloudflare R2
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2Endpoint        string
	R2PublicURL       string

	// Email verification
	EmailVerifierMethod string // "default", "smtp", or "off"

	// Public URL
	BaseURL string

	// Rate limiting
	RateLimitMaxAttempts  int
	RateLimitWindow      time.Duration
	RateLimitLockDuration time.Duration
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
			Host:         os.Getenv("POSTGRES_HOST"),
			Port:         os.Getenv("POSTGRES_PORT"),
			User:         os.Getenv("POSTGRES_USER"),
			Password:     os.Getenv("POSTGRES_PASSWORD"),
			DB:           os.Getenv("POSTGRES_DB"),
			SMTPHost:     os.Getenv("SMTP_HOST"),
			SMTPPort:     os.Getenv("SMTP_PORT"),
			SMTPUser:     os.Getenv("SMTP_USER"),
			SMTPPassword: os.Getenv("SMTP_PASSWORD"),
			SMTPFrom:     os.Getenv("SMTP_FROM"),
			SMTPFromName: os.Getenv("SMTP_FROM_NAME"),
			ServerPort:       envWithDefault("SERVER_PORT", "8080"),
			R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
			R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
			R2BucketName:      os.Getenv("R2_BUCKET_NAME"),
			R2Endpoint:        os.Getenv("R2_ENDPOINT"),
			R2PublicURL:       os.Getenv("R2_PUBLIC_URL"),

			EmailVerifierMethod: envWithDefault("EMAIL_VERIFIER_METHOD", "default"),
			BaseURL:            envWithDefault("BASE_URL", "http://localhost:"+envWithDefault("SERVER_PORT", "8080")),

			RateLimitMaxAttempts:  envIntWithDefault("RATE_LIMIT_MAX_ATTEMPTS", 5),
			RateLimitWindow:      envDurationWithDefault("RATE_LIMIT_WINDOW", 15*time.Minute),
			RateLimitLockDuration: envDurationWithDefault("RATE_LIMIT_LOCK_DURATION", 30*time.Minute),
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

	switch cfg.EmailVerifierMethod {
	case "default", "smtp", "off":
	default:
		return fmt.Errorf("invalid EMAIL_VERIFIER_METHOD %q: must be default, smtp, or off", cfg.EmailVerifierMethod)
	}

	return nil
}

func envWithDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntWithDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envDurationWithDefault(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil && d > 0 {
			return d
		}
	}
	return def
}

func (cfg *Config) PostgresURL() string {
	return (&url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   cfg.DB,
	}).String()
}
