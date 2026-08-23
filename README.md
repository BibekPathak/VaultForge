# VaultForge

MPC-threshold institutional treasury and settlement platform for Solana.

## Overview

VaultForge provides institutional-grade custody with:

- **MPC Threshold Signing** — 2-of-3 FROST threshold signatures, no single key holder can sign alone
- **Policy Engine** — Configurable rules: daily limits, per-tx limits, allowed recipients/tokens, time windows, required signatures
- **ZK Policy Verification** — Hash-based zero-knowledge proofs that intent satisfies policy without revealing private inputs
- **Replay Protection** — Nonce-based and intent-hash-based replay prevention
- **Full Auditability** — Every state transition logged with request correlation

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

## Project Structure

```
vaultforge/
├── crates/                    # Rust library crates
│   ├── crypto/               # SHA-256, AES-GCM, Merkle tree
│   ├── mpc/                  # FROST 2-of-3 threshold signing + DKG
│   ├── zk/                   # ZK policy verification proofs
│   ├── policy/               # Policy engine (7 rule types)
│   ├── types/                # Shared domain types
│   ├── transaction/          # Solana transaction builder
│   └── solana/               # Solana utilities
├── programs/
│   └── vault_policy/         # Anchor on-chain program
├── services/
│   └── api/                  # Go REST API service
│       ├── core/             # Business logic, stores, crypto
│       └── routes/           # HTTP handlers, middleware
├── docker/                   # Dockerfile + docker-compose
├── scripts/                  # Deployment + test scripts
├── tests/                    # Integration + e2e tests
├── docs/                     # Architecture, threat model, invariants
├── Anchor.toml               # Anchor project config
└── Makefile                  # Build automation
```

## Quick Start

### Prerequisites

- Go 1.25+
- Rust (stable)
- Docker
- Solana CLI + Anchor CLI (for on-chain deployment)

### Run Locally

```bash
# Start PostgreSQL
make docker-up

# Run API server
cd services/api && go run .

# Run all tests
make test

# Build everything
make build
```

### Deploy to Devnet

```bash
# Deploy on-chain program
make deploy-devnet

# Start API (connects to devnet)
make docker-up
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness probe |
| GET | `/ready` | Readiness probe |
| GET | `/metrics` | Operational metrics |
| POST | `/v1/intents` | Create intent |
| GET | `/v1/intents` | List intents |
| GET | `/v1/intents/:id` | Get intent |
| POST | `/v1/intents/:id/approve` | Approve (policy + ZK check) |
| POST | `/v1/intents/:id/execute` | Execute (simulate + sign + submit) |
| POST | `/v1/intents/:id/reject` | Reject intent |
| POST | `/v1/intents/:id/cancel` | Cancel intent |
| GET | `/v1/wallets` | List wallets |
| GET | `/v1/transactions` | List transactions |
| GET | `/v1/audit-events` | List audit events |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | API server port |
| `DATABASE_URL` | (local pg) | PostgreSQL connection string |
| `SOLANA_RPC_URL` | `https://api.devnet.solana.com` | Solana RPC endpoint |
| `VAULTFORGE_ENV` | `development` | Environment |
| `LOG_LEVEL` | `info` | Log level |

## Testing

```bash
# Go tests (77 tests)
make test-go

# Rust tests (48 tests)
make test-rust

# All tests
make test

# Integration tests
make test-integration
```

## Security

- All database queries use parameterized statements (GORM)
- MPC keys never exist in a single location
- Every state transition is audit-logged with request correlation
- Rate limiting enforced per-tenant
- Request timeouts at server and handler level
- CORS configured for browser clients

See [docs/threat-model.md](docs/threat-model.md) for the full threat model.

## License

Proprietary — VaultForge
