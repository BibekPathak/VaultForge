# VaultForge Deployment Guide

## Prerequisites

- [Solana CLI](https://docs.solanalabs.com/cli/install) v1.18+
- [Anchor CLI](https://www.anchor-lang.com/docs/installation) v0.30+
- [Docker](https://docs.docker.com/get-docker/) (for PostgreSQL)
- [Go 1.25](https://go.dev/dl/) (for API service)
- [Rust](https://rustup.rs/) (for crate builds)

## Quick Start

```bash
# 1. Install Solana CLI and Anchor
sh -c "$(curl -sSfL https://release.anza.xyz/v1.18.26/install)"
cargo install --git https://github.com/coral-xyz/anchor avm --locked
avm install 0.30.1
avm use 0.30.1

# 2. Configure for devnet
solana config set --url devnet

# 3. Create deployer wallet
solana-keygen new --no-bip39-passphrase -o ~/.config/solana/vaultforge-deployer.json

# 4. Fund deployer wallet
solana airdrop 5 --url devnet

# 5. Deploy to devnet
./scripts/deploy-devnet.sh

# 6. Start API service
make docker-up

# 7. Run integration tests
./scripts/run-integration-tests.sh
```

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│  Client App │────▶│  Go API      │────▶│  Solana Devnet  │
│             │     │  :8080       │     │  vault_policy   │
└─────────────┘     └──────┬───────┘     └─────────────────┘
                           │
                    ┌──────▼───────┐
                    │  PostgreSQL  │
                    │  :5432       │
                    └──────────────┘
```

## Devnet Deployment

The Anchor program `vault_policy` handles on-chain policy verification:

1. **Initialize Vault** — Creates a PDA-owned vault with MPC signer threshold
2. **Set Policy** — Configures daily and per-transaction limits
3. **Verify Intent** — On-chain verification that intent satisfies policy
4. **Record Signer** — On-chain attestation of MPC signing participation

### Program Commands

```bash
# Build
anchor build

# Deploy
anchor deploy --provider.cluster devnet

# Verify
solana program show <PROGRAM_ID> --url devnet
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API server port |
| `DATABASE_URL` | (local pg) | PostgreSQL connection string |
| `SOLANA_RPC_URL` | `https://api.devnet.solana.com` | Solana RPC endpoint |
| `VAULTFORGE_ENV` | `development` | Environment (development/staging/production) |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown timeout |

## Test Wallets

Create test wallets for devnet testing:

```bash
./scripts/create-test-wallets.sh
```

This creates wallets in `.test-wallets/`:
- `deployer.json` — Deployment authority
- `mpc-signer-{1,2,3}.json` — MPC signer keypairs
- `treasury.json` — Treasury wallet

**WARNING**: Test wallets are for devnet only. Never use with real funds.

## API Endpoints

After deployment, the API is available at `http://localhost:8080`:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Liveness probe |
| `/ready` | GET | Readiness probe (DB check) |
| `/metrics` | GET | Prometheus-compatible metrics |
| `/v1/intents` | POST | Create intent |
| `/v1/intents` | GET | List intents |
| `/v1/intents/:id` | GET | Get intent |
| `/v1/intents/:id/approve` | POST | Approve intent |
| `/v1/intents/:id/execute` | POST | Execute intent |
| `/v1/intents/:id/reject` | POST | Reject intent |
| `/v1/intents/:id/cancel` | POST | Cancel intent |
| `/v1/wallets` | GET | List wallets |
| `/v1/wallets/:id` | GET | Get wallet |
| `/v1/transactions` | GET | List transactions |
| `/v1/audit-events` | GET | List audit events |

## Troubleshooting

### Airdrop Rate Limited
```bash
# Wait 30 seconds and retry, or use faucet:
# https://faucet.solana.com
solana airdrop 2 --url devnet
```

### Program ID Mismatch
Update `declare_id!()` in `programs/vault_policy/src/lib.rs` and `[programs.devnet]` in `Anchor.toml`:

```bash
# Generate new keypair
solana-keygen new -o target/deploy/vault_policy-keypair.json
# Get pubkey
solana-keygen pubkey target/deploy/vault_policy-keypair.json
# Update declare_id! in lib.rs with this value
```

### Database Connection Failed
```bash
# Start PostgreSQL
cd docker && docker compose up -d postgres
# Wait for ready
docker compose exec postgres pg_isready -U vaultforge
```
