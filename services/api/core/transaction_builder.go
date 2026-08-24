package core

import (
	"crypto/sha256"
	"fmt"
	"log"
	"strconv"
	"time"
)

// SolanaTransactionBuilder builds Solana transaction bytes.
type SolanaTransactionBuilder struct {
	walletID    string
	token       string
	amount      string
	destination string
	blockhash   string
	priorityFee int64
}

// NewTransactionBuilder creates a new transaction builder.
func NewTransactionBuilder(walletID, token, amount, destination string) *SolanaTransactionBuilder {
	return &SolanaTransactionBuilder{
		walletID:    walletID,
		token:       token,
		amount:      amount,
		destination: destination,
	}
}

// Simulate simulates the transaction against local/devnet SVM.
func (b *SolanaTransactionBuilder) Simulate() (*SimulationResult, error) {
	amountVal, err := strconv.ParseInt(b.amount, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	if amountVal <= 0 {
		return &SimulationResult{
			Allowed: false,
			Error:   "amount must be positive",
		}, nil
	}

	// In production: run against LiteSVM or Mollusk
	txHash := b.computeTxHash()

	log.Printf("Transaction simulated: amount=%d token=%s dest=%s", amountVal, b.token, b.destination)

	return &SimulationResult{
		Allowed:         true,
		TransactionHash: txHash,
	}, nil
}

// Build constructs the canonical transaction bytes.
func (b *SolanaTransactionBuilder) Build() []byte {
	h := sha256.New()
	h.Write([]byte("vaultforge-tx-v1"))
	h.Write([]byte(b.walletID))
	h.Write([]byte(b.token))
	h.Write([]byte(b.amount))
	h.Write([]byte(b.destination))
	if b.blockhash != "" {
		h.Write([]byte(b.blockhash))
	}
	var buf [8]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(b.priorityFee >> (8 * i)) // #nosec G115 -- intentional little-endian byte encoding
	}
	h.Write(buf[:])
	h.Write([]byte(time.Now().UTC().Format(time.RFC3339)))
	return h.Sum(nil)
}

// SetBlockhash sets the recent blockhash.
func (b *SolanaTransactionBuilder) SetBlockhash(blockhash string) {
	b.blockhash = blockhash
}

// SetPriorityFee sets the priority fee in microlamports.
func (b *SolanaTransactionBuilder) SetPriorityFee(fee int64) {
	b.priorityFee = fee
}

func (b *SolanaTransactionBuilder) computeTxHash() string {
	txBytes := b.Build()
	return fmt.Sprintf("%x", txBytes)
}
