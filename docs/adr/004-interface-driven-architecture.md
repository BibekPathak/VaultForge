# ADR-004: Interface-Driven Architecture

## Status

Accepted

## Context

The API service depends on external systems (Solana RPC, database, MPC signer, ZK verifier). These dependencies must be swappable for testing and deployment.

## Decision

Use Go interfaces for all external dependencies. The `IntentHandler` struct accepts interfaces, not concrete implementations.

```go
type IntentHandler struct {
    store       IntentStore       // interface
    audit       IntentAuditor     // interface
    policy      PolicyEngine      // interface
    zk          ZKVerifier        // interface
    mpc         MPCSigner         // interface
    reconciler  Reconciler        // interface
    solana      SolanaSubmitter   // interface
    webhooks    StateNotifier     // interface
    txStore     TransactionStore  // interface
}
```

## Consequences

### Positive
- Full test coverage via mock implementations
- Easy to swap Solana RPC endpoints (devnet → mainnet)
- Can test without database using in-memory stores
- Clear dependency boundaries
- Supports dependency injection in main.go

### Negative
- More boilerplate (mock types in mocks.go)
- Interface drift if not maintained carefully

### Mitigations
- Export all mock types for reuse in tests
- Interface compliance checks at compile time (`var _ IntentAuditor = (*AuditLogger)(nil)`)
