# ADR-001: Use FROST for MPC Threshold Signing

## Status

Accepted

## Context

VaultForge requires threshold signing where no single key holder can sign alone. The system needs a 2-of-3 threshold scheme for institutional treasury operations.

Options considered:
1. **FROST (Flexible Round-Optimized Schnorr Threshold)** — IETF draft, widely reviewed
2. **GG20** — GMW-based, more complex, less mature implementations
3. **Custom ECDSA threshold** — Higher risk, more implementation burden

## Decision

Use FROST 2-of-3 for threshold signing.

## Consequences

### Positive
- FROST is a well-studied protocol with formal security proofs
- Simple 2-round signing (sign + combine)
- Compatible with standard Ed25519/Ristretto curves
- DKG produces shares without a trusted dealer
- Active IETF standardization process

### Negative
- Requires 2 online participants for each signature
-签名延迟约100-300ms for multi-party coordination
- Limited to Ed25519/Ristretto curves (not ECDSA/secp256k1)

### Risks
- FROST is still an IETF draft — monitor for changes
- Implementation correctness is critical — extensive testing required
