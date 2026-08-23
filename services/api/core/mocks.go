package core

import (
	"fmt"
	"sync"
)

// MockIntentStore is an exported mock IntentStore for use by other packages' tests.
type MockIntentStore struct {
	Intents      map[string]*Intent
	Wallets      map[string]*Wallet
	Transactions map[string]*Transaction
	AuditEvents  map[string]*AuditEvent
	NonceToTx    map[string]*Transaction
	mu           sync.RWMutex
}

func NewMockIntentStore() *MockIntentStore {
	return &MockIntentStore{
		Intents:      make(map[string]*Intent),
		Wallets:      make(map[string]*Wallet),
		Transactions: make(map[string]*Transaction),
		AuditEvents:  make(map[string]*AuditEvent),
		NonceToTx:    make(map[string]*Transaction),
	}
}

func (m *MockIntentStore) Create(intent *Intent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Intents[intent.ID] = intent
	return nil
}

func (m *MockIntentStore) GetByID(id string) (*Intent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if i, ok := m.Intents[id]; ok {
		return i, nil
	}
	return nil, fmt.Errorf("intent %s not found", id)
}

func (m *MockIntentStore) Update(intent *Intent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Intents[intent.ID] = intent
	return nil
}

func (m *MockIntentStore) ListByTenant(tenantID, status string) ([]Intent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Intent
	for _, i := range m.Intents {
		if i.TenantID == tenantID && (status == "" || i.Status == status) {
			result = append(result, *i)
		}
	}
	return result, nil
}

func (m *MockIntentStore) GetTransactionByNonce(nonce string) (*Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if tx, ok := m.NonceToTx[nonce]; ok {
		return tx, nil
	}
	return nil, nil
}

func (m *MockIntentStore) GetWallet(id string) (*Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if w, ok := m.Wallets[id]; ok {
		return w, nil
	}
	return nil, fmt.Errorf("wallet %s not found", id)
}

func (m *MockIntentStore) ListWallets(tenantID string) ([]Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Wallet
	for _, w := range m.Wallets {
		if w.TenantID == tenantID {
			result = append(result, *w)
		}
	}
	return result, nil
}

func (m *MockIntentStore) ListTransactions(tenantID string) ([]Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Transaction
	for _, tx := range m.Transactions {
		if tx.TenantID == tenantID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

func (m *MockIntentStore) ListAuditEvents(tenantID, action string) ([]AuditEvent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []AuditEvent
	for _, e := range m.AuditEvents {
		if e.TenantID == tenantID && (action == "" || e.Action == action) {
			result = append(result, *e)
		}
	}
	return result, nil
}

// MockTransactionStore is an exported mock TransactionStore.
type MockTransactionStore struct {
	Txs map[string]*Transaction
	mu  sync.RWMutex
}

func NewMockTransactionStore() *MockTransactionStore {
	return &MockTransactionStore{Txs: make(map[string]*Transaction)}
}

func (m *MockTransactionStore) Create(tx *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Txs[tx.ID] = tx
	return nil
}

func (m *MockTransactionStore) GetByID(id string) (*Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if tx, ok := m.Txs[id]; ok {
		return tx, nil
	}
	return nil, fmt.Errorf("transaction %s not found", id)
}

func (m *MockTransactionStore) GetByIntentID(intentID string) (*Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, tx := range m.Txs {
		if tx.IntentID == intentID {
			return tx, nil
		}
	}
	return nil, fmt.Errorf("transaction for intent %s not found", intentID)
}

func (m *MockTransactionStore) Update(tx *Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Txs[tx.ID] = tx
	return nil
}

func (m *MockTransactionStore) ListByTenant(tenantID string) ([]Transaction, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []Transaction
	for _, tx := range m.Txs {
		if tx.TenantID == tenantID {
			result = append(result, *tx)
		}
	}
	return result, nil
}

// MockZKVerifier is an exported mock ZKVerifier.
type MockZKVerifier struct {
	GenerateFn func(input PolicyInputs) (*ZKProof, error)
	VerifyFn   func(proof *ZKProof) (bool, error)
}

func (m *MockZKVerifier) GenerateProof(input PolicyInputs) (*ZKProof, error) {
	if m.GenerateFn != nil {
		return m.GenerateFn(input)
	}
	return &ZKProof{ProofID: "mock-proof-id", ProofBytes: []byte(`{"test":"proof"}`)}, nil
}

func (m *MockZKVerifier) VerifyPolicyProof(proof *ZKProof) (bool, error) {
	if m.VerifyFn != nil {
		return m.VerifyFn(proof)
	}
	return true, nil
}

// MockMPCSigner is an exported mock MPCSigner.
type MockMPCSigner struct {
	SignFn func(input SigningInput) (*SigningResult, error)
}

func (m *MockMPCSigner) Sign(input SigningInput) (*SigningResult, error) {
	if m.SignFn != nil {
		return m.SignFn(input)
	}
	return &SigningResult{Signature: make([]byte, 64), Participants: []uint32{1, 2}}, nil
}

// MockReconciler is an exported mock Reconciler.
type MockReconciler struct {
	Started []string
}

func (m *MockReconciler) Start(intentID string) {
	m.Started = append(m.Started, intentID)
}

// MockPolicyDB implements DBQuerier for testing the policy engine.
type MockPolicyDB struct {
	Wallets  map[string]*Wallet
	Tenants  map[string]*Tenant
	Policies map[string][]*Policy
	mu       sync.RWMutex
}

func NewMockPolicyDB() *MockPolicyDB {
	return &MockPolicyDB{
		Wallets:  make(map[string]*Wallet),
		Tenants:  make(map[string]*Tenant),
		Policies: make(map[string][]*Policy),
	}
}

func (m *MockPolicyDB) GetWallet(id string) (*Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if w, ok := m.Wallets[id]; ok {
		return w, nil
	}
	return nil, fmt.Errorf("wallet %s not found", id)
}

func (m *MockPolicyDB) GetTenant(id string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.Tenants[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("tenant %s not found", id)
}

func (m *MockPolicyDB) ListPolicies(tenantID string) ([]*Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.Policies[tenantID]; ok {
		return p, nil
	}
	return nil, nil
}

// MockAuditLogger is a no-op audit logger for testing.
type MockAuditLogger struct{}

func (m *MockAuditLogger) LogIntentCreated(tenant, actor, intentID, requestID string)    {}
func (m *MockAuditLogger) LogIntentApproved(tenant, actor, intentID, requestID string, r *PolicyResult) {}
func (m *MockAuditLogger) LogIntentRejected(tenant, actor, intentID, requestID string)   {}
func (m *MockAuditLogger) LogIntentExecuted(tenant, actor, intentID, requestID string)   {}
func (m *MockAuditLogger) LogIntentExpired(tenant, actor, intentID, requestID string)    {}
func (m *MockAuditLogger) LogIntentConfirmed(tenant, actor, intentID, requestID string)  {}
func (m *MockAuditLogger) LogIntentSigned(tenant, actor, intentID, requestID string, participants []uint32) {}
func (m *MockAuditLogger) LogIntentSubmittedOnChain(tenant, actor, intentID, requestID, txSig string) {}
func (m *MockAuditLogger) LogIntentSimulated(tenant, actor, intentID, requestID string, allowed bool) {}
func (m *MockAuditLogger) LogIntentPolicyDenied(tenant, actor, intentID, requestID, reason string) {}
func (m *MockAuditLogger) LogIntentZKDenied(tenant, actor, intentID, requestID string) {}
func (m *MockAuditLogger) LogIntentFailed(tenant, actor, intentID, requestID, reason, errType string) {}

// MockSolanaClient is a no-op Solana client for testing.
type MockSolanaClient struct{}

func (m *MockSolanaClient) SubmitTransaction(txBytes []byte) (*SubmitResult, error) {
	return &SubmitResult{Success: true, Signature: "mock-sig-123"}, nil
}

func (m *MockSolanaClient) WaitForConfirmation(sig string) (bool, error) {
	return true, nil
}

func (m *MockSolanaClient) GetRecentBlockhash() (string, error) {
	return "mock-blockhash", nil
}

// MockWebhookNotifier is a no-op webhook notifier for testing.
type MockWebhookNotifier struct{}

func (m *MockWebhookNotifier) NotifyIntentStateChange(intent *Intent, eventType string) {}
func (m *MockWebhookNotifier) NotifyTransactionConfirmed(tx *Transaction)               {}
