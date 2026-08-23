package core

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Intent represents a transaction intent through its lifecycle.
type Intent struct {
	ID              string           `json:"id" gorm:"primaryKey"`
	TenantID        string           `json:"tenant_id" gorm:"index;not null"`
	WalletID        string           `json:"wallet_id" gorm:"index;not null"`
	Destination     string           `json:"destination" gorm:"not null"`
	Token           string           `json:"token" gorm:"column:token_mint;not null"`
	Amount          string           `json:"amount" gorm:"type:numeric(78);not null"`
	Chain           string           `json:"chain" gorm:"default:solana"`
	Nonce           string           `json:"nonce" gorm:"uniqueIndex;not null"`
	Creator         string           `json:"creator" gorm:"not null"`
	Approvers       []string         `json:"approvers" gorm:"type:text[]"`
	RequiredSigs    int              `json:"required_signatures" gorm:"default:2"`
	PolicyVersion   string           `json:"policy_version" gorm:"default:v1"`
	Expiry          time.Time        `json:"expiry" gorm:"not null"`
	Status          string           `json:"status" gorm:"default:draft;index"`
	FailureReason   *string          `json:"failure_reason,omitempty"`
	TXSignature     []byte           `json:"transaction_signature,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	ApprovedAt      *time.Time       `json:"approved_at,omitempty"`
	ExecutedAt      *time.Time       `json:"executed_at,omitempty"`
	ConfirmedAt     *time.Time       `json:"confirmed_at,omitempty"`
}

func (Intent) TableName() string { return "intents" }

// IntentTimestamps is embedded in Intent via the timestamp fields above.

// PolicyResult is the output of policy evaluation.
type PolicyResult struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason,omitempty"`
}

// Tenant represents an institution or user group.
type Tenant struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"not null"`
	Domain    string    `json:"domain" gorm:"uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Tenant) TableName() string { return "tenants" }

// Wallet is a Solana wallet bound to a tenant.
type Wallet struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	TenantID   string    `json:"tenant_id" gorm:"index;not null"`
	Name       string    `json:"name" gorm:"not null"`
	DailyLimit int64     `json:"daily_limit"`
	Status     string    `json:"status" gorm:"default:active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Wallet) TableName() string { return "wallets" }

// Transaction tracks an on-chain transaction.
type Transaction struct {
	ID                string     `json:"id" gorm:"primaryKey"`
	IntentID          string     `json:"intent_id" gorm:"index;not null"`
	TenantID          string     `json:"tenant_id" gorm:"index;not null"`
	WalletID          string     `json:"wallet_id" gorm:"index;not null"`
	TransactionBytes  []byte     `json:"transaction_bytes"`
	Blockhash         string     `json:"blockhash"`
	PriorityFee       int64      `json:"priority_fee"`
	Status            string     `json:"status" gorm:"default:pending;index"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
	ConfirmedAt       *time.Time `json:"confirmed_at,omitempty"`
	ConfirmSignature  string     `json:"confirmation_signature,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (Transaction) TableName() string { return "transactions" }

// AuditEvent records every state transition.
type AuditEvent struct {
	ID        string          `json:"id" gorm:"primaryKey"`
	TenantID  string          `json:"tenant_id" gorm:"index"`
	Actor     string          `json:"actor" gorm:"not null"`
	Action    string          `json:"action" gorm:"not null;index"`
	Resource  string          `json:"resource" gorm:"not null"`
	IntentID  string          `json:"intent_id" gorm:"index"`
	RequestID string          `json:"request_id" gorm:"not null"`
	Result    string          `json:"result" gorm:"not null"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at" gorm:"index"`
}

func (AuditEvent) TableName() string { return "audit_events" }

// Policy is a configurable rule for a tenant.
type Policy struct {
	ID       string          `json:"id" gorm:"primaryKey"`
	TenantID string          `json:"tenant_id" gorm:"index;not null"`
	Name     string          `json:"name" gorm:"not null"`
	RuleType string          `json:"rule_type" gorm:"not null"`
	Config   json.RawMessage `json:"config" gorm:"type:jsonb"`
	IsActive bool            `json:"is_active" gorm:"default:true"`
	Version  string          `json:"version" gorm:"default:v1"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func (Policy) TableName() string { return "policies" }

// MPCShareRecord stores an encrypted MPC share for threshold signing.
type MPCShareRecord struct {
	ID             string    `json:"id" gorm:"primaryKey"`
	ShareID        uint32    `json:"share_id" gorm:"not null"`
	TenantID       string    `json:"tenant_id" gorm:"index;not null"`
	EncryptedShare []byte    `json:"encrypted_share" gorm:"type:bytea;not null"`
	ShareType      string    `json:"share_type" gorm:"not null"`
	CreatedAt      time.Time `json:"created_at"`
}

func (MPCShareRecord) TableName() string { return "mpc_shares" }

// SigningContext binds a signature to a specific intent/tx.
type SigningContext struct {
	IntentID  string
	TxHash    []byte
	Chain     string
	Domain    string
	Timestamp int64
}

// MPCShare holds a single threshold signing share.
type MPCShare struct {
	ShareID  uint32
	ShareKey []byte
}

// SigningResult is the output of MPC signing.
type SigningResult struct {
	Signature    []byte
	Participants []uint32
}

// ZKProof is a zero-knowledge proof for policy verification.
type ZKProof struct {
	ProofID      string          `json:"proof_id"`
	PublicInputs json.RawMessage `json:"public_inputs"`
	ProofBytes   []byte          `json:"-"`
}

// ReplayKey prevents replay of intent executions.
type ReplayKey struct {
	IntentID string
	Chain    string
	Version  uint64
	Used     bool
}

// NewIntent creates a new intent in draft status.
func NewIntent(tenantID, walletID, destination, token, chain, creator string, amount int64) *Intent {
	now := time.Now().UTC()
	return &Intent{
		ID:          GenerateUUID(),
		TenantID:    tenantID,
		WalletID:    walletID,
		Destination: destination,
		Token:       token,
		Amount:      fmt.Sprintf("%d", amount),
		Chain:       chain,
		Nonce:       generateNonce(),
		Creator:     creator,
		Status:      "draft",
		Expiry:      now.Add(1 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GenerateUUID returns a new unique identifier.
func GenerateUUID() string {
	return fmt.Sprintf("uuid-%d", time.Now().UnixNano())
}

// GenerateRequestID returns a unique request ID for idempotency.
func GenerateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// ComputeIntentHash produces a SHA-256 digest over the canonical intent fields.
func ComputeIntentHash(intent *Intent) []byte {
	h := sha256.New()
	h.Write([]byte(intent.ID))
	h.Write([]byte(intent.TenantID))
	h.Write([]byte(intent.WalletID))
	h.Write([]byte(intent.Destination))
	h.Write([]byte(intent.Token))
	h.Write([]byte(intent.Amount))
	h.Write([]byte(intent.Chain))
	h.Write([]byte(intent.Nonce))
	h.Write([]byte(intent.Creator))
	h.Write([]byte(intent.Status))
	return h.Sum(nil)
}

// AmountToString converts an int64 amount to its string representation.
func AmountToString(amount int64) string {
	return fmt.Sprintf("%d", amount)
}

func generateNonce() string {
	return fmt.Sprintf("nonce-%d", time.Now().UnixNano())
}
