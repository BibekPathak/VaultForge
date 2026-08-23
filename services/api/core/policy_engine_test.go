package core

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPolicyEngine_AllowWhenNoPolicies(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}

	pe := NewPolicyEngine(db)
	result, err := pe.Evaluate(&Intent{
		WalletID: "w1",
		Amount:   "50000",
		Token:    "SOL",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allow {
		t.Error("should allow when no policies exist")
	}
}

func TestPolicyEngine_DailyLimit_Allow(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "daily-limit", RuleType: "daily_limit", IsActive: true, Config: json.RawMessage(`{"limit":100000}`)},
	}

	pe := NewPolicyEngine(db)
	result, err := pe.Evaluate(&Intent{WalletID: "w1", Amount: "50000", Token: "SOL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allow {
		t.Errorf("should allow under daily limit, got reason: %s", result.Reason)
	}
}

func TestPolicyEngine_DailyLimit_Deny(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "daily-limit", RuleType: "daily_limit", IsActive: true, Config: json.RawMessage(`{"limit":100000}`)},
	}

	pe := NewPolicyEngine(db)
	result, err := pe.Evaluate(&Intent{WalletID: "w1", Amount: "200000", Token: "SOL"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allow {
		t.Error("should deny over daily limit")
	}
}

func TestPolicyEngine_SingleTxLimit(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "tx-limit", RuleType: "single_tx_limit", IsActive: true, Config: json.RawMessage(`{"limit":10000}`)},
	}

	pe := NewPolicyEngine(db)
	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "50000", Token: "SOL"})
	if result.Allow {
		t.Error("should deny over single tx limit")
	}
}

func TestPolicyEngine_AllowedRecipients(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "allowed-dest", RuleType: "allowed_recipients", IsActive: true, Config: json.RawMessage(`["addr-A","addr-B"]`)},
	}

	pe := NewPolicyEngine(db)

	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Destination: "addr-A", Token: "SOL"})
	if !result.Allow {
		t.Error("should allow known recipient")
	}

	result, _ = pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Destination: "addr-C", Token: "SOL"})
	if result.Allow {
		t.Error("should deny unknown recipient")
	}
}

func TestPolicyEngine_BlockedRecipients(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "blocked-dest", RuleType: "blocked_recipients", IsActive: true, Config: json.RawMessage(`["bad-addr"]`)},
	}

	pe := NewPolicyEngine(db)

	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Destination: "good-addr", Token: "SOL"})
	if !result.Allow {
		t.Error("should allow non-blocked recipient")
	}

	result, _ = pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Destination: "bad-addr", Token: "SOL"})
	if result.Allow {
		t.Error("should deny blocked recipient")
	}
}

func TestPolicyEngine_AllowedTokens(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "allowed-tokens", RuleType: "allowed_tokens", IsActive: true, Config: json.RawMessage(`["SOL","USDC"]`)},
	}

	pe := NewPolicyEngine(db)

	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Token: "SOL"})
	if !result.Allow {
		t.Error("should allow known token")
	}

	result, _ = pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Token: "DOGE"})
	if result.Allow {
		t.Error("should deny unknown token")
	}
}

func TestPolicyEngine_RequiredSigs(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "req-sigs", RuleType: "required_signatures", IsActive: true, Config: json.RawMessage(`{"required":3}`)},
	}

	pe := NewPolicyEngine(db)

	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Token: "SOL", Approvers: []string{"a", "b"}})
	if result.Allow {
		t.Error("should deny with insufficient approvers")
	}

	result, _ = pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Token: "SOL", Approvers: []string{"a", "b", "c"}})
	if !result.Allow {
		t.Error("should allow with sufficient approvers")
	}
}

func TestPolicyEngine_TimeRestriction(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}

	futureEnd := time.Now().UTC().Add(1 * time.Hour).Format(time.RFC3339)
	pastEnd := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)

	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "time-restrict", RuleType: "time_restriction", IsActive: true, Config: json.RawMessage(`{"start":"","end":"` + pastEnd + `"}`)},
	}

	pe := NewPolicyEngine(db)
	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Token: "SOL"})
	if result.Allow {
		t.Error("should deny outside time window")
	}

	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "time-restrict", RuleType: "time_restriction", IsActive: true, Config: json.RawMessage(`{"start":"","end":"` + futureEnd + `"}`)},
	}

	result, _ = pe.Evaluate(&Intent{WalletID: "w1", Amount: "100", Token: "SOL"})
	if !result.Allow {
		t.Error("should allow inside time window")
	}
}

func TestPolicyEngine_InactivePolicyIgnored(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "daily-limit", RuleType: "daily_limit", IsActive: false, Config: json.RawMessage(`{"limit":1}`)},
	}

	pe := NewPolicyEngine(db)
	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "999999", Token: "SOL"})
	if !result.Allow {
		t.Error("inactive policy should be ignored")
	}
}

func TestPolicyEngine_MultiplePolicies_AllPass(t *testing.T) {
	db := newMockDBQuerier()
	db.wallets["w1"] = &Wallet{ID: "w1", TenantID: "t1", DailyLimit: 100000}
	db.tenants["t1"] = &Tenant{ID: "t1"}
	db.policies["t1"] = []*Policy{
		{ID: "p1", TenantID: "t1", Name: "daily", RuleType: "daily_limit", IsActive: true, Config: json.RawMessage(`{"limit":100000}`)},
		{ID: "p2", TenantID: "t1", Name: "tokens", RuleType: "allowed_tokens", IsActive: true, Config: json.RawMessage(`["SOL"]`)},
	}

	pe := NewPolicyEngine(db)
	result, _ := pe.Evaluate(&Intent{WalletID: "w1", Amount: "50000", Token: "SOL"})
	if !result.Allow {
		t.Errorf("should allow when all policies pass, got: %s", result.Reason)
	}
}

func TestParseIntentAmount(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"1000", 1000},
		{"0", 0},
		{"999999999999", 999999999999},
		{"abc", 0},
		{"", 0},
		{"12ab34", 1234},
	}
	for _, tt := range tests {
		got := parseIntentAmount(tt.input)
		if got != tt.want {
			t.Errorf("parseIntentAmount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
