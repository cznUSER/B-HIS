package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config matches 工程约定 §6 plus Auth (H03 v0.3.2).
type Config struct {
	Host     HostConfig     `yaml:"Host"`
	Auth     AuthConfig     `yaml:"Auth"`
	Logging  LoggingConfig  `yaml:"Logging"`
	Database DatabaseConfig `yaml:"Database"`
	CORS     CORSConfig     `yaml:"CORS"`
}

type HostConfig struct {
	ApiAddr string `yaml:"ApiAddr"`
}

type AuthConfig struct {
	JWTSecret      string          `yaml:"JWTSecret"`
	AccessTokenTTL string          `yaml:"AccessTokenTTL"`
	RateLimit      RateLimitConfig `yaml:"RateLimit"`
}

type RateLimitConfig struct {
	MaxFailures int    `yaml:"MaxFailures"`
	Window      string `yaml:"Window"`
}

type LoggingConfig struct {
	Dir   string `yaml:"Dir"`
	Level string `yaml:"Level"`
}

type DatabaseConfig struct {
	Driver string `yaml:"Driver"`
	DSN    string `yaml:"DSN"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"AllowedOrigins"`
}

// Defaults are safe for a local memory demo except JWTSecret, which must be supplied.
func Defaults() Config {
	return Config{
		Host: HostConfig{ApiAddr: "127.0.0.1:8080"},
		Auth: AuthConfig{
			AccessTokenTTL: "8h",
			RateLimit:      RateLimitConfig{MaxFailures: 8, Window: "15m"},
		},
		Logging:  LoggingConfig{Dir: "./logs", Level: "info"},
		Database: DatabaseConfig{Driver: "memory"},
		CORS: CORSConfig{AllowedOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost:4173",
			"http://127.0.0.1:4173",
		}},
	}
}

// Load reads YAML (flag, BHIS_CONFIG, config.local.yaml, config.yaml), then .env.local/.env,
// then process environment. BHIS_DB_DRIVER and BHIS_DB_DSN override the file.
// Missing local files are ignored. This function never logs the DSN.
func Load(path string) (Config, error) {
	cfg := Defaults()
	if path == "" {
		path = os.Getenv("BHIS_CONFIG")
	}
	if path == "" {
		for _, candidate := range []string{"config.local.yaml", "config.yaml"} {
			if _, err := os.Stat(candidate); err == nil {
				path = candidate
				break
			}
		}
	}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config: %w", err)
		}
	}
	if err := loadLocalEnvFiles(); err != nil {
		return Config{}, err
	}
	applyEnv(&cfg)
	if err := cfg.normalize(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("BHIS_API_ADDR"); v != "" {
		cfg.Host.ApiAddr = v
	}
	if v := os.Getenv("BHIS_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("BHIS_ACCESS_TOKEN_TTL"); v != "" {
		cfg.Auth.AccessTokenTTL = v
	}
	if v := os.Getenv("BHIS_LOG_DIR"); v != "" {
		cfg.Logging.Dir = v
	}
	if v := os.Getenv("BHIS_LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("BHIS_DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("BHIS_DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("BHIS_CORS_ORIGINS"); v != "" {
		cfg.CORS.AllowedOrigins = splitCSV(v)
	}
	if v := os.Getenv("BHIS_AUTH_MAX_FAILURES"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			cfg.Auth.RateLimit.MaxFailures = n
		}
	}
	if v := os.Getenv("BHIS_AUTH_WINDOW"); v != "" {
		cfg.Auth.RateLimit.Window = v
	}
}

// loadLocalEnvFiles fills unset process env from gitignored local files.
// Existing process environment wins. Line text is never included in errors.
func loadLocalEnvFiles() error {
	for _, name := range []string{".env.local", ".env"} {
		if err := loadEnvFile(name); err != nil {
			return err
		}
	}
	return nil
}

func loadEnvFile(name string) error {
	b, err := os.ReadFile(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s", name)
	}
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(k) == "" {
			return fmt.Errorf("%s:%d: invalid line", name, i+1)
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if len(v) >= 2 {
			if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
				v = v[1 : len(v)-1]
			}
		}
		if _, exists := os.LookupEnv(k); exists {
			continue
		}
		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("set %s", k)
		}
	}
	return nil
}

func (c *Config) normalize() error {
	c.Host.ApiAddr = strings.TrimSpace(c.Host.ApiAddr)
	c.Database.Driver = strings.ToLower(strings.TrimSpace(c.Database.Driver))
	c.Logging.Level = strings.ToLower(strings.TrimSpace(c.Logging.Level))
	if c.Host.ApiAddr == "" {
		return fmt.Errorf("Host.ApiAddr is required")
	}
	if strings.TrimSpace(c.Auth.JWTSecret) == "" {
		return fmt.Errorf("Auth.JWTSecret is required (BHIS_JWT_SECRET or config file; do not commit real secrets)")
	}
	if _, err := c.TokenTTL(); err != nil {
		return err
	}
	if _, err := c.RateWindow(); err != nil {
		return err
	}
	switch c.Database.Driver {
	case "memory", "sqlserver":
	default:
		return fmt.Errorf("Database.Driver must be memory or sqlserver")
	}
	c.Database.DSN = strings.TrimSpace(c.Database.DSN)
	if c.Database.Driver == "sqlserver" {
		if c.Database.DSN == "" {
			return fmt.Errorf("BHIS_DB_DSN or Database.DSN is required when Driver is sqlserver")
		}
		if strings.Contains(c.Database.DSN, "<") || strings.Contains(c.Database.DSN, ">") {
			return fmt.Errorf("Database.DSN still contains placeholders; set BHIS_DB_DSN in the environment or a gitignored local file")
		}
	}
	if c.Logging.Dir == "" {
		c.Logging.Dir = "./logs"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	return nil
}

func (c Config) TokenTTL() (time.Duration, error) {
	raw := c.Auth.AccessTokenTTL
	if raw == "" {
		raw = "8h"
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("Auth.AccessTokenTTL: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("Auth.AccessTokenTTL must be positive")
	}
	return d, nil
}

func (c Config) RateWindow() (time.Duration, error) {
	raw := c.Auth.RateLimit.Window
	if raw == "" {
		raw = "15m"
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("Auth.RateLimit.Window: %w", err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("Auth.RateLimit.Window must be positive")
	}
	return d, nil
}

func (c Config) MaxFailures() int {
	if c.Auth.RateLimit.MaxFailures <= 0 {
		return 8
	}
	return c.Auth.RateLimit.MaxFailures
}

// PlaceholderSecret reports the committed example secret so startup can warn.
func (c Config) PlaceholderSecret() bool {
	s := c.Auth.JWTSecret
	return strings.Contains(s, "change-me") || strings.Contains(s, "dev-only") || strings.Contains(s, "replace-me")
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
