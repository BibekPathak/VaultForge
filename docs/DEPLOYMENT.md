# VaultForge Deployment Guide

## Prerequisites

- [Solana CLI](https://docs.solanalabs.com/cli/install) v1.18+ (includes `cargo-build-sbf`)
- [Docker](https://docs.docker.com/get-docker/) (for PostgreSQL)
- [Go 1.25](https://go.dev/dl/) (for API service)
- [Rust](https://rustup.rs/) (for crate builds)

> **Note on Anchor CLI:** The repo uses Anchor for the on-chain program source, but
> deployment builds with `cargo-build-sbf` and deploys with `solana program deploy`.
> This avoids `anchor build`'s workspace detection, which conflicts with the
> standalone Rust crates in `./crates`.

## Quick Start

```bash
# 1. Configure for devnet
solana config set --url devnet

# 2. Deploy to devnet (creates + funds deployer wallet automatically)
./scripts/deploy-devnet.sh

# 3. Start API service
make docker-up
make seed-db
cd services/api && go run . &

# 4. Run integration tests (API + Solana connectivity)
./scripts/integration-tests.sh

# 5. Execute a REAL devnet SOL transfer (solana-go, real signature + confirmation)
./scripts/e2e-devnet.sh
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

`./scripts/deploy-devnet.sh`:
- Creates/funds `~/.config/solana/vaultforge-deployer.json` (payer)
- Ensures the program keypair exists at `target/deploy/vault_policy-keypair.json`
- Builds with `cargo-build-sbf --tools-version v1.52`
- Deploys with `solana program deploy <so> --program-id <keypair>`
- Verifies with `solana program show <PROGRAM_ID>`

### Program Commands

```bash
# Build (SBF toolchain; newer platform-tools for edition2024 support)
(cd programs/vault_policy && cargo-build-sbf --tools-version v1.52 --manifest-path Cargo.toml)

# Deploy
solana program deploy programs/vault_policy/target/deploy/vault_policy.so \
  --program-id target/deploy/vault_policy-keypair.json \
  --keypair ~/.config/solana/vaultforge-deployer.json --url devnet

# Verify
solana program show 9J4EcFGBxvMqiYBDN9A1Ke4f73iJckGG6ibhqx5W4aX6 --url devnet
```

### Program ID

The deployed program ID is `9J4EcFGBxvMqiYBDN9A1Ke4f73iJckGG6ibhqx5W4aX6`,
derived from `target/deploy/vault_policy-keypair.json` and set in
`declare_id!()` (`programs/vault_policy/src/lib.rs`) and `[programs.devnet]`
(`Anchor.toml`).

## Real End-to-End Transaction

`./scripts/e2e-devnet.sh` executes a real devnet SOL transfer:

1. Creates/funds a sender wallet (`.test-wallets/treasury.json`)
2. Creates a recipient wallet (`.test-wallets/recipient.json`)
3. Builds a real `system.Transfer` transaction with **solana-go**
4. Signs with the real Ed25519 private key
5. Submits through the platform's `SolanaClient` (base64-encoded, exponential backoff)
6. Polls for on-chain confirmation (`WaitForConfirmation`)
7. Prints the signature + explorer URL and verifies the recipient's balance

```bash
./scripts/e2e-devnet.sh            # or: make e2e-devnet
```

> The API's intent **approve** flow is fully real (policy + ZK + audit). The
> **execute** path's MPC signer is still a hash-based simulation; the e2e harness
> performs the real on-chain signing/submission directly with a real keypair.

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
