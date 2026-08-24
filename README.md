# VaultForge

**MPC-threshold institutional treasury & settlement platform for Solana.**

Version 1.0.0 · 150 tests (102 Go + 48 Rust) · [Changelog](CHANGELOG.md)

---

## What it is

VaultForge lets a team move money on Solana **safely and auditably**: every transfer
is an *intent* that must pass a configurable **policy**, a **zero-knowledge** check, and a
**2-of-3 MPC threshold signature** before it hits the chain. No single person or key can
move funds alone, and every step is audit-logged.

```
  1  Client submits an INTENT  (who pays whom, how much, which token)
  2  Auth + validation
  3  POLICY ENGINE checks rules  (daily limits, per-tx limits, allowlists, time windows…)
  4  ZK PROOF proves the intent satisfies policy WITHOUT revealing private limits
  5  Transaction is constructed + SIMULATED
  6  MPC signing — 2 of 3 signer keys must cooperate
  7  Signed tx submitted to Solana devnet/mainnet
  8  RECONCILER polls until on-chain confirmation
  9  Status + signature recorded, webhooks fired
 10  Every step written to the AUDIT LOG
```

**Core concepts**

| Concept | What it is |
|---------|-----------|
| **Intent** | A proposed transfer (wallet, destination, token, amount) with a lifecycle: `pending → approved → executing → confirmed` (or rejected/cancelled/failed) |
| **Policy** | JSON rules enforced off-chain by the API: daily limit, single-tx limit, allowed tokens, required signatures, allowed destinations, time windows, velocity |
| **MPC signing** | FROST 2-of-3 threshold — any 2 of 3 key shares produce a signature; no single key exists |
| **ZK proof** | Hash-based commitment proving `amount ≤ limit` without revealing the limit |
| **Reconciliation** | Background poller that watches Solana for the submitted tx's confirmation |
| **Audit log** | Append-only record of every intent state change with request correlation |

---

## Quick start

```bash
make docker-up                # start PostgreSQL
make seed-db                  # load sample tenant/wallets/policies
cd services/api && go run .   # start the API on :8080
curl http://localhost:8080/health
```

Full local setup: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

---

## Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────────────┐
│  Client     │────▶│  Go REST API     │────▶│  Solana (devnet)    │
│  (curl/SDK) │     │  intent handlers │     │  vault_policy       │
└─────────────┘     │  policy + ZK     │     │  program            │
                    │  MPC + reconciler│     └─────────────────────┘
                    └───────┬──────────┘
                    ┌───────▼──────────┐
                    │  PostgreSQL 16    │  intents, wallets,
                    │  (GORM)           │  transactions, audit
                    └──────────────────┘
```

- **Rust crates** (`crates/`): crypto (SHA-256, AES-GCM, Merkle), FROST MPC + DKG,
  hash-based ZK proofs, policy engine — with Criterion benchmarks.
- **Go API** (`services/api/`): Gin + GORM; interface-driven stores, 9 middleware layers
  (auth, rate-limit, CORS, timeouts, metrics, structured logging), full intent lifecycle.
- **On-chain program** (`programs/vault_policy/`): Anchor program for vault init, policy
  set, intent verify, and signer recording.
- **Deployment** (`deploy/`, `docker/`): distroless Docker image, docker-compose, systemd,
  Nginx, Helm chart, OpenAPI spec, Grafana dashboard + Prometheus alerts.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` `/ready` `/metrics` `/v1/version` | Liveness, readiness, metrics, version |
| POST | `/v1/intents` | Create intent (idempotency-key support) |
| GET | `/v1/intents` `/v1/intents/:id` | List / get intents |
| POST | `/v1/intents/:id/approve` | Policy + ZK approval |
| POST | `/v1/intents/:id/execute` | Simulate → MPC sign → submit |
| POST | `/v1/intents/:id/reject` `/cancel` | Reject / cancel |
| GET | `/v1/wallets/:id` `/v1/transactions` `/v1/audit-events` | Wallets, transactions, audit |

Full spec: [deploy/openapi.yaml](deploy/openapi.yaml) · Usage examples: [docs/EXAMPLES.md](docs/EXAMPLES.md)

## Verify & test

```bash
make test            # 150 tests (102 Go + 48 Rust)
make verify          # race tests + full security scan (cargo-audit, gosec, govulncheck)
make integration-test# live API + Solana devnet checks
make bench           # reproducible Criterion benchmarks
```

## Documentation

[Architecture](docs/architecture.md) · [Invariants](docs/invariants.md) · [Threat model](docs/threat-model.md) ·
[Performance (measured)](docs/PERFORMANCE.md) · [Runbook/DR](docs/RUNBOOK.md) ·
[Deployment](docs/DEPLOYMENT.md) · [Development](docs/DEVELOPMENT.md) ·
[Troubleshooting](docs/TROUBLESHOOTING.md) · [Production readiness](docs/PRODUCTION_READINESS.md) ·
[Contributing](CONTRIBUTING.md) · [Release guide](RELEASE.md)

**Operations:** `make status` · `make restart` · `make backup-db` · `make security-scan` · `make load-test`

## Security

- 2-of-3 MPC — no single key holder can sign alone (invariant I10)
- Policy + ZK checks gate every approval; replay protection via nonce + intent hash
- Every state transition audit-logged (invariant I9)
- Hardened runtime: distroless image, nonroot, systemd sandbox, TLS, rate limiting, security headers

See [docs/invariants.md](docs/invariants.md) and [docs/threat-model.md](docs/threat-model.md).

## License

Proprietary — VaultForge
