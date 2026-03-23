package configs

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Store    StoreConfig    `yaml:"store"`
	Bloom    BloomConfig    `yaml:"bloom"`
	WAL      WALConfig      `yaml:"wal"`
	LogLevel string         `yaml:"log_level"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port string `yaml:"port"`
}

// StoreConfig holds store settings.
type StoreConfig struct {
	MaxKeys      int           `yaml:"max_keys"`
	TTLInterval  time.Duration `yaml:"ttl_interval"`
}

// BloomConfig holds Bloom filter settings.
type BloomConfig struct {
	ExpectedKeys       int     `yaml:"expected_keys"`
	FalsePositiveRate  float64 `yaml:"false_positive_rate"`
}

// WALConfig holds Write-Ahead Log settings.
type WALConfig struct {
	Path string `yaml:"path"`
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port: "9801",
		},
		Store: StoreConfig{
			MaxKeys:     10000,
			TTLInterval: 1 * time.Second,
		},
		Bloom: BloomConfig{
			ExpectedKeys:      10000,
			FalsePositiveRate: 0.01,
		},
		WAL: WALConfig{
			Path: "./data/wal.log",
		},
		LogLevel: "info",
	}
}

// Load loads config from file and env. File values are overridden by env.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, err
			}
		}
	}

	// Env overrides
	if v := os.Getenv("PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("MAX_KEYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Store.MaxKeys = n
		}
	}
	if v := os.Getenv("TTL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Store.TTLInterval = d
		}
	}
	if v := os.Getenv("WAL_PATH"); v != "" {
		cfg.WAL.Path = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	return cfg, nil
}
