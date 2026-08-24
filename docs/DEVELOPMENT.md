# Local Development Guide

Step-by-step instructions for running VaultForge locally.

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.25+ | `brew install go` / [go.dev](https://go.dev/dl/) |
| Rust | stable | `curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs \| sh` |
| Docker | 24+ | [docker.com](https://docs.docker.com/get-docker/) |
| Docker Compose | v2+ | Included with Docker Desktop |
| Solana CLI | 1.18+ | `sh -c "$(curl -sSfL https://release.solana.com/v1.18/install)"` |
| Anchor CLI | 0.30+ | `cargo install --git https://github.com/coral-xyz/anchor avm --locked` |
| PostgreSQL | 16 | Via Docker (recommended) or local install |

## Quick Start (5 minutes)

```bash
# 1. Clone the repo
git clone https://github.com/vaultforge/vaultforge.git
cd vaultforge

# 2. Start PostgreSQL
make docker-up

# 3. Copy environment file
cp .env.example .env

# 4. Run the API server
cd services/api && go run .

# 5. Verify it's running
curl http://localhost:8080/health
```

## Detailed Setup

### 1. Start Database

```bash
# Using docker-compose (recommended)
make docker-up

# Or run PostgreSQL directly
docker run -d --name vaultforge-pg \
  -e POSTGRES_USER=vaultforge \
  -e POSTGRES_PASSWORD=vaultforge \
  -e POSTGRES_DB=vaultforge \
  -p 5432:5432 \
  postgres:16-alpine
```

### 2. Configure Environment

```bash
cp .env.example .env
```

Key variables in `.env`:

```bash
# Database
DATABASE_URL="host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable"

# Solana
SOLANA_RPC_URL="https://api.devnet.solana.com"
SOLANA_WS_URL="wss://api.devnet.solana.com"

# Server
PORT=8080
VAULTFORGE_ENV=development
LOG_LEVEL=debug
```

### 3. Run API Server

```bash
cd services/api
go run .
```

The server starts on `http://localhost:8080`.

### 4. Seed Test Data (Optional)

```bash
make seed-db
```

This creates:
- Tenant: `tenant-1` (Acme Corp)
- Wallets: wallet-1 (Treasury), wallet-2 (Ops), wallet-3 (Reserve)
- Policies: daily_limit, single_tx_limit, allowed_tokens, required_signatures
- Test intents: intent-draft-1, intent-pending-1

### 5. Verify

```bash
# Health check
curl http://localhost:8080/health

# Ready check (verifies DB + Solana RPC)
curl http://localhost:8080/ready

# List wallets
curl -H "X-Tenant-ID: tenant-1" http://localhost:8080/v1/wallets/wallet-1
```

## Running Tests

```bash
# All tests (Go + Rust)
make test

# Go tests only
cd services/api && go test ./core/... ./routes/...

# Rust tests only
for crate in crypto mpc zk policy transaction solana; do
  cd crates/$crate && cargo test && cd ../..
done

# Rust benchmarks
cd crates/crypto && cargo bench
cd crates/zk && cargo bench
```

## Project Layout

```
vaultforge/
├── crates/                    # Rust library crates (no workspace)
│   ├── crypto/               # SHA-256, AES-256-GCM, Merkle tree, KDF
│   ├── mpc/                  # FROST 2-of-3 threshold signing + DKG
│   ├── zk/                   # Hash-based ZK policy verification
│   ├── policy/               # 7-rule policy engine + ZK integration
│   ├── types/                # Shared domain types
│   ├── transaction/          # Solana transaction builder (canonical bytes)
│   └── solana/               # Solana utilities + replay key store
├── programs/
│   └── vault_policy/         # Anchor on-chain program (devnet)
├── services/
│   └── api/                  # Go REST API
│       ├── main.go           # Entry point + DI wiring
│       ├── core/             # Business logic, stores, crypto
│       └── routes/           # HTTP handlers + middleware
├── deploy/                   # Production deployment configs
│   ├── systemd/              # Systemd unit files
│   ├── nginx/                # Nginx reverse proxy
│   ├── helm/                 # Kubernetes Helm chart
│   ├── logrotate/            # Log rotation
│   └── openapi.yaml          # API specification
├── docker/                   # Dockerfile + docker-compose
├── scripts/                  # Operational scripts
├── docs/                     # Documentation
│   ├── architecture.md       # System architecture
│   ├── invariants.md         # Invariants I1-I10
│   ├── threat-model.md       # Threat model
│   ├── PERFORMANCE.md        # Performance baseline
│   ├── DEPLOYMENT.md         # Devnet deployment guide
│   ├── DEVELOPMENT.md        # This file
│   ├── TROUBLESHOOTING.md    # Common issues & fixes
│   ├── CONTRIBUTING.md       # Contribution guide
│   └── adr/                  # Architecture Decision Records
├── tests/                    # Integration tests
├── Anchor.toml               # Anchor config
├── Makefile                  # 26 targets
└── CHANGELOG.md              # Version history
```

## IDE Setup

### VS Code / Cursor

Recommended extensions:
- `golang.go` — Go language support
- `rust-lang.rust-analyzer` — Rust language support
- `tamasfe.even-better-toml` — TOML support

### GoLand / RustRover

Import the `.idea/` config or:
- Set Go SDK to 1.25+
- Set Rust toolchain to stable
- Mark `services/api` as Go project root

## Common Development Tasks

### Add a new Rust crate

```bash
mkdir crates/newcrate
cd crates/newcrate
cargo init --lib --name vaultforge-newcrate
# Edit Cargo.toml, add dependencies
# Add to Makefile CRATES list
```

### Add a new Go package

```bash
mkdir -p services/api/core/newpackage
# Create files, import as github.com/vaultforge/vaultforge/services/api/core/newpackage
```

### Add a new API endpoint

1. Add handler in `services/api/routes/`
2. Add interface method in `services/api/core/interfaces.go`
3. Add mock method in `services/api/core/mocks.go`
4. Add store method if needed in `services/api/core/stores.go`
5. Wire in `RegisterRoutes()` in `services/api/routes/intent_routes.go`
6. Add tests

### Run a single test

```bash
# Go
cd services/api && go test -run TestCreateIntent ./core/...

# Rust
cd crates/crypto && cargo test test_sha256_hash
```

## Architecture

See [docs/architecture.md](architecture.md) for the full system design.

See [docs/invariants.md](invariants.md) for the 10 system invariants that must never be violated.

See [docs/threat-model.md](threat-model.md) for the security threat model.
