VAULTFORGE THREAT MODEL
========================

## Overview

This document identifies and categorizes threats to the VaultForge system. The model assumes a defense-in-depth approach with multiple independent layers of protection. The system must fail closed - if any layer fails, the transaction must not proceed.

## Threat Categories

### 1. Compromised API Server

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Unauthorized intent creation | Malicious intents injected | Input validation, rate limiting, authenticated APIs, JWT/session auth |
| Intent tampering | Modified destination/amount | Signed intents, immutable intent ID, cryptographic bindings |
| Policy bypass | Skipping policy checks | Policy engine as separate service, deterministic evaluation, audit logging |
| Webhook forgery | Fake chain events | Signed webhooks, timestamp validation, request ID correlation |
| Tenant isolation breach | Accessing other tenant's data | Tenant derivation from authenticated identity, not client input; RBAC |

### 2. Compromised Signer Node

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Single node reveals complete key | Full private key exposure | DKG key generation; no node stores complete share set; shares encrypted at rest |
| Node signs unauthorized transactions | Unauthorized transfers | Signing context binds to intent hash + transaction hash; threshold quorum required |
| MPC share leakage | Reduced signing threshold | Encrypted share storage; zeroization after use; regular share rotation |
| Node becomes single point of failure | Denial of service | 2-of-3 quorum; any 2 of 3 nodes can sign; healthy node set monitoring |

### 3. Malicious Signer

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Signer colludes to steal funds | Unauthorized transactions | 2-of-3 design requires 2 colluding nodes; economic incentive alignment; audit of all signatures |
| Signer modifies transaction after policy verification | Transaction substitution | Transaction hash bound to intent; policy verification occurs before MPC; audit event on every signature |
| Partial signature leakage | Cryptanalysis attacks | Partial signatures are threshold-shared; no complete signature until aggregation; immediate zeroization |

### 4. Replayed Intent

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Old intent re-submitted | Duplicate transaction | Nonce/idempotency key per execution; signature bound to intent ID + chain; replay cache |
| Replayed signature | Duplicate transfer | Signature binds to intent hash + transaction hash + network; single-use nonce |

### 5. Replayed Signature

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Signature from one transaction reused | Unrelated transaction authorized | Domain separation per signing context; intent hash binding; transaction hash binding |

### 6. Double Execution

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Same intent executed twice | Double transfer | Idempotency key enforcement; concurrency control; reconciliation detects duplicates |

### 7. Transaction Substitution

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Transaction bytes changed after policy verification | Different transfer than approved | Policy verification on exact transaction bytes; MPC signs canonical representation; hash binding to intent |

### 8. Stale Blockhash

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Expired blockhash causes failure | Transaction rejection | Blockhash validation; recent blockhash tracking; retry with fresh blockhash; re-signing if too much time elapsed |

### 9. RPC Failure

| Threat | Impact | Mitigation |
|--------|--------|------------|
| RPC timeout/errors | Unconfirmed state | Idempotent execution; retry with exponential backoff; reconciliation monitors confirmation; timeout -> PERMANENT_FAILURE |

### 10. Partial MPC Failure

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Only 1 of 3 signers available | Cannot reach quorum | Quorum availability check; graceful failure; FAILED state; not PROCEEDING with partial signature |

### 11. Malicious Policy Input

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Crafted policy denies all transactions | Denial of service | Policy engine is deterministic; open policy format; audited policy code; override requires separate auth |

### 12. Unauthorized Recipient

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Funds sent to unintended recipient | Loss of funds | Policy engine allowed/blocked recipients; ZK proof of recipient approval; multi-sig on destination |

### 13. Compromised Database

| Threat | Impact | Mitigation |
|--------|--------|------------|
| DB read access to other tenant's data | Data exfiltration | Multi-tenant isolation at DB level; tenant_id derived from auth; encrypted at-rest storage; row-level security |

### 14. Leaked Encrypted Key Material

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Encrypted shares leaked | Brute-force attack | Argon2id key derivation; AES-256-GCM encryption; short-lived encrypted blobs; key rotation |

### 15. Concurrent Execution

| Threat | Impact | Mitigation |
|--------|--------|------------|
| Two workers execute same intent | Double transfer | Distributed lock (Redis/postgres advisory lock); intent version locking; idempotency key enforcement |

### 16. Transaction Confirmation Ambiguity

| Threat | Impact | Mitigation |
|--------|--------|------------|
| RPC success but transaction reverted | False confirmation | Reconciliation engine; multiple confirmation checks; indexer-sourced state; never trust RPC alone |

## Attack Surface Summary

| Entry Point | Likelihood | Impact | Risk Score |
|-------------|------------|--------|------------|
| API server | Medium | High | Critical |
| Signer nodes | Low | Critical | Critical |
| Policy engine | Low | High | High |
| RPC connections | Medium | Medium | Medium |
| Database | Low | High | High |
| Webhooks | Medium | Medium | Medium |

## Security Principles

1. **Fail closed**: Every failure point must default to rejecting the transaction
2. **Defense in depth**: No single point of failure should compromise the system
3. **Least privilege**: No service has more access than needed
4. **Auditability**: Every security-sensitive action is logged append-only
5. **Key isolation**: No single entity holds the complete private key
6. **Replay protection**: Signatures and intents are bound to unique contexts
7. **Never trust RPC alone**: Chain state is authoritative; RPC is a cache
8. **Tenant isolation**: Strong boundaries between institutions

## Assumptions

- Solana protocol is correct and finality is achieved at confirmed commitment
- Client applications authenticate properly (JWT, mTLS, etc.)
- Signer nodes are operated by trusted-but-fallible operators
- Database is encrypted at rest and access-controlled
- RPC provider has honest majority (standard Solana assumption)

## Incident Response

- Compromised API: Revoke keys, rotate auth secrets, audit log review
- Compromised signer: DKG re-key, share rotation, node replacement
- Policy breach: Policy code audit, parameter review, tenant notification
- DB compromise: Encryption key rotation, data integrity check, forensic analysis