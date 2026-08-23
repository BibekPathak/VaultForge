package core

import (
	"os"
	"testing"
	"time"
)

func TestConfig_Validate_Valid(t *testing.T) {
	cfg := &Config{
		Port:          "8080",
		DatabaseURL:   "host=localhost",
		SolanaRPCURL:  "https://api.devnet.solana.com",
		Environment:   "development",
		ReadTimeout:   15 * time.Second,
		WriteTimeout:  30 * time.Second,
		IdleTimeout:   60 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestConfig_Validate_MissingDatabaseURL(t *testing.T) {
	cfg := &Config{Port: "8080", SolanaRPCURL: "https://rpc.solana.com", Environment: "development"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing DATABASE_URL")
	}
}

func TestConfig_Validate_MissingPort(t *testing.T) {
	cfg := &Config{DatabaseURL: "host=localhost", SolanaRPCURL: "https://rpc.solana.com", Environment: "development"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing PORT")
	}
}

func TestConfig_Validate_MissingSolanaRPC(t *testing.T) {
	cfg := &Config{Port: "8080", DatabaseURL: "host=localhost", Environment: "development"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for missing SOLANA_RPC_URL")
	}
}

func TestConfig_Validate_InvalidEnvironment(t *testing.T) {
	cfg := &Config{Port: "8080", DatabaseURL: "host=localhost", SolanaRPCURL: "https://rpc.solana.com", Environment: "invalid"}
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid environment")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	// Clear any existing env vars
	keys := []string{"PORT", "DATABASE_URL", "SOLANA_RPC_URL", "VAULTFORGE_ENV", "LOG_LEVEL"}
	for _, k := range keys {
		os.Unsetenv(k)
	}

	cfg := LoadConfig()
	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.Environment != "development" {
		t.Errorf("expected default env development, got %s", cfg.Environment)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected 15s read timeout, got %v", cfg.ReadTimeout)
	}
}

func TestLoadConfig_EnvOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	defer os.Unsetenv("PORT")

	os.Setenv("VAULTFORGE_ENV", "production")
	defer os.Unsetenv("VAULTFORGE_ENV")

	cfg := LoadConfig()
	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.Environment != "production" {
		t.Errorf("expected env production, got %s", cfg.Environment)
	}
}

func TestLoadConfig_DurationParsing(t *testing.T) {
	os.Setenv("READ_TIMEOUT", "30s")
	defer os.Unsetenv("READ_TIMEOUT")

	cfg := LoadConfig()
	if cfg.ReadTimeout != 30*time.Second {
		t.Errorf("expected 30s read timeout, got %v", cfg.ReadTimeout)
	}
}

func TestLoadConfig_InvalidDurationFallback(t *testing.T) {
	os.Setenv("READ_TIMEOUT", "invalid")
	defer os.Unsetenv("READ_TIMEOUT")

	cfg := LoadConfig()
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected fallback 15s read timeout, got %v", cfg.ReadTimeout)
	}
}
