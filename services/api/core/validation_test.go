package core

import (
	"testing"
)

func TestValidateSolanaAddress_Valid(t *testing.T) {
	valid := []string{
		"11111111111111111111111111111111",
		"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
		"So11111111111111111111111111111111111111112",
		"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
	}
	for _, addr := range valid {
		if !ValidateSolanaAddress(addr) {
			t.Errorf("expected %s to be valid", addr)
		}
	}
}

func TestValidateSolanaAddress_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"short",
		"0OIl", // invalid base58 chars
		"Address with spaces",
		"abc123!@#$%",
	}
	for _, addr := range invalid {
		if ValidateSolanaAddress(addr) {
			t.Errorf("expected %s to be invalid", addr)
		}
	}
}

func TestValidateStringLen(t *testing.T) {
	if err := ValidateStringLen("field", "short", 100); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if err := ValidateStringLen("field", "this is a very long string that exceeds the limit", 10); err == nil {
		t.Error("expected error for long string")
	}
}

func TestValidateAmountString_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"0", 0},
		{"100", 100},
		{"999999999999", 999999999999},
	}
	for _, tt := range tests {
		got, err := ValidateAmountString(tt.input)
		if err != nil {
			t.Errorf("ValidateAmountString(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ValidateAmountString(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestValidateAmountString_Invalid(t *testing.T) {
	invalid := []string{"", "abc", "12.34", "-1", "1a2b"}
	for _, s := range invalid {
		_, err := ValidateAmountString(s)
		if err == nil {
			t.Errorf("expected error for %q", s)
		}
	}
}

func TestValidateChain(t *testing.T) {
	if !ValidateChain("solana") {
		t.Error("solana should be valid")
	}
	if !ValidateChain("Solana") {
		t.Error("Solana should be valid (case insensitive)")
	}
	if ValidateChain("bitcoin") {
		t.Error("bitcoin should be invalid")
	}
}

func TestValidateIntentInput_AllValid(t *testing.T) {
	errs := ValidateIntentInput("w1", "dest-addr", "SOL", "solana", "user-1", 1000)
	if len(errs) > 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateIntentInput_MultipleErrors(t *testing.T) {
	errs := ValidateIntentInput("", "", "", "", "", 0)
	if len(errs) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateIntentInput_NegativeAmount(t *testing.T) {
	errs := ValidateIntentInput("w1", "d", "SOL", "solana", "u", -100)
	found := false
	for _, e := range errs {
		if e == "amount must be greater than zero" {
			found = true
		}
	}
	if !found {
		t.Error("expected amount error")
	}
}
