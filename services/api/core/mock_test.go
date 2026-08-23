package core

import (
	"fmt"
	"sync"
)

// mockDBQuerier implements DBQuerier for testing the policy engine.
type mockDBQuerier struct {
	wallets  map[string]*Wallet
	tenants  map[string]*Tenant
	policies map[string][]*Policy
	mu       sync.RWMutex
}

func newMockDBQuerier() *mockDBQuerier {
	return &mockDBQuerier{
		wallets:  make(map[string]*Wallet),
		tenants:  make(map[string]*Tenant),
		policies: make(map[string][]*Policy),
	}
}

func (m *mockDBQuerier) GetWallet(id string) (*Wallet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if w, ok := m.wallets[id]; ok {
		return w, nil
	}
	return nil, fmt.Errorf("wallet %s not found", id)
}

func (m *mockDBQuerier) GetTenant(id string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if t, ok := m.tenants[id]; ok {
		return t, nil
	}
	return nil, fmt.Errorf("tenant %s not found", id)
}

func (m *mockDBQuerier) ListPolicies(tenantID string) ([]*Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if p, ok := m.policies[tenantID]; ok {
		return p, nil
	}
	return nil, nil
}
