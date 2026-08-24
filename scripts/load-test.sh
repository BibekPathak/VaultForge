#!/usr/bin/env bash
set -euo pipefail

# VaultForge Load Test Script
# Uses 'hey' for HTTP load testing: https://github.com/rakyll/hey
#
# Prerequisites:
#   go install github.com/rakyll/hey@latest
#
# Usage:
#   ./scripts/load-test.sh [BASE_URL] [CONCURRENCY] [REQUESTS]

BASE_URL="${1:-http://localhost:8080}"
CONCURRENCY="${2:-20}"
REQUESTS="${3:-1000}"
DURATION="${4:-30s}"

echo "=== VaultForge Load Test ==="
echo "Target:   $BASE_URL"
echo "Threads:  $CONCURRENCY"
echo "Requests: $REQUESTS"
echo ""

# Check hey is installed
if ! command -v hey &>/dev/null; then
    echo "ERROR: 'hey' not installed. Install with: go install github.com/rakyll/hey@latest"
    exit 1
fi

# Auth header (test tenant)
AUTH="Authorization: Bearer test-token"
TENANT="X-Tenant-ID: tenant-1"
REQUEST_ID="X-Request-ID: $(uuidgen)"

echo "--- 1. Liveness Check ---"
hey -n 100 -c 5 "$BASE_URL/health"
echo ""

echo "--- 2. Readiness Check ---"
hey -n 100 -c 5 "$BASE_URL/ready"
echo ""

echo "--- 3. Metrics Endpoint ---"
hey -n 100 -c 5 "$BASE_URL/metrics"
echo ""

echo "--- 4. Create Intent (POST /v1/intents) ---"
PAYLOAD='{"wallet_id":"wallet-1","destination":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","token":"USDC","amount":1000,"chain":"solana","creator":"loadtest@acme.com"}'
hey -n "$REQUESTS" -c "$CONCURRENCY" -m POST \
    -H "$AUTH" \
    -H "$TENANT" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" \
    "$BASE_URL/v1/intents"
echo ""

echo "--- 5. List Intents (GET /v1/intents) ---"
hey -n "$REQUESTS" -c "$CONCURRENCY" \
    -H "$AUTH" \
    -H "$TENANT" \
    "$BASE_URL/v1/intents"
echo ""

echo "--- 6. Get Wallet (GET /v1/wallets/wallet-1) ---"
hey -n "$REQUESTS" -c "$CONCURRENCY" \
    -H "$AUTH" \
    -H "$TENANT" \
    "$BASE_URL/v1/wallets/wallet-1"
echo ""

echo "--- 7. List Transactions (GET /v1/transactions) ---"
hey -n "$REQUESTS" -c "$CONCURRENCY" \
    -H "$AUTH" \
    -H "$TENANT" \
    "$BASE_URL/v1/transactions"
echo ""

echo "--- 8. List Audit Events (GET /v1/audit-events) ---"
hey -n "$REQUESTS" -c "$CONCURRENCY" \
    -H "$AUTH" \
    -H "$TENANT" \
    "$BASE_URL/v1/audit-events"
echo ""

echo "--- 9. Concurrent Mixed Workload (30s sustained) ---"
hey -z "$DURATION" -c "$CONCURRENCY" -m POST \
    -H "$AUTH" \
    -H "$TENANT" \
    -H "Content-Type: application/json" \
    -d "$PAYLOAD" \
    "$BASE_URL/v1/intents"
echo ""

echo "=== Load Test Complete ==="
echo ""
echo "Tips for analyzing results:"
echo "  - Look for Status code distribution (should be mostly 200/201)"
echo "  - Check Average/Percentile latencies"
echo "  - For p99 > 500ms, consider scaling (HPA) or profiling"
echo "  - For error rate > 0.1%, investigate rate limiting or DB contention"
