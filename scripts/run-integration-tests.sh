#!/usr/bin/env bash
set -euo pipefail

# VaultForge Integration Test Runner
# Runs the full API pipeline against devnet

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "╔══════════════════════════════════════════╗"
echo "║  VaultForge Integration Tests            ║"
echo "╚══════════════════════════════════════════╝"

# ── Check prerequisites ───────────────────────────────────────────────

echo ""
echo "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker not found"
    exit 1
fi

# ── Start infrastructure ──────────────────────────────────────────────

echo ""
echo "Starting PostgreSQL..."
cd "$ROOT_DIR/docker"
docker compose up -d postgres
echo "Waiting for PostgreSQL to be ready..."
sleep 5

# ── Run Go tests ──────────────────────────────────────────────────────

echo ""
echo "Running Go unit tests..."
cd "$ROOT_DIR/services/api"
DATABASE_URL="host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable" \
    go test -v -count=1 -race ./core/... ./routes/...

echo ""
echo "Running Rust crate tests..."
for crate in crypto mpc zk policy types transaction solana; do
    echo "=== $crate ==="
    cd "$ROOT_DIR/crates/$crate"
    cargo test 2>&1 | tail -3
done

# ── Run full Go test suite ────────────────────────────────────────────

echo ""
echo "Running full Go test suite..."
cd "$ROOT_DIR/services/api"
DATABASE_URL="host=localhost user=vaultforge password=vaultforge dbname=vaultforge port=5432 sslmode=disable" \
    go test -v -count=1 -race ./...

# ── Cleanup ───────────────────────────────────────────────────────────

echo ""
echo "Stopping PostgreSQL..."
cd "$ROOT_DIR/docker"
docker compose down

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║  All integration tests passed!           ║"
echo "╚══════════════════════════════════════════╝"
