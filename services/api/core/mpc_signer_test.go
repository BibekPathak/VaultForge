package core

import (
	"testing"
)

func TestComputeSigningMessage_Deterministic(t *testing.T) {
	intentHash := []byte("intent-hash-123")
	txHash := []byte("tx-hash-456")
	chain := "solana"
	context := "vaultforge-intent-1"

	msg1 := computeSigningMessage(intentHash, txHash, chain, context)
	msg2 := computeSigningMessage(intentHash, txHash, chain, context)

	if len(msg1) != 32 {
		t.Fatalf("expected 32-byte message, got %d", len(msg1))
	}
	for i := range msg1 {
		if msg1[i] != msg2[i] {
			t.Fatal("same inputs should produce same signing message")
		}
	}
}

func TestComputeSigningMessage_DiffChain_DiffOutput(t *testing.T) {
	intentHash := []byte("intent-hash")
	txHash := []byte("tx-hash")

	msg1 := computeSigningMessage(intentHash, txHash, "solana", "ctx")
	msg2 := computeSigningMessage(intentHash, txHash, "ethereum", "ctx")

	same := true
	for i := range msg1 {
		if msg1[i] != msg2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different chains should produce different messages")
	}
}

func TestComputePartialSignature_Deterministic(t *testing.T) {
	share := []byte("share-data")
	message := []byte("message-data")

	sig1 := computePartialSignature(share, message)
	sig2 := computePartialSignature(share, message)

	if len(sig1) != 32 {
		t.Fatalf("expected 32-byte signature, got %d", len(sig1))
	}
	for i := range sig1 {
		if sig1[i] != sig2[i] {
			t.Fatal("same inputs should produce same partial signature")
		}
	}
}

func TestComputePartialSignature_DiffShare_DiffOutput(t *testing.T) {
	message := []byte("message")

	sig1 := computePartialSignature([]byte("share-1"), message)
	sig2 := computePartialSignature([]byte("share-2"), message)

	same := true
	for i := range sig1 {
		if sig1[i] != sig2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different shares should produce different signatures")
	}
}

func TestAggregatePartialSignatures_64Bytes(t *testing.T) {
	partials := [][]byte{
		computePartialSignature([]byte("s1"), []byte("m")),
		computePartialSignature([]byte("s2"), []byte("m")),
	}

	sig := aggregatePartialSignatures(partials)
	if len(sig) != 64 {
		t.Errorf("expected 64-byte signature, got %d", len(sig))
	}
}

func TestAggregatePartialSignatures_DiffInputs_DiffOutput(t *testing.T) {
	sig1 := aggregatePartialSignatures([][]byte{[]byte("a"), []byte("b")})
	sig2 := aggregatePartialSignatures([][]byte{[]byte("c"), []byte("d")})

	same := true
	for i := range sig1 {
		if sig1[i] != sig2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("different partials should produce different aggregate")
	}
}

func TestMPCSigner_MissingIntentHash(t *testing.T) {
	s := NewMPCSigner(nil)
	_, err := s.Sign(SigningInput{
		IntentHash: nil,
		TxHash:     "tx-hash",
		Chain:      "solana",
		Context:    "ctx",
	})
	if err == nil {
		t.Error("expected error for missing intent hash")
	}
}

func TestMPCSigner_MissingTxHash(t *testing.T) {
	s := NewMPCSigner(nil)
	_, err := s.Sign(SigningInput{
		IntentHash: []byte("hash"),
		TxHash:     "",
		Chain:      "solana",
		Context:    "ctx",
	})
	if err == nil {
		t.Error("expected error for missing tx hash")
	}
}
