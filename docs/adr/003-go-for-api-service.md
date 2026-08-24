# ADR-003: Go for API Service

## Status

Accepted

## Context

VaultForge needs a REST API service to orchestrate intent lifecycle, policy checks, MPC signing, and Solana submission.

Options considered:
1. **Go** — Fast compilation, excellent concurrency, strong stdlib
2. **Rust (Actix/Axum)** — Performance, type safety, shared crate ecosystem
3. **Node.js (Fastify)** — Rapid development, large ecosystem

## Decision

Use Go for the API service.

## Consequences

### Positive
- Excellent standard library for HTTP, JSON, testing
- Goroutines for concurrent request handling
- Fast compile times for rapid iteration
- Mature ecosystem (GORM, gin, etc.)
- Strong typing catches errors at compile time
- Team familiarity

### Negative
- Separate language from Rust crates (requires FFI or HTTP boundary)
- No shared types between Go and Rust (mitigated by interface boundaries)
- GC pauses (mitigated by tuning GOGC)

### Mitigations
- Rust crates expose simple data types (JSON-serializable structs)
- Interface-based design allows swapping implementations
- Go tests use mocks that mirror Rust crate behavior
