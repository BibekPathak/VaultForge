package core

import (
	"encoding/json"
	"time"
	"fmt"
)

// Intent represents a transaction intent
type Intent struct {
	ID           string `json:"id"`
	TenantID     string `json:"tenant_id"`
	WalletID     string `json:"wallet_id"`
	Destination  string `json:"destination"`
	Token        string `json:"token"`
	Amount       string `json:"amount"`
	Chain        string `json:"chain"`
	Nonce        string `json:"nonce"`
	Creator      string `json:"creator"`
	Approvers    []string `json:"approvers"`
	RequiredSigs  int     `json:"required_signatures"`
	PolicyVersion string `json:"policy_version"`
	Expiry       int64  `json:"expiry"`
	Status       string `json:"status"`
	FailureReason *string `json:"failure_reason,omitempty"`
	Timestamps   IntentTimestamps `json:"timestamps"`
	TXSignature  []byte   `json:"transaction_signature,omitempty"`
}

type IntentTimestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	ExecutedAt *time.Time `json:"executed_at,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
}

// PolicyResult policy evaluation result
type PolicyResult struct {
	Allow bool `json:"allow"`
	Reason string `json:"reason,omitempty"`
}

// Wallet struct
type Wallet struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Status    string `json:"status"`
}

// Transaction struct
type Transaction struct {
	ID        string `json:"id"`
	IntentID  string `json:"intent_id"`
	TenantID  string `json:"tenant_id"`
	WalletID  string `json:"wallet_id"`
	Status    string `json:"status"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	Signature string `json:"signature,omitempty"`
}

// AuditEvent struct
type AuditEvent struct {
	ID        string `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	TenantID  string `json:"tenant"`
	Actor     string `json:"actor"`
	Action    string `json:"action"`
	Resource  string `json:"resource"`
	IntentID  string `json:"intent_id"`
	RequestID string `json:"request_id"`
	Result    string `json:"result"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

// SigningContext binds signature to intent/tx
type SigningContext struct {
	IntentID   string
	TxHash     []byte
	Chain      string
	Domain     string
	Timestamp  int64
}

// MPCShare represents a share in threshold signing
type MPCShare struct {
	ShareID  uint32
	ShareKey []byte
}

// SigningResult from MPC
type SigningResult struct {
	Signature []byte
	Participants []uint32
}

// ZKProof struct
type ZKProof struct {
	ProofID string `json:"proof_id"`
	PublicInputs json.RawMessage `json:"public_inputs"`
	ProofBytes []byte `json:"-"`
}

// ReplayKey for replay protection
type ReplayKey struct {
	IntentID   string
	Chain      string
	Version    uint64
	Used       bool
}

// NewIntent creates a new intent
func NewIntent(tenantID, walletID, destination, token, chain, creator string, amount int64) *Intent {
	now := time.Now().UTC()
	return &Intent{
		ID: generateUUID(),
		TenantID: tenantID,
		WalletID: walletID,
		Destination: destination,
		Token: token,
		Amount: fmt.Sprintf("%d", amount),
		Chain: chain,
		Nonce: generateNonce(),
		Creator: creator,
		Status: "draft",
		Timestamps: IntentTimestamps{
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func generateUUID() string {
	// Placeholder - use proper UUID generation
	return "uuid-" + fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateNonce() string {
	return "nonce-" + fmt.Sprintf("%d", time.Now().UnixNano())
}
