package core

// IntentStore provides persistence for intents.
type IntentStore interface {
	Create(intent *Intent) error
	GetByID(id string) (*Intent, error)
	Update(intent *Intent) error
	ListByTenant(tenantID, status string) ([]Intent, error)
	GetTransactionByNonce(nonce string) (*Transaction, error)
	GetWallet(id string) (*Wallet, error)
	ListWallets(tenantID string) ([]Wallet, error)
	ListTransactions(tenantID string) ([]Transaction, error)
	ListAuditEvents(tenantID, action string) ([]AuditEvent, error)
}

// WalletStore provides persistence for wallets.
type WalletStore interface {
	GetByID(id string) (*Wallet, error)
	ListByTenant(tenantID string) ([]Wallet, error)
	Create(wallet *Wallet) error
	Update(wallet *Wallet) error
}

// TransactionStore provides persistence for transactions.
type TransactionStore interface {
	Create(tx *Transaction) error
	GetByID(id string) (*Transaction, error)
	GetByIntentID(intentID string) (*Transaction, error)
	Update(tx *Transaction) error
	ListByTenant(tenantID string) ([]Transaction, error)
}

// AuditStore provides persistence for audit events.
type AuditStore interface {
	Create(event *AuditEvent) error
	ListByTenant(tenantID, action string) ([]AuditEvent, error)
}

// ZKVerifier generates and verifies ZK policy proofs.
type ZKVerifier interface {
	GenerateProof(input PolicyInputs) (*ZKProof, error)
	VerifyPolicyProof(proof *ZKProof) (bool, error)
}

// MPCSigner performs threshold signing via MPC.
type MPCSigner interface {
	Sign(input SigningInput) (*SigningResult, error)
}

// Reconciler compares internal state against chain state.
type Reconciler interface {
	Start(intentID string)
}

// TransactionBuilder constructs Solana transactions.
type TransactionBuilder interface {
	Simulate() (*SimulationResult, error)
	Build() []byte
}

// PolicyInputs are the inputs for ZK proof generation.
type PolicyInputs struct {
	Amount        string
	DailyLimit    *int64
	Recipient     string
	PolicyVersion string
	IntentID      string
	WalletID      string
}

// SigningInput contains the data needed for MPC signing.
type SigningInput struct {
	IntentHash []byte
	TxHash     string
	Chain      string
	Context    string
}

// SimulationResult is the output of transaction simulation.
type SimulationResult struct {
	Allowed         bool
	TransactionHash string
	Error           string
}

// IntentAuditor records audit events for intent lifecycle transitions.
type IntentAuditor interface {
	LogIntentCreated(tenantID, actor, intentID, requestID string)
	LogIntentApproved(tenantID, actor, intentID, requestID string, policyResult *PolicyResult)
	LogIntentRejected(tenantID, actor, intentID, requestID string)
	LogIntentExecuted(tenantID, actor, intentID, requestID string)
	LogIntentExpired(tenantID, actor, intentID, requestID string)
	LogIntentConfirmed(tenantID, actor, intentID, requestID string)
	LogIntentSigned(tenantID, actor, intentID, requestID string, participants []uint32)
	LogIntentSubmittedOnChain(tenantID, actor, intentID, requestID, txSignature string)
	LogIntentSimulated(tenantID, actor, intentID, requestID string, allowed bool)
	LogIntentPolicyDenied(tenantID, actor, intentID, requestID, reason string)
	LogIntentZKDenied(tenantID, actor, intentID, requestID string)
	LogIntentFailed(tenantID, actor, intentID, requestID, reason, failureType string)
}

// SolanaSubmitter submits transactions to Solana and polls for confirmation.
type SolanaSubmitter interface {
	SubmitTransaction(txBytes []byte) (*SubmitResult, error)
	WaitForConfirmation(signature string) (bool, error)
	GetRecentBlockhash() (string, error)
}

// StateNotifier sends notifications on state changes.
type StateNotifier interface {
	NotifyIntentStateChange(intent *Intent, eventType string)
	NotifyTransactionConfirmed(tx *Transaction)
}
