package core

import (
	"encoding/json"
	"testing"
)

func TestZKVerifier_GenerateProof_Valid(t *testing.T) {
	zk := NewZKVerifier()
	dailyLimit := int64(100000)
	proof, err := zk.GenerateProof(PolicyInputs{
		Amount:        "50000",
		DailyLimit:    &dailyLimit,
		Recipient:     "dest-addr",
		PolicyVersion: "v1",
		IntentID:      "intent-1",
		WalletID:      "wallet-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proof == nil {
		t.Fatal("proof should not be nil")
	}
	if proof.ProofID == "" {
		t.Error("proof ID should not be empty")
	}
	if len(proof.ProofBytes) == 0 {
		t.Error("proof bytes should not be empty")
	}
	if len(proof.PublicInputs) == 0 {
		t.Error("public inputs should not be empty")
	}
}

func TestZKVerifier_GenerateProof_DefaultDailyLimit(t *testing.T) {
	zk := NewZKVerifier()
	proof, err := zk.GenerateProof(PolicyInputs{
		Amount:        "50000",
		Recipient:     "dest",
		PolicyVersion: "v1",
		IntentID:      "i1",
		WalletID:      "w1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proof == nil {
		t.Fatal("proof should not be nil")
	}
}

func TestZKVerifier_GenerateProof_InvalidAmount(t *testing.T) {
	zk := NewZKVerifier()
	_, err := zk.GenerateProof(PolicyInputs{
		Amount:        "abc",
		PolicyVersion: "v1",
		IntentID:      "i1",
		WalletID:      "w1",
	})
	if err == nil {
		t.Error("expected error for invalid amount")
	}
}

func TestZKVerifier_GenerateProof_ZeroAmount(t *testing.T) {
	zk := NewZKVerifier()
	_, err := zk.GenerateProof(PolicyInputs{
		Amount:        "0",
		PolicyVersion: "v1",
		IntentID:      "i1",
		WalletID:      "w1",
	})
	if err == nil {
		t.Error("expected error for zero amount")
	}
}

func TestZKVerifier_GenerateProof_ExceedsDailyLimit(t *testing.T) {
	zk := NewZKVerifier()
	dailyLimit := int64(100)
	_, err := zk.GenerateProof(PolicyInputs{
		Amount:        "200",
		DailyLimit:    &dailyLimit,
		PolicyVersion: "v1",
		IntentID:      "i1",
		WalletID:      "w1",
	})
	if err == nil {
		t.Error("expected error when amount exceeds daily limit")
	}
}

func TestZKVerifier_VerifyPolicyProof_Valid(t *testing.T) {
	zk := NewZKVerifier()
	dailyLimit := int64(100000)
	proof, err := zk.GenerateProof(PolicyInputs{
		Amount:        "50000",
		DailyLimit:    &dailyLimit,
		Recipient:     "dest",
		PolicyVersion: "v1",
		IntentID:      "intent-1",
		WalletID:      "wallet-1",
	})
	if err != nil {
		t.Fatalf("unexpected error generating proof: %v", err)
	}

	valid, err := zk.VerifyPolicyProof(proof)
	if err != nil {
		t.Fatalf("unexpected error verifying proof: %v", err)
	}
	if !valid {
		t.Error("proof should be valid")
	}
}

func TestZKVerifier_VerifyPolicyProof_NilProof(t *testing.T) {
	zk := NewZKVerifier()
	_, err := zk.VerifyPolicyProof(nil)
	if err == nil {
		t.Error("expected error for nil proof")
	}
}

func TestZKVerifier_VerifyPolicyProof_EmptyBytes(t *testing.T) {
	zk := NewZKVerifier()
	_, err := zk.VerifyPolicyProof(&ZKProof{ProofID: "x", ProofBytes: []byte{}})
	if err == nil {
		t.Error("expected error for empty proof bytes")
	}
}

func TestZKVerifier_VerifyPolicyProof_Tampered(t *testing.T) {
	zk := NewZKVerifier()
	dailyLimit := int64(100000)
	proof, err := zk.GenerateProof(PolicyInputs{
		Amount:        "50000",
		DailyLimit:    &dailyLimit,
		PolicyVersion: "v1",
		IntentID:      "i1",
		WalletID:      "w1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tamper with the proof
	var internal ZKProofInternal
	if err := json.Unmarshal(proof.ProofBytes, &internal); err != nil {
		t.Fatalf("failed to unmarshal proof: %v", err)
	}
	internal.PublicInputs.Amount = 999999
	tamperedBytes, _ := json.Marshal(internal)
	proof.ProofBytes = tamperedBytes

	valid, err := zk.VerifyPolicyProof(proof)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("tampered proof should be invalid")
	}
}
