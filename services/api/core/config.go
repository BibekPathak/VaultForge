package core

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Port          string
	DatabaseURL   string
	SolanaRPCURL  string
	Environment   string
	LogLevel      string
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	IdleTimeout   time.Duration
	MaxBodyBytes  int64
	ShutdownTimeout time.Duration
}

// LoadConfig reads configuration from environment variables with defaults.
func LoadConfig() *Config {
	cfg := &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable"),
		SolanaRPCURL:    getEnv("SOLANA_RPC_URL", "https://api.devnet.solana.com"),
		Environment:     getEnv("VAULTFORGE_ENV", "development"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ReadTimeout:     getDurationEnv("READ_TIMEOUT", 15*time.Second),
		WriteTimeout:    getDurationEnv("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:     getDurationEnv("IDLE_TIMEOUT", 60*time.Second),
		MaxBodyBytes:    getInt64Env("MAX_BODY_BYTES", 10<<20),
		ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	return cfg
}

// Validate checks that required configuration values are present.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.Port == "" {
		return fmt.Errorf("PORT is required")
	}
	if c.SolanaRPCURL == "" {
		return fmt.Errorf("SOLANA_RPC_URL is required")
	}
	validEnvs := map[string]bool{"development": true, "staging": true, "production": true}
	if !validEnvs[c.Environment] {
		return fmt.Errorf("VAULTFORGE_ENV must be one of: development, staging, production")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getInt64Env(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}
