# Changelog

All notable changes to VaultForge will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-08-24

### Added
- Reconciler now polls Solana RPC for real transaction confirmation
- Idempotency key support via `X-Idempotency-Key` header on `POST /v1/intents`
- Input validation wired in `CreateIntent` handler
- Exponential backoff retry on Solana `SubmitTransaction`
- Exponential backoff retry on webhook delivery (5 retries)
- Rate limiting middleware (per-tenant token bucket)
- CORS middleware for browser clients
- Per-request timeout middleware
- Request body size limiting via `MAX_BODY_BYTES`
- Structured `ApiError` responses with error codes and request ID correlation
- Solana RPC check in readiness probe
- DB connection pool stats in `/metrics`
- Goroutine count in `/metrics` and shutdown logs
- 500ms drain period on graceful shutdown
- Systemd unit file with security hardening (NoNewPrivileges, ProtectSystem, etc.)
- Nginx reverse proxy config with TLS 1.2+, rate limiting, security headers
- Logrotate configuration for API logs (30-day retention)
- OpenAPI 3.0.3 spec for all API endpoints
- Helm chart for Kubernetes deployment (Deployment, Service, Ingress, HPA, ConfigMap, Secrets)
- Security scanning script (cargo-audit + gosec + govulncheck + secrets scan)
- Load testing script using `hey` (9 endpoint scenarios + sustained mixed workload)
- Rust benchmark suite for crypto primitives (SHA-256, AES-GCM, KDF, Merkle)
- Rust benchmark suite for ZK policy (prove, verify, full roundtrip)
- Integration test script for API lifecycle + Solana devnet connectivity
- Performance baseline documentation with latency targets and scaling thresholds
- Local development setup guide with step-by-step instructions
- Architecture Decision Records (ADRs) for key technical choices
- API usage examples with curl commands for all endpoints
- Troubleshooting & FAQ guide
- Contributing guide with code style, PR templates, and security checklist
- PostgreSQL backup script with pg_dump, compression, integrity check, and retention
- Disaster recovery runbook (backup, restore, failover, incident response, communication templates)
- Release automation CI (tag → test → Docker build/push → GitHub Release)
- Grafana dashboard JSON (11 panels)
- Prometheus alert rules (20 rules)
- `/v1/version` endpoint exposing build version, environment, Go runtime, and platform
- VERSION file for semantic versioning
- API key validation with format, length, and character checks
- Tenant ID sanitization (strips non-alphanumeric chars)
- Security headers middleware (HSTS in production, CSP, X-Frame-Options, X-Content-Type-Options)
- Configurable CORS origins via `CORS_ORIGINS` environment variable
- Production JWT_SECRET validation (required in production, min 32 chars)
- Graceful restart script (drains in-flight requests, verifies health)
- Service status script (health, readiness, metrics, version, DB, disk — text or JSON)
- Docker image hardened: distroless base image, nonroot user, trimmed binary, OCI labels
- CI pipeline: cargo-audit, govulncheck, Trivy Docker scan, gofmt check, coverage artifact
- Production readiness checklist document

### Changed
- Auth middleware now validates Authorization header format and sanitizes tenant IDs
- Config validation enforces JWT_SECRET in production and minimum secret length
- CORS middleware supports configurable origins (not just wildcard)
- Startup log includes version and Go runtime version
- Server header set to "VaultForge" on all responses
- Docker image uses distroless base instead of Alpine
- All crate versions bumped to 1.0.0
- Helm chart version bumped to 1.0.0
- docker-compose.yml: read_only root filesystem, no-new-privileges, resource limits

### Changed
- Upgraded webhook retry from linear to exponential backoff
- Health check now requires Solana RPC connectivity for readiness

## [0.12.0] - 2026-08-23

### Added
- Production resilience: exponential backoff, health checks, connection metrics

## [0.11.0] - 2026-08-23

### Added
- Security hardening: rate limiting, CORS, structured errors, validation, README

## [0.10.0] - 2026-08-23

### Added
- Solana on-chain program (vault_policy) with Anchor
- Devnet deployment scripts and documentation
- Integration test runner

## [0.9.0] - 2026-08-23

### Added
- Comprehensive Go test suite (92 tests)
- Exported mock types for all interfaces
- Handler tests with httptest
- Middleware tests (auth, request ID, metrics, CORS, rate limit, timeout)

## [0.8.0] - 2026-08-23

### Added
- Containerization (multi-stage Dockerfile, docker-compose)
- GitHub Actions CI pipeline
- Makefile with 23 targets
- `.golangci.yml` lint configuration

## [0.7.0] - 2026-08-23

### Added
- Operational infrastructure: health checks, metrics, structured logging
- Graceful shutdown with signal handling
- Environment-based configuration with validation

## [0.6.0] - 2026-08-23

### Added
- Execution pipeline: audit logging, Solana RPC client, webhooks
- Transaction record creation on submission
- Full intent lifecycle in API handlers

## [0.5.0] - 2026-08-23

### Added
- Go service: interfaces, GORM stores, concrete implementations
- Policy engine with JSON config parsing
- MPC signer, ZK verifier, reconciler, transaction builder
- Webhook notifier

## [0.4.0] - 2026-08-23

### Added
- ZK policy verification crate (hash-based Pedersen-like commitments)
- Fiat-Shamir challenge protocol
- Integration with policy engine

## [0.3.0] - 2026-08-23

### Fixed
- DKG compilation errors
- Test bugs in transaction and solana crates
- Unused imports across all crates

## [0.2.0] - 2026-08-23

### Added
- MPC threshold signing (FROST 2-of-3)
- DKG key generation with domain-separated shares
- Signing context with replay protection

## [0.1.0] - 2026-08-23

### Added
- Core crypto primitives (SHA-256, AES-GCM, Merkle tree)
- Architecture documentation
- Threat model
- System invariants
