package core

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

var base58Chars = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]+$`)

// ValidateSolanaAddress checks if the given string is a valid Solana base58 address (32-44 chars).
func ValidateSolanaAddress(addr string) bool {
	if len(addr) < 32 || len(addr) > 44 {
		return false
	}
	return base58Chars.MatchString(addr)
}

// ValidateTokenMint checks if the given string looks like a valid Solana token mint address.
func ValidateTokenMint(mint string) bool {
	return ValidateSolanaAddress(mint)
}

// ValidateStringLen returns an error if the string exceeds the max length.
func ValidateStringLen(field, value string, maxLen int) error {
	if len(value) > maxLen {
		return fmt.Errorf("%s must be at most %d characters", field, maxLen)
	}
	return nil
}

// ValidateAmountString checks if the string is a valid non-negative integer amount.
func ValidateAmountString(amount string) (int64, error) {
	if amount == "" {
		return 0, fmt.Errorf("amount is required")
	}
	var v int64
	for _, c := range amount {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("amount must be a numeric string")
		}
		v = v*10 + int64(c-'0')
		if v < 0 {
			return 0, fmt.Errorf("amount overflow")
		}
	}
	return v, nil
}

// ValidateChain checks if the chain is a supported value.
func ValidateChain(chain string) bool {
	supported := map[string]bool{"solana": true, "ethereum": true, "polygon": true}
	return supported[strings.ToLower(chain)]
}

// ValidateIntentInput performs comprehensive validation on intent creation inputs.
func ValidateIntentInput(walletID, destination, token, chain, creator string, amount int64) []string {
	var errs []string

	if walletID == "" {
		errs = append(errs, "wallet_id is required")
	}
	if err := ValidateStringLen("destination", destination, 64); err != nil {
		errs = append(errs, err.Error())
	}
	if err := ValidateStringLen("token", token, 64); err != nil {
		errs = append(errs, err.Error())
	}
	if err := ValidateStringLen("creator", creator, 128); err != nil {
		errs = append(errs, err.Error())
	}
	if !ValidateChain(chain) {
		errs = append(errs, fmt.Sprintf("chain must be one of: solana, ethereum, polygon"))
	}
	if amount <= 0 {
		errs = append(errs, "amount must be greater than zero")
	}
	if amount > math.MaxInt64/2 {
		errs = append(errs, "amount is too large")
	}

	return errs
}
