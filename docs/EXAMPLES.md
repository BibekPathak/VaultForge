# API Usage Examples

All examples use `http://localhost:8080` as the base URL. Replace with your deployed URL.

## Authentication

Every request must include:

```
Authorization: Bearer <your-api-key>
X-Tenant-ID: <your-tenant-id>
```

## Intent Lifecycle

### 1. Create Intent

```bash
curl -X POST http://localhost:8080/v1/intents \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "Content-Type: application/json" \
  -d '{
    "wallet_id": "wallet-1",
    "destination": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
    "token": "USDC",
    "amount": 25000,
    "chain": "solana",
    "creator": "alice@acme.com"
  }'
```

Response:

```json
{
  "intent": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "tenant-1",
    "wallet_id": "wallet-1",
    "destination": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
    "token": "USDC",
    "amount": "25000",
    "chain": "solana",
    "status": "draft",
    "creator": "alice@acme.com"
  }
}
```

### 2. Approve Intent (Policy + ZK Check)

```bash
curl -X POST http://localhost:8080/v1/intents/INTENT_ID/approve \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Actor: alice@acme.com"
```

### 3. Execute Intent (MPC Sign + Solana Submit)

```bash
curl -X POST http://localhost:8080/v1/intents/INTENT_ID/execute \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Actor: alice@acme.com"
```

Response:

```json
{
  "intent": { ... },
  "tx_hash": "5VERv8NMhJQV..."
}
```

### 4. Get Intent Status

```bash
curl http://localhost:8080/v1/intents/INTENT_ID \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1"
```

### 5. List Intents

```bash
curl "http://localhost:8080/v1/intents?status=pending&limit=10" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1"
```

### 6. Reject Intent

```bash
curl -X POST http://localhost:8080/v1/intents/INTENT_ID/reject \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Actor: bob@acme.com"
```

### 7. Cancel Intent

```bash
curl -X POST http://localhost:8080/v1/intents/INTENT_ID/cancel \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Actor: alice@acme.com"
```

## Idempotent Requests

Use the `X-Idempotency-Key` header to prevent duplicate intents:

```bash
curl -X POST http://localhost:8080/v1/intents \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Idempotency-Key: unique-key-123" \
  -H "Content-Type: application/json" \
  -d '{
    "wallet_id": "wallet-1",
    "destination": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
    "token": "USDC",
    "amount": 10000,
    "chain": "solana",
    "creator": "alice@acme.com"
  }'
```

If the same idempotency key is used with identical parameters, the server returns `409 Conflict`.

## Health & Monitoring

```bash
# Liveness (always 200 if server is running)
curl http://localhost:8080/health

# Readiness (200 if DB + Solana RPC are healthy)
curl http://localhost:8080/ready

# Metrics (JSON with counters and latency)
curl http://localhost:8080/metrics
```

## Wallet Operations

```bash
# Get wallet details
curl http://localhost:8080/v1/wallets/wallet-1 \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1"
```

## Transactions & Audit

```bash
# List transactions
curl "http://localhost:8080/v1/transactions?limit=20" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1"

# List audit events
curl "http://localhost:8080/v1/audit-events?limit=50" \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1"
```

## Error Handling

All errors return JSON:

```json
{
  "error": "validation failed: amount must be positive",
  "code": "VALIDATION_ERROR",
  "request_id": "req-abc123"
}
```

Common HTTP status codes:

| Code | Meaning |
|------|---------|
| 400 | Validation error (bad input) |
| 401 | Missing or invalid auth token |
| 403 | Policy denied the operation |
| 404 | Resource not found |
| 409 | Duplicate (idempotency conflict) |
| 429 | Rate limit exceeded |
| 500 | Internal server error |

## Request Correlation

Include `X-Request-ID` header to trace requests across logs:

```bash
curl -X POST http://localhost:8080/v1/intents \
  -H "Authorization: Bearer test-token" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Request-ID: my-trace-id-123" \
  ...
```

The request ID appears in structured logs and audit events for end-to-end tracing.
