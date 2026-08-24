#!/usr/bin/env bash
set -euo pipefail

# VaultForge Integration Tests
# Runs against a live Solana devnet with deployed program
#
# Prerequisites:
#   - Solana CLI configured for devnet
#   - Program deployed (run deploy-devnet.sh first)
#   - API server running locally or accessible via URL
#
# Usage:
#   ./scripts/integration-tests.sh [API_URL] [SOLANA_RPC]

API_URL="${1:-http://localhost:8080}"
SOLANA_RPC="${2:-https://api.devnet.solana.com}"
PASS=0
FAIL=0

assert_status() {
    local desc="$1" expected="$2" actual="$3"
    if [ "$expected" = "$actual" ]; then
        echo "  PASS: $desc (HTTP $actual)"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc (expected HTTP $expected, got $actual)"
        FAIL=$((FAIL + 1))
    fi
}

assert_contains() {
    local desc="$1" needle="$2" haystack="$3"
    if echo "$haystack" | grep -q "$needle"; then
        echo "  PASS: $desc"
        PASS=$((PASS + 1))
    else
        echo "  FAIL: $desc (missing: $needle)"
        FAIL=$((FAIL + 1))
    fi
}

echo "=== VaultForge Integration Tests ==="
echo "API:     $API_URL"
echo "Solana:  $SOLANA_RPC"
echo ""

# ── Health & Readiness ────────────────────────────────
echo "--- Health Checks ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
assert_status "GET /health returns 200" 200 "$STATUS"

BODY=$(curl -s "$API_URL/health")
assert_contains "Health response has status=ok" "ok" "$BODY"

STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/ready")
assert_status "GET /ready returns 200 or 503" "" "$STATUS"

# ── Metrics ───────────────────────────────────────────
echo ""
echo "--- Metrics ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/metrics")
assert_status "GET /metrics returns 200" 200 "$STATUS"

# ── Intent Lifecycle ──────────────────────────────────
echo ""
echo "--- Intent Lifecycle ---"
TENANT="X-Tenant-ID: tenant-1"
AUTH="Authorization: Bearer test-token"

# Create intent
CREATE_RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/v1/intents" \
    -H "$AUTH" -H "$TENANT" -H "Content-Type: application/json" \
    -d '{"wallet_id":"wallet-1","destination":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","token":"USDC","amount":1000,"chain":"solana","creator":"integration@test.com"}')
CREATE_STATUS=$(echo "$CREATE_RESP" | tail -1)
CREATE_BODY=$(echo "$CREATE_RESP" | sed '$d')
assert_status "POST /v1/intents creates intent" 201 "$CREATE_STATUS"

INTENT_ID=$(echo "$CREATE_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['intent']['id'])" 2>/dev/null || echo "")
if [ -n "$INTENT_ID" ]; then
    echo "  INFO: Created intent $INTENT_ID"

    # Get intent
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/intents/$INTENT_ID" -H "$AUTH" -H "$TENANT")
    assert_status "GET /v1/intents/:id returns 200" 200 "$STATUS"

    # List intents
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/intents" -H "$AUTH" -H "$TENANT")
    assert_status "GET /v1/intents returns 200" 200 "$STATUS"

    # Approve intent
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/v1/intents/$INTENT_ID/approve" \
        -H "$AUTH" -H "$TENANT" -H "X-Actor: integration@test.com")
    assert_status "POST /v1/intents/:id/approve returns 200" 200 "$STATUS"

    # Reject a different intent
    REJECT_RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/v1/intents" \
        -H "$AUTH" -H "$TENANT" -H "Content-Type: application/json" \
        -d '{"wallet_id":"wallet-1","destination":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","token":"USDC","amount":500,"chain":"solana","creator":"integration@test.com"}')
    REJECT_ID=$(echo "$REJECT_RESP" | sed '$d' | python3 -c "import sys,json; print(json.load(sys.stdin)['intent']['id'])" 2>/dev/null || echo "")
    if [ -n "$REJECT_ID" ]; then
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/v1/intents/$REJECT_ID/reject" \
            -H "$AUTH" -H "$TENANT" -H "X-Actor: integration@test.com")
        assert_status "POST /v1/intents/:id/reject returns 200" 200 "$STATUS"
    fi

    # Cancel a different intent
    CANCEL_RESP=$(curl -s -w "\n%{http_code}" -X POST "$API_URL/v1/intents" \
        -H "$AUTH" -H "$TENANT" -H "Content-Type: application/json" \
        -d '{"wallet_id":"wallet-1","destination":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","token":"USDC","amount":200,"chain":"solana","creator":"integration@test.com"}')
    CANCEL_ID=$(echo "$CANCEL_RESP" | sed '$d' | python3 -c "import sys,json; print(json.load(sys.stdin)['intent']['id'])" 2>/dev/null || echo "")
    if [ -n "$CANCEL_ID" ]; then
        STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/v1/intents/$CANCEL_ID/cancel" \
            -H "$AUTH" -H "$TENANT" -H "X-Actor: integration@test.com")
        assert_status "POST /v1/intents/:id/cancel returns 200" 200 "$STATUS"
    fi
fi

# ── Wallet ────────────────────────────────────────────
echo ""
echo "--- Wallet ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/wallets/wallet-1" -H "$AUTH" -H "$TENANT")
assert_status "GET /v1/wallets/wallet-1 returns 200" 200 "$STATUS"

# ── Transactions ──────────────────────────────────────
echo ""
echo "--- Transactions ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/transactions" -H "$AUTH" -H "$TENANT")
assert_status "GET /v1/transactions returns 200" 200 "$STATUS"

# ── Audit Events ──────────────────────────────────────
echo ""
echo "--- Audit Events ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/v1/audit-events" -H "$AUTH" -H "$TENANT")
assert_status "GET /v1/audit-events returns 200" 200 "$STATUS"

# ── Validation ────────────────────────────────────────
echo ""
echo "--- Input Validation ---"
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/v1/intents" \
    -H "$AUTH" -H "$TENANT" -H "Content-Type: application/json" \
    -d '{"wallet_id":"","destination":"bad","token":"USDC","amount":0,"chain":"solana","creator":"test"}')
assert_status "POST /v1/intents rejects invalid input" 400 "$STATUS"

# ── Solana Devnet ─────────────────────────────────────
echo ""
echo "--- Solana Devnet Connectivity ---"
SOLANA_RESP=$(curl -s -X POST "$SOLANA_RPC" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}')
assert_contains "Solana devnet responds" "result" "$SOLANA_RESP"

SOLANA_SLOT=$(curl -s -X POST "$SOLANA_RPC" \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","id":1,"method":"getSlot"}')
assert_contains "Solana devnet returns slot" "result" "$SOLANA_SLOT"

# ── Summary ───────────────────────────────────────────
echo ""
echo "=== Results ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo ""

if [ "$FAIL" -gt 0 ]; then
    echo "SOME TESTS FAILED"
    exit 1
else
    echo "ALL TESTS PASSED"
    exit 0
fi
