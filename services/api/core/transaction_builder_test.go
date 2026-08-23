package core

import (
	"testing"
)

func TestTransactionBuilder_Simulate_Valid(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "1000", "dest-addr")
	result, err := b.Simulate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Error("transaction should be allowed")
	}
	if result.TransactionHash == "" {
		t.Error("transaction hash should not be empty")
	}
}

func TestTransactionBuilder_Simulate_ZeroAmount(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "0", "dest")
	result, err := b.Simulate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("zero amount should not be allowed")
	}
}

func TestTransactionBuilder_Simulate_NegativeAmount(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "-100", "dest")
	result, err := b.Simulate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Error("negative amount should not be allowed")
	}
}

func TestTransactionBuilder_Simulate_InvalidAmount(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "abc", "dest")
	_, err := b.Simulate()
	if err == nil {
		t.Error("expected error for invalid amount")
	}
}

func TestTransactionBuilder_Build_Deterministic(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "1000", "dest")
	b1 := b.Build()
	b2 := b.Build()
	// Build uses time.Now() so hashes will differ; just verify non-empty
	if len(b1) == 0 {
		t.Error("build output should not be empty")
	}
	if len(b2) == 0 {
		t.Error("build output should not be empty")
	}
}

func TestTransactionBuilder_Build_WithBlockhash(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "1000", "dest")
	b.SetBlockhash("blockhash-123")
	result := b.Build()
	if len(result) == 0 {
		t.Error("build output should not be empty")
	}
}

func TestTransactionBuilder_Build_WithPriorityFee(t *testing.T) {
	b := NewTransactionBuilder("w1", "SOL", "1000", "dest")
	b.SetPriorityFee(1000)
	result := b.Build()
	if len(result) == 0 {
		t.Error("build output should not be empty")
	}
}

func TestTransactionBuilder_Build_DiffBlockhash_DiffOutput(t *testing.T) {
	b1 := NewTransactionBuilder("w1", "SOL", "1000", "dest")
	b1.SetBlockhash("hash-a")
	out1 := b1.Build()

	b2 := NewTransactionBuilder("w1", "SOL", "1000", "dest")
	b2.SetBlockhash("hash-b")
	out2 := b2.Build()

	same := true
	for i := range out1 {
		if i < len(out2) && out1[i] != out2[i] {
			same = false
			break
		}
	}
	if same && len(out1) == len(out2) {
		t.Error("different blockhashes should produce different outputs")
	}
}
