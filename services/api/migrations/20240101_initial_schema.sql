-- VaultForge Initial Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Wallets table
CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW,
    UNIQUE(tenant_id, name)
);

-- Intents table
CREATE TABLE IF NOT EXISTS intents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    wallet_id UUID REFERENCES wallets(id) ON DELETE CASCADE,
    destination VARCHAR(255) NOT NULL,
    token_mint VARCHAR(255) NOT NULL,
    amount numeric(78) NOT NULL, -- Solana uses 6-9 decimals, enough for any token
    chain VARCHAR(50) NOT NULL DEFAULT 'solana',
    nonce VARCHAR(255) NOT NULL UNIQUE,
    creator VARCHAR(255) NOT NULL,
    required_signatures INTEGER NOT NULL DEFAULT 2,
    policy_version VARCHAR(50) NOT NULL DEFAULT 'v1',
    expiry TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    failure_reason VARCHAR(100),
    transaction_signature BYTEA,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    executed_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    UNIQUE(tenant_id, nonce)
);

-- Indexes for intents
CREATE INDEX IF NOT EXISTS idx_intents_tenant ON intents(tenant_id);
CREATE INDEX IF NOT EXISTS idx_intents_wallet ON intents(wallet_id);
CREATE INDEX IF NOT EXISTS idx_intents_status ON intents(status);
CREATE INDEX IF NOT EXISTS idx_intents_expiry ON intents(expiry);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    intent_id UUID REFERENCES intents(id) ON DELETE CASCADE,
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    wallet_id UUID REFERENCES wallets(id) ON DELETE CASCADE,
    transaction_bytes BYTEA NOT NULL,
    blockhash VARCHAR(255),
    priority_fee INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    submitted_at TIMESTAMPTZ,
    confirmed_at TIMESTAMPTZ,
    confirmation_signature VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for transactions
CREATE INDEX IF NOT EXISTS idx_transactions_intent ON transactions(intent_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_submitted ON transactions(submitted_at);

-- Audit events table
CREATE TABLE IF NOT EXISTS audit_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    actor VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL,
    resource VARCHAR(255) NOT NULL,
    intent_id UUID REFERENCES intents(id) ON DELETE SET NULL,
    request_id VARCHAR(255) NOT NULL,
    result VARCHAR(50) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for audit events
CREATE INDEX IF NOT EXISTS idx_audit_tenant ON audit_events(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_intent ON audit_events(intent_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_events(action);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_events(created_at);

-- Policy configurations table
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    config JSONB NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT 'v1',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW,
    UNIQUE(tenant_id, name, version)
);

-- Index for policies
CREATE INDEX IF NOT EXISTS idx_policies_tenant ON policies(tenant_id);

-- MPC shares table (encrypted shares for 2-of-3 threshold)
CREATE TABLE IF NOT EXISTS mpc_shares (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_id INTEGER NOT NULL, -- Node identifier (1, 2, or 3)
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    encrypted_share BYTEA NOT NULL,
    share_type VARCHAR(50) NOT NULL, -- e.g., "frost", "eddsa"
    created_at TIMESTAMPTZ DEFAULT NOW,
    UNIQUE(tenant_id, share_id)
);

-- Index for mpc shares
CREATE INDEX IF NOT EXISTS idx_mpc_shares_tenant ON mpc_shares(tenant_id);

-- Webhook deliveries table
CREATE TABLE IF NOT EXISTS webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    endpoint_url VARCHAR(255) NOT NULL,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    retry_count INTEGER DEFAULT 0,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(tenant_id, endpoint_url, event_type)
);

-- Index for webhook deliveries
CREATE INDEX IF NOT EXISTS idx_webhook_endpoint ON webhook_deliveries(endpoint_url);
CREATE INDEX IF NOT EXISTS idx_webhook_event ON webhook_deliveries(event_type);
