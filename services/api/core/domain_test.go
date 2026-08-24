package core

import (
	"crypto/sha256"
	"encoding/json"
	"testing"
)

func TestComputeIntentHash_Deterministic(t *testing.T) {
	intent := &Intent{
		ID:          "intent-1",
		TenantID:    "tenant-1",
		WalletID:    "wallet-1",
		Destination: "dest-addr",
		Token:       "So11111111111111111111111111111111111111112",
		Amount:      "1000000",
		Chain:       "solana",
		Nonce:       "nonce-1",
		Creator:     "user-1",
		Status:      "pending",
	}

	hash1 := ComputeIntentHash(intent)
	hash2 := ComputeIntentHash(intent)

	if len(hash1) != sha256.Size {
		t.Fatalf("expected hash length %d, got %d", sha256.Size, len(hash1))
	}
	for i := range hash1 {
		if hash1[i] != hash2[i] {
			t.Fatal("same inputs should produce same hash")
		}
	}
}

func TestComputeIntentHash_DifferentInputs(t *testing.T) {
	intent1 := &Intent{ID: "a", TenantID: "t", WalletID: "w", Destination: "d1", Token: "tok", Amount: "100", Chain: "solana", Nonce: "n", Creator: "c", Status: "pending"}
	intent2 := &Intent{ID: "a", TenantID: "t", WalletID: "w", Destination: "d2", Token: "tok", Amount: "100", Chain: "solana", Nonce: "n", Creator: "c", Status: "pending"}

	h1 := ComputeIntentHash(intent1)
	h2 := ComputeIntentHash(intent2)

	same := true
	for i := range h1 {
		if h1[i] != h2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different inputs should produce different hashes")
	}
}

func TestComputeIntentHash_StatusMatters(t *testing.T) {
	intent1 := &Intent{ID: "a", TenantID: "t", WalletID: "w", Destination: "d", Token: "tok", Amount: "100", Chain: "solana", Nonce: "n", Creator: "c", Status: "draft"}
	intent2 := &Intent{ID: "a", TenantID: "t", WalletID: "w", Destination: "d", Token: "tok", Amount: "100", Chain: "solana", Nonce: "n", Creator: "c", Status: "approved"}

	h1 := ComputeIntentHash(intent1)
	h2 := ComputeIntentHash(intent2)

	same := true
	for i := range h1 {
		if h1[i] != h2[i] {
			same = false
			break
		}
	}
	if same {
		t.Fatal("different status should produce different hashes")
	}
}

func TestNewIntent_Defaults(t *testing.T) {
	intent := NewIntent("tenant-1", "wallet-1", "dest", "SOL", "solana", "user-1", 1000)

	if intent.ID == "" {
		t.Error("ID should not be empty")
	}
	if intent.TenantID != "tenant-1" {
		t.Errorf("expected TenantID=tenant-1, got %s", intent.TenantID)
	}
	if intent.Status != "pending" {
		t.Errorf("expected status=pending, got %s", intent.Status)
	}
	if intent.Amount != "1000" {
		t.Errorf("expected amount=1000, got %s", intent.Amount)
	}
	if intent.Expiry.IsZero() {
		t.Error("Expiry should be set")
	}
	if intent.Nonce == "" {
		t.Error("Nonce should not be empty")
	}
}

func TestNewIntent_UniqueNonces(t *testing.T) {
	i1 := NewIntent("t", "w", "d", "SOL", "solana", "u", 100)
	i2 := NewIntent("t", "w", "d", "SOL", "solana", "u", 100)
	if i1.Nonce == i2.Nonce {
		t.Error("each intent should have a unique nonce")
	}
}

func TestGenerateUUID_NotEmpty(t *testing.T) {
	uuid := GenerateUUID()
	if uuid == "" {
		t.Error("UUID should not be empty")
	}
	if len(uuid) < 5 {
		t.Error("UUID should have reasonable length")
	}
}

func TestGenerateRequestID_NotEmpty(t *testing.T) {
	rid := GenerateRequestID()
	if rid == "" {
		t.Error("RequestID should not be empty")
	}
}

func TestTableName(t *testing.T) {
	tests := []struct {
		entity interface{ TableName() string }
		want   string
	}{
		{&Intent{}, "intents"},
		{&Tenant{}, "tenants"},
		{&Wallet{}, "wallets"},
		{&Transaction{}, "transactions"},
		{&AuditEvent{}, "audit_events"},
		{&Policy{}, "policies"},
		{&MPCShareRecord{}, "mpc_shares"},
	}
	for _, tt := range tests {
		if got := tt.entity.TableName(); got != tt.want {
			t.Errorf("TableName() = %q, want %q", got, tt.want)
		}
	}
}

func TestNewIntent_JSONMarshalable(t *testing.T) {
	intent := NewIntent("t", "w", "d", "SOL", "solana", "u", 100)
	data, err := json.Marshal(intent)
	if err != nil {
		t.Fatalf("failed to marshal intent: %v", err)
	}
	var decoded Intent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal intent: %v", err)
	}
	if decoded.ID != intent.ID {
		t.Errorf("decoded ID mismatch: %s vs %s", decoded.ID, intent.ID)
	}
}
