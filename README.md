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
├── deploy/                   # Production deployment configs
│   ├── systemd/              # Systemd unit files
│   ├── nginx/                # Nginx reverse proxy
│   ├── helm/                 # Kubernetes Helm chart
│   ├── logrotate/            # Log rotation
│   └── openapi.yaml          # API specification
├── docker/                   # Dockerfile + docker-compose
├── scripts/                  # Deployment + test scripts
├── tests/                    # Integration + e2e tests
├── docs/                     # Documentation
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
| GET | `/v1/wallets/:id` | Get wallet |
| GET | `/v1/transactions` | List transactions |
| GET | `/v1/audit-events` | List audit events |

Full API specification: [deploy/openapi.yaml](deploy/openapi.yaml)

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
# All tests (96 Go + 48 Rust = 144 total)
make test

# Go tests with race detector
make test-race

# Go tests with coverage report (HTML)
make test-coverage

# Go tests only
cd services/api && go test ./core/... ./routes/...

# Rust tests only
for crate in crypto mpc zk policy transaction solana; do
  cd crates/$$crate && cargo test && cd ../..
done

# Rust benchmarks
cd crates/crypto && cargo bench
cd crates/zk && cargo bench

# Load tests (requires: go install github.com/rakyll/hey@latest)
make load-test

# Integration tests
make integration-test

# Full verification (tests + security scan)
make verify
```

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture](docs/architecture.md) | System design and component overview |
| [Invariants](docs/invariants.md) | 10 system invariants (I1-I10) |
| [Threat Model](docs/threat-model.md) | Security threat analysis |
| [Performance](docs/PERFORMANCE.md) | Benchmarks and latency targets |
| [Deployment](docs/DEPLOYMENT.md) | Devnet deployment guide |
| [Development](docs/DEVELOPMENT.md) | Local setup and workflow |
| [API Examples](docs/EXAMPLES.md) | Curl examples for all endpoints |
| [Troubleshooting](docs/TROUBLESHOOTING.md) | Common issues and fixes |
| [Contributing](CONTRIBUTING.md) | How to contribute |
| [Production Readiness](docs/PRODUCTION_READINESS.md) | Pre-deployment checklist and file inventory |
| [Changelog](CHANGELOG.md) | Version history |

### Architecture Decision Records

| ADR | Decision |
|-----|----------|
| [ADR-001](docs/adr/001-use-frost-for-mpc.md) | Use FROST for MPC threshold signing |
| [ADR-002](docs/adr/002-hash-based-zk-proofs.md) | Hash-based ZK proofs (prototyping) |
| [ADR-003](docs/adr/003-go-for-api-service.md) | Go for API service |
| [ADR-004](docs/adr/004-interface-driven-architecture.md) | Interface-driven architecture |

### Operations

| Document | Description |
|----------|-------------|
| [Runbook](docs/RUNBOOK.md) | Disaster recovery, backup/restore, failover, incident response |
| [Monitoring](deploy/monitoring/) | Grafana dashboard + Prometheus alert rules |

## Disaster Recovery

```bash
# Backup database
make backup-db

# Restore from backup
gunzip -c /var/backups/vaultforge/vaultforge_*.sql.gz | psql vaultforge
```

See [docs/RUNBOOK.md](docs/RUNBOOK.md) for full procedures (RTO: 15min, RPO: 5min).

## Operations

```bash
# Check service status
make status

# Graceful restart (drains in-flight requests)
make restart

# Backup database
make backup-db

# Security scan
make security-scan

# Load test
make load-test
```

See [docs/RUNBOOK.md](docs/RUNBOOK.md) for disaster recovery and incident response procedures.

## Release Process

```bash
# 1. Update CHANGELOG.md
# 2. Bump version in Cargo.toml files and Chart.yaml
# 3. Create and push tag
git tag v0.17.0
git push origin v0.17.0
# 4. CI runs tests → builds Docker → creates GitHub Release
```

See [.github/workflows/release.yml](.github/workflows/release.yml) for the full pipeline.

## Security

- All database queries use parameterized statements (GORM)
- MPC keys never exist in a single location
- Every state transition is audit-logged with request correlation
- Rate limiting enforced per-tenant
- Request timeouts at server and handler level
- CORS configurable via `CORS_ORIGINS` environment variable
- Tenant ID sanitization (strips non-alphanumeric characters)
- API key format validation (length, character set)
- Security headers on all responses (HSTS, CSP, X-Frame-Options, X-Content-Type-Options)
- Production requires `JWT_SECRET` (min 32 characters)
- Systemd hardening (NoNewPrivileges, ProtectSystem, MemoryDenyWriteExecute)
- Security scanning: cargo-audit + gosec + govulncheck

See [docs/threat-model.md](docs/threat-model.md) for the full threat model.

See [docs/invariants.md](docs/invariants.md) for the 10 system invariants.

## License

Proprietary — VaultForge
