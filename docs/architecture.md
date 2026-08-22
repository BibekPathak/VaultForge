VAULTFORGE ARCHITECTURE
========================

Components:
- API Service: REST/gRPC entry, request validation, tenant isolation, audit logging
- Policy Engine: Pre-signing ALLOW/DENY evaluation
- ZK Policy Verification: Privacy-preserving policy proofs
- MPC Signer: 2-of-3 FROST threshold signing
- Solana Engine: Transaction building, simulation, signing, submission
- Transaction Builder: Canonical tx representation, Token-2022 support
- Reconciliation Engine: Internal DB vs chain state comparison
- Indexer: Solana event pipeline, event decoding, PostgreSQL, webhooks
- Audit System: Append-only audit trail for security events
- Multi-Tenant Isolation: Tenant-bound resources, RBAC

Data Flow:
Client API -> Request Validation & Auth -> Policy Engine -> ZK Verification ->
Transaction Builder -> Transaction Simulation -> MPC Signer -> Solana RPC Submission ->
Confirmation -> Reconciliation -> Audit Event

Trust Boundaries:
API->Policy: Untrusted client input; deterministic policy decisions
Policy->ZK: Policy output is public; ZK protects private parameters
ZK->MPC: ZK verified; MPC never decides authorization
MPC->Solana: Partial sigs aggregated; no single node has complete key
Solana->Indexer: RPC responses verified; never trusted alone for financial state
Reconciliation->DB: Internal DB source of truth; chain state authoritative

Signing Flow:
1. User submits intent via API
2. Intent validated and stored (DRAFT status)
3. Approvers approve (PENDING->APPROVED)
4. Policy engine evaluates approved intent
5. ZK policy verification runs on private policy params
6. If ALLOW: transaction constructed
7. Transaction simulated on local/devnet SVM
8. If simulation passes: MPC signing triggered
9. 2-of-3 MPC signers produce threshold signature
10. Transaction submitted to Solana
11. Confirmation via webhooks/RPC polling
12. Reconciliation compares internal state vs chain state
13. Audit event recorded for each state transition

Transaction Lifecycle:
DRAFT -> PENDING -> APPROVED -> EXECUTING -> SUBMITTED -> CONFIRMED
     ↓              ↓              ↓              ↓              ↓
REJECTED       EXPIRED       FAILED       PERMANENT_FAILURE

Each intent carries: unique ID, tenant ID, wallet ID, destination, token,
amount, chain, nonce/idempotency key, creator, approvers, required sigs,
policy version, expiry, status, timestamps, tx signature, failure reason.

Solana Interaction:
- Program deployment on Solana
- SPL Token / Token-2022 transfers, associated token accounts
- Transaction building: canonical representation bound to approved intent
- Blockhash handling: validation, expiration, rebuild, re-signing
- Priority fees: compute budget instructions
- Simulation: LiteSVM/Mollusk pre-submission
- Confirmation: webhooks + RPC polling + commitment logic

MPC Architecture:
Key Generation (DKG):
  Share A -> Node A, Share B -> Node B, Share C -> Node C
No node possesses complete private key. Any 2 of 3 can produce valid Ed25519 signature.

Signing:
Intent hash + Transaction hash -> MPC signing round -> Partial signatures ->
Threshold aggregation -> Final Ed25519 signature (binds to intent hash)
Domain separation via signing context. Never sign arbitrary bytes without context.

Policy Engine:
Evaluates BEFORE signing. Supports: max amount, daily spending, allowed/blocked
recipients, allowed tokens, allowed programs, time-based restrictions, required
approval count, per-wallet limits, per-tenant limits. Returns ALLOW/DENY with
structured reason.

ZK Architecture:
Circuit proves amount <= daily_limit without revealing daily_limit.
Public inputs: amount, policy version, intent ID.
Private inputs: daily_limit, per-wallet config.
Constraints: range check, comparison.
Proof size: ~500-2000 bytes.
Generation time: ~200-2000ms.
Verification time: ~5-50ms.
ZK is cryptographic verification layer, NOT replacement for authorization.

Reconciliation Architecture:
Internal DB (source of truth) -> Expected transaction state -> Solana RPC/Indexer ->
Actual chain state -> Reconciliation Result
Detects: missing, failed, confirmed, duplicate, unexpected transactions,
balance mismatches.

Interfaces:
type Signer interface { Sign(intentHash, txHash []byte) (*Signature, error) }
type PolicyEngine interface { Evaluate(*Intent) (PolicyResult, error) }
type ZKVerifier interface { VerifyPolicyProof(*ZKProof) (bool, error) }