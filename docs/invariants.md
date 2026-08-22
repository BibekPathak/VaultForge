VAULTFORGE INVARIANTS
=====================

I1: An intent cannot execute without authorization.
    - Intent must have sufficient approvers before transitioning from PENDING to APPROVED
    - Policy engine must return ALLOW before any signing occurs
    - Enforced by: API validation + policy engine + approver quorum

I2: An intent cannot be signed without successful policy verification.
    - Policy engine must return ALLOW for the transaction
    - ZK policy proof must verify successfully
    - Enforced by: Policy engine check before MPC signing initiation

I3: A transaction can execute at most once.
    - Idempotency key enforcement prevents duplicate execution
    - Distributed lock prevents concurrent execution
    - Reconciliation detects and reports duplicate attempts
    - Enforced by: Idempotency key + concurrency control + reconciliation

I4: A failed transaction cannot silently transition to SENT.
    - Failed state is permanent unless explicitly reset
    - Automatic transition to SENT only occurs on confirmed success
    - Enforced by: State machine with explicit transitions

I5: MPC signing requires threshold quorum.
    - 2-of-3 requires exactly 2 of 3 nodes to participate
    - If < 2 nodes available, signing fails with FAILED state
    - Enforced by: MPC quorum check before aggregation

I6: No service stores the complete private key.
    - DKG distributes shares across nodes
    - Shares are encrypted at rest
    - No single node has access to >1 share
    - Enforced by: Key generation protocol + share protection

I7: Every state transition produces an audit event.
    - API mutations create audit entries
    - Policy evaluations create audit entries
    - MPC signing events create audit entries
    - Reconciliation creates audit entries
    - Enforced by: Append-only audit log on every transition

I8: A signer cannot modify the transaction after policy verification.
    - Transaction hash is bound to intent during policy evaluation
    - MPC signs canonical representation of approved intent
    - Any modification invalidates the policy verification
    - Enforced by: Hash binding + policy re-evaluation on change

I9: Expired intents cannot be executed.
    - Expiry timestamp checked before every state transition
    - Expired intents transition to EXPIRED status
    - Enforced by: Expiry check in API middleware + policy engine

I10: Concurrent workers cannot execute the same wallet intent simultaneously.
    - Distributed lock (PostgreSQL advisory lock or Redis) per wallet/intent
    - Only one worker can hold the lock per intent
    - Enforced by: Lock acquisition before EXECUTING transition