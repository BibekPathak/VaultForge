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
