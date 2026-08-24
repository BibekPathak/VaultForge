#!/usr/bin/env bash
set -euo pipefail

# VaultForge Service Status
# Shows comprehensive status of the API server
#
# Usage:
#   ./scripts/status.sh            # Show status
#   ./scripts/status.sh --json     # Output as JSON

JSON_MODE=false
if [ "${1:-}" = "--json" ]; then
    JSON_MODE=true
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

API_URL="${API_URL:-http://localhost:8080}"

if [ "$JSON_MODE" = true ]; then
    STATUS="{}"
fi

print_status() {
    local label="$1" value="$2" ok="$3"
    if [ "$JSON_MODE" = true ]; then
        STATUS=$(echo "$STATUS" | python3 -c "import sys,json; d=json.load(sys.stdin); d['$label']='$value'; print(json.dumps(d))" 2>/dev/null || echo "$STATUS")
    else
        if [ "$ok" = "true" ]; then
            echo -e "  ${GREEN}✓${NC} $label: $value"
        elif [ "$ok" = "warn" ]; then
            echo -e "  ${YELLOW}!${NC} $label: $value"
        else
            echo -e "  ${RED}✗${NC} $label: $value"
        fi
    fi
}

if [ "$JSON_MODE" = false ]; then
    echo "=== VaultForge Service Status ==="
    echo ""
fi

# ── Systemd Service ──────────────────────────────────
if command -v systemctl &>/dev/null; then
    SVC_ACTIVE=$(systemctl is-active vaultforge-api 2>/dev/null || echo "inactive")
    SVC_ENABLED=$(systemctl is-enabled vaultforge-api 2>/dev/null || echo "disabled")
    SVC_PID=$(systemctl show -p MainPID --value vaultforge-api 2>/dev/null || echo "0")
    SVC_MEM=$(systemctl show -p MemoryCurrent --value vaultforge-api 2>/dev/null || echo "0")

    if [ "$SVC_ACTIVE" = "active" ]; then
        print_status "Service" "active (PID: $SVC_PID)" "true"
    else
        print_status "Service" "$SVC_ACTIVE" "false"
    fi
    print_status "Boot" "$SVC_ENABLED" "$([ "$SVC_ENABLED" = "enabled" ] && echo true || echo warn)"

    if [ "$SVC_MEM" != "0" ] && [ "$SVC_MEM" != "[not set]" ]; then
        MEM_MB=$((SVC_MEM / 1024 / 1024))
        print_status "Memory" "${MEM_MB} MB" "$([ $MEM_MB -lt 512 ] && echo true || echo warn)"
    fi
fi

# ── Health Check ─────────────────────────────────────
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health" 2>/dev/null || echo "000")
if [ "$HEALTH" = "200" ]; then
    print_status "Health" "healthy (HTTP $HEALTH)" "true"
else
    print_status "Health" "unhealthy (HTTP $HEALTH)" "false"
fi

# ── Readiness Check ──────────────────────────────────
READY=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/ready" 2>/dev/null || echo "000")
if [ "$READY" = "200" ]; then
    print_status "Ready" "ready (HTTP $READY)" "true"
else
    print_status "Ready" "not ready (HTTP $READY)" "warn"
fi

# ── Metrics ──────────────────────────────────────────
METRICS=$(curl -s "$API_URL/metrics" 2>/dev/null || echo "{}")
if echo "$METRICS" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
    GOROUTINES=$(echo "$METRICS" | python3 -c "import sys,json; print(json.load(sys.stdin).get('goroutines', '?'))" 2>/dev/null || echo "?")
    REQUESTS=$(echo "$METRICS" | python3 -c "import sys,json; print(json.load(sys.stdin).get('requests_total', '?'))" 2>/dev/null || echo "?")
    ERRORS=$(echo "$METRICS" | python3 -c "import sys,json; print(json.load(sys.stdin).get('errors_total', '?'))" 2>/dev/null || echo "?")

    print_status "Goroutines" "$GOROUTINES" "$([ "$GOROUTINES" != "?" ] && ([ "$GOROUTINES" -lt 5000 ] 2>/dev/null && echo true || echo warn) || echo warn)"
    print_status "Requests" "$REQUESTS" "true"
    print_status "Errors" "$ERRORS" "$([ "$ERRORS" = "0" ] && echo true || echo warn)"
else
    print_status "Metrics" "unavailable" "false"
fi

# ── Version ──────────────────────────────────────────
VERSION_RESP=$(curl -s "$API_URL/v1/version" 2>/dev/null || echo "{}")
if echo "$VERSION_RESP" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
    VER=$(echo "$VERSION_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('version', '?'))" 2>/dev/null || echo "?")
    ENV=$(echo "$VERSION_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('environment', '?'))" 2>/dev/null || echo "?")
    RUNTIME=$(echo "$VERSION_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('runtime', '?'))" 2>/dev/null || echo "?")
    print_status "Version" "$VER" "true"
    print_status "Environment" "$ENV" "true"
    print_status "Runtime" "$RUNTIME" "true"
else
    print_status "Version" "unavailable" "warn"
fi

# ── Database ─────────────────────────────────────────
if command -v pg_isready &>/dev/null; then
    if pg_isready -h localhost -p 5432 -U vaultforge >/dev/null 2>&1; then
        print_status "PostgreSQL" "connected" "true"
    else
        print_status "PostgreSQL" "disconnected" "false"
    fi
fi

# ── Disk Space ───────────────────────────────────────
DISK_AVAIL=$(df -h / | awk 'NR==2 {print $4}' 2>/dev/null || echo "?")
print_status "Disk Available" "$DISK_AVAIL" "true"

if [ "$JSON_MODE" = true ]; then
    echo "$STATUS"
else
    echo ""
    echo "API: $API_URL"
fi
