package core

import (
	"gorm.io/gorm"
)

// DBAdapter wraps *gorm.DB to implement DBQuerier for the policy engine.
type DBAdapter struct {
	db *gorm.DB
}

func NewDBAdapter(db *gorm.DB) *DBAdapter {
	return &DBAdapter{db: db}
}

func (a *DBAdapter) GetWallet(id string) (*Wallet, error) {
	var wallet Wallet
	if err := a.db.Where("id = ?", id).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (a *DBAdapter) GetTenant(id string) (*Tenant, error) {
	var tenant Tenant
	if err := a.db.Where("id = ?", id).First(&tenant).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (a *DBAdapter) ListPolicies(tenantID string) ([]*Policy, error) {
	var policies []*Policy
	if err := a.db.Where("tenant_id = ?", tenantID).Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// PostgresIntentStore implements IntentStore using GORM.
type PostgresIntentStore struct {
	db *gorm.DB
}

func NewPostgresIntentStore(db *gorm.DB) *PostgresIntentStore {
	return &PostgresIntentStore{db: db}
}

func (s *PostgresIntentStore) Create(intent *Intent) error {
	return s.db.Create(intent).Error
}

func (s *PostgresIntentStore) GetByID(id string) (*Intent, error) {
	var intent Intent
	if err := s.db.Where("id = ?", id).First(&intent).Error; err != nil {
		return nil, err
	}
	return &intent, nil
}

func (s *PostgresIntentStore) Update(intent *Intent) error {
	return s.db.Save(intent).Error
}

func (s *PostgresIntentStore) ListByTenant(tenantID, status string) ([]Intent, error) {
	var intents []Intent
	query := s.db.Where("tenant_id = ?", tenantID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Order("created_at DESC").Find(&intents).Error; err != nil {
		return nil, err
	}
	return intents, nil
}

func (s *PostgresIntentStore) GetTransactionByNonce(nonce string) (*Transaction, error) {
	var intent Intent
	if err := s.db.Where("nonce = ?", nonce).First(&intent).Error; err != nil {
		return nil, err
	}
	// If intent exists and has been executed, return its transaction
	var tx Transaction
	if err := s.db.Where("intent_id = ?", intent.ID).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (s *PostgresIntentStore) GetWallet(id string) (*Wallet, error) {
	var wallet Wallet
	if err := s.db.Where("id = ?", id).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (s *PostgresIntentStore) ListWallets(tenantID string) ([]Wallet, error) {
	var wallets []Wallet
	if err := s.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

func (s *PostgresIntentStore) ListTransactions(tenantID string) ([]Transaction, error) {
	var txs []Transaction
	if err := s.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

func (s *PostgresIntentStore) ListAuditEvents(tenantID, action string) ([]AuditEvent, error) {
	var events []AuditEvent
	query := s.db.Where("tenant_id = ?", tenantID)
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// PostgresWalletStore implements WalletStore using GORM.
type PostgresWalletStore struct {
	db *gorm.DB
}

func NewPostgresWalletStore(db *gorm.DB) *PostgresWalletStore {
	return &PostgresWalletStore{db: db}
}

func (s *PostgresWalletStore) GetByID(id string) (*Wallet, error) {
	var wallet Wallet
	if err := s.db.Where("id = ?", id).First(&wallet).Error; err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (s *PostgresWalletStore) ListByTenant(tenantID string) ([]Wallet, error) {
	var wallets []Wallet
	if err := s.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&wallets).Error; err != nil {
		return nil, err
	}
	return wallets, nil
}

func (s *PostgresWalletStore) Create(wallet *Wallet) error {
	return s.db.Create(wallet).Error
}

func (s *PostgresWalletStore) Update(wallet *Wallet) error {
	return s.db.Save(wallet).Error
}

// PostgresTransactionStore implements TransactionStore using GORM.
type PostgresTransactionStore struct {
	db *gorm.DB
}

func NewPostgresTransactionStore(db *gorm.DB) *PostgresTransactionStore {
	return &PostgresTransactionStore{db: db}
}

func (s *PostgresTransactionStore) Create(tx *Transaction) error {
	return s.db.Create(tx).Error
}

func (s *PostgresTransactionStore) GetByID(id string) (*Transaction, error) {
	var tx Transaction
	if err := s.db.Where("id = ?", id).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (s *PostgresTransactionStore) GetByIntentID(intentID string) (*Transaction, error) {
	var tx Transaction
	if err := s.db.Where("intent_id = ?", intentID).First(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (s *PostgresTransactionStore) Update(tx *Transaction) error {
	return s.db.Save(tx).Error
}

func (s *PostgresTransactionStore) ListByTenant(tenantID string) ([]Transaction, error) {
	var txs []Transaction
	if err := s.db.Where("tenant_id = ?", tenantID).Order("created_at DESC").Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}

// PostgresAuditStore implements AuditStore using GORM.
type PostgresAuditStore struct {
	db *gorm.DB
}

func NewPostgresAuditStore(db *gorm.DB) *PostgresAuditStore {
	return &PostgresAuditStore{db: db}
}

func (s *PostgresAuditStore) Create(event *AuditEvent) error {
	return s.db.Create(event).Error
}

func (s *PostgresAuditStore) ListByTenant(tenantID, action string) ([]AuditEvent, error) {
	var events []AuditEvent
	query := s.db.Where("tenant_id = ?", tenantID)
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if err := query.Order("created_at DESC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
