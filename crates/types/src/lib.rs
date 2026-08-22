use serde_json::Value;
use std::fmt;

// Unique intent identifier (UUIDv7)
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub struct IntentId(pub uuid::Uuid);

// Tenant/institution identifier
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub struct TenantId(pub uuid::Uuid);

// Wallet identifier
#[derive(Clone, Debug, PartialEq, Eq, Hash)]
pub struct WalletId(pub uuid::Uuid);

// Policy version
#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub struct PolicyVersion(pub String);

// nonce/idempotency key
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Nonce(pub String);

// Failure reason for intent
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FailureReason {
    Rejected,
    Expired,
    SimulationFailed,
    SigningFailed,
    PolicyDenied,
    SubmissionFailed,
    ConfirmationTimeout,
    BalanceMismatch,
    PermanentFailure,
}

// Intent status through the lifecycle
#[derive(Clone, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum IntentStatus {
    Draft,
    Pending,
    Approved,
    Executing,
    Submitted,
    Confirmed,
    Rejected,
    Expired,
    Failed,
    PermanentFailure,
}

impl fmt::Display for IntentStatus {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{:?}", self)
    }
}

// Intent contains all information for a transaction
// Serialize/Deserialize handled by Go service, not Rust
#[derive(Clone, Debug)]
pub struct Intent {
    pub id: IntentId,
    pub tenant_id: TenantId,
    pub wallet_id: WalletId,
    pub destination: String,
    pub token: String,
    pub amount: u64,
    pub chain: String,
    pub nonce: Nonce,
    pub creator: String,
    pub approvers: Vec<String>,
    pub required_signatures: u16,
    pub policy_version: PolicyVersion,
    pub expiry: time::Timestamp,
    pub status: IntentStatus,
    pub transaction_signature: Option<Vec<u8>>,
    pub failure_reason: Option<FailureReason>,
}

#[derive(Clone, Debug)]
pub struct IntentTimestamps {
    pub created_at: time::Timestamp,
    pub updated_at: time::Timestamp,
    pub approved_at: Option<time::Timestamp>,
    pub executed_at: Option<time::Timestamp>,
    pub confirmed_at: Option<time::Timestamp>,
}

// Policy engine result
#[derive(Clone, Debug, PartialEq)]
pub enum PolicyResult {
    Allow,
    Deny { reason: String },
}

// Wallet information
#[derive(Clone, Debug)]
pub struct Wallet {
    pub id: WalletId,
    pub tenant_id: TenantId,
    pub name: String,
    pub description: Option<String>,
    pub created_at: time::Timestamp,
    pub status: WalletStatus,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum WalletStatus {
    Active,
    Inactive,
    Frozen,
}

// Transaction record
#[derive(Clone, Debug)]
pub struct Transaction {
    pub id: uuid::Uuid,
    pub intent_id: IntentId,
    pub tenant_id: TenantId,
    pub wallet_id: WalletId,
    pub transaction_bytes: Vec<u8>,
    pub blockhash: Option<String>,
    pub priority_fee: u64,
    pub status: TransactionStatus,
    pub submitted_at: Option<time::Timestamp>,
    pub confirmed_at: Option<time::Timestamp>,
    pub confirmed_signature: Option<String>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum TransactionStatus {
    Pending,
    Submitted,
    Confirmed,
    Failed,
    Replaced,
}

// MPC share share for DKG
#[derive(Clone, Debug)]
pub struct MPCShare {
    pub share_id: u32,
    pub share_bytes: Vec<u8>,
    pub encrypted_with: String,
}

// Signer result
#[derive(Clone, Debug)]
pub struct SigningResult {
    pub intent_hash: Vec<u8>,
    pub transaction_hash: Vec<u8>,
    pub signature: Vec<u8>,
    pub signers_participated: Vec<u32>,
}

// ZK proof for policy verification
#[derive(Clone, Debug)]
pub struct ZKProof {
    pub proof_id: uuid::Uuid,
    pub public_inputs: Value,
    pub private_inputs: Value,
    pub proof_bytes: Vec<u8>,
}

// Audit event record
#[derive(Clone, Debug)]
pub struct AuditEvent {
    pub id: uuid::Uuid,
    pub timestamp: time::Timestamp,
    pub tenant: String,
    pub actor: String,
    pub action: String,
    pub resource: String,
    pub intent_id: Option<IntentId>,
    pub request_id: String,
    pub result: String,
    pub metadata: Value,
}

// Replay protection key
#[derive(Clone, Debug)]
pub struct ReplayKey {
    pub intent_id: IntentId,
    pub chain: String,
    pub version: u64,
    pub used: bool,
}