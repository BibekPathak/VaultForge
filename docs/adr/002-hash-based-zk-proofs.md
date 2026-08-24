# ADR-002: Hash-Based ZK Proofs for Policy Verification

## Status

Accepted (prototyping phase — to be replaced with Groth16/PLONK for production)

## Context

VaultForge needs zero-knowledge proofs to verify that an intent satisfies policy constraints (e.g., amount <= daily_limit) without revealing private inputs.

Options considered:
1. **Hash-based Pedersen commitments** — Simple, no trusted setup, larger proofs
2. **Groth16 (arkworks/bellman)** — Succinct proofs, requires trusted setup
3. **PLONK** — Universal setup, more flexible, complex implementation

## Decision

Use hash-based Pedersen-like commitments with Fiat-Shamir for prototyping. Plan migration to Groth16 for production.

## Consequences

### Positive
- No trusted setup required
- Simple to implement and audit
- Correct interface and types for easy migration
- Proof generation ~12ms, verification ~3ms (adequate for prototyping)

### Negative
- Proof size ~324 bytes (vs ~128 bytes for Groth16)
- Not post-quantum secure (hash-based is, but current implementation isn't)
- Verification is slower than Groth16 (~3ms vs ~5ms is acceptable)

### Migration Path
- Keep `Prover`/`Verifier` interface stable
- Replace internal implementation with arkworks Groth16
- Update proof size and verification benchmarks
