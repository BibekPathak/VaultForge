#!/usr/bin/env bash
set -euo pipefail

# VaultForge Database Seeder
# Seeds initial test data for local development

DATABASE_URL="${DATABASE_URL:-host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable}"

echo "Seeding VaultForge database..."

psql "$DATABASE_URL" << 'SQL'
-- Tenant
INSERT INTO tenants (id, name, domain, created_at, updated_at)
VALUES ('tenant-1', 'Acme Corp', 'acme.vaultforge.io', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Wallets
INSERT INTO wallets (id, tenant_id, name, daily_limit, status, created_at, updated_at)
VALUES
  ('wallet-1', 'tenant-1', 'Treasury Primary', 100000000, 'active', NOW(), NOW()),
  ('wallet-2', 'tenant-1', 'Operations', 50000000, 'active', NOW(), NOW()),
  ('wallet-3', 'tenant-1', 'Reserve', 500000000, 'active', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Policies
INSERT INTO policies (id, tenant_id, name, rule_type, config, is_active, version, created_at, updated_at)
VALUES
  ('policy-daily', 'tenant-1', 'Daily Limit 100k', 'daily_limit', '{"limit":100000}', true, 'v1', NOW(), NOW()),
  ('policy-tx', 'tenant-1', 'Single TX 50k', 'single_tx_limit', '{"limit":50000}', true, 'v1', NOW(), NOW()),
  ('policy-tokens', 'tenant-1', 'Allowed Tokens', 'allowed_tokens', '["SOL","USDC","USDT"]', true, 'v1', NOW(), NOW()),
  ('policy-sigs', 'tenant-1', '2-of-3 Required', 'required_signatures', '{"required":2}', true, 'v1', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Test intents
INSERT INTO intents (id, tenant_id, wallet_id, destination, token, amount, chain, nonce, creator, status, policy_version, expiry, created_at, updated_at)
VALUES
  ('intent-draft-1', 'tenant-1', 'wallet-1', 'EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v', 'USDC', '25000', 'solana', 'nonce-draft-1', 'alice@acme.com', 'draft', 'v1', NOW() + INTERVAL '1 hour', NOW(), NOW()),
  ('intent-pending-1', 'tenant-1', 'wallet-1', 'So11111111111111111111111111111111111111112', 'SOL', '10000', 'solana', 'nonce-pending-1', 'bob@acme.com', 'pending', 'v1', NOW() + INTERVAL '1 hour', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

SQL

echo "Seeding complete."
echo ""
echo "Test data:"
echo "  Tenant:    tenant-1 (Acme Corp)"
echo "  Wallets:   wallet-1 (Treasury), wallet-2 (Ops), wallet-3 (Reserve)"
echo "  Policies:  daily_limit, single_tx_limit, allowed_tokens, required_signatures"
echo "  Intents:   intent-draft-1 (draft), intent-pending-1 (pending)"
