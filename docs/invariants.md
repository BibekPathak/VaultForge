VAULTFORGE INTENT INVARIANTS
=============================

I1: An intent cannot execute without authorization.
   - An intent must transition from PENDING to APPROVED before execution can begin.
   - The policy engine must evaluate and return ALLOW before any signing path is entered.
   - If policy returns DENY, the intent status transitions to REJECTED and execution stops.

I2: An intent cannot be signed without successful policy verification.
   - ZK policy proof verification must pass before MPC signing is attempted.
   - The MPC signer must never independently decide whether a transaction is allowed.
   - Signing with an expired, rejected, or unauthorized intent is strictly prohibited.

I3: A transaction can execute at most once.
   - Idempotency is enforced via nonce checking: executing the same intent N times
     results in exactly one transaction being submitted to Solana.
   - The replay protection mechanism (ReplayKey) ensures a nonce cannot be reused.
   - Duplicate execution requests receive the same result without creating duplicate transfers.

I4: A failed transaction cannot silently transition to SENT.
   - If transaction simulation fails, the intent status transitions to FAILED.
   - If policy verification fails before signing, the intent status transitions to FAILED.
   - If MPC signing fails (quorum unavailable), the intent status transitions to FAILED.
   - No state transition bypasses failure handling.

I5: MPC signing requires threshold quorum.
   - 2-of-3 MPC signing requires at least 2 signers to participate.
   - One signer cannot produce a valid threshold signature alone.
   - If fewer than the required quorum of signers are available, the intent status
     transitions to FAILED rather than proceeding with an incomplete signature.

I6: No service stores the complete private key.
   - MPC shares (Share A, Share B, Share C) are never combined in one location.
   - No single database record, log entry, or service configuration contains the full private key.
   - Key material is split via DKG and never reconstructed outside the MPC protocol.

I7: Every state transition produces an audit event.
   - Security-sensitive state changes (DRAFT→PENDING, PENDING→APPROVED,
     APPROVED→EXECUTING, EXECUTING→SUBMITTED, SUBMITTED→CONFIRMED, and all
     failure transitions) produce append-only audit events.
   - Audit entries include: timestamp, tenant, actor, action, resource, intent_id,
     request_id, result, and metadata.

I8: A signer cannot modify the transaction after policy verification.
   - The transaction bytes being signed are bound to the approved intent hash.
   - Any modification to the transaction after policy verification invalidates the
     ZK proof and requires re-evaluation.
   - Domain separation via signing context ensures signatures are bound to specific
     intent/transaction combinations.

I9: Expired intents cannot be executed.
   - An intent's expiry (Unix timestamp) is checked before execution.
   - If current time UTC exceeds the intent's expiry, the status transitions to EXPIRED
     and execution is rejected.
   - This check occurs in the ExecuteIntent API endpoint before any signing or simulation.

I10: Concurrent workers cannot execute the same wallet intent simultaneously.
   - The idempotency check (via nonce/replay key) prevents concurrent execution of
     the same intent from producing duplicate transfers.
   - At most one execution goroutine per intent ID can proceed at a time.
   - The system fails closed: if concurrent execution is attempted, only the first
     request succeeds and subsequent requests receive a "duplicate execution detected" response.