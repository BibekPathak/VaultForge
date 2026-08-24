#!/usr/bin/env bash
set -euo pipefail

# VaultForge Devnet Deployment Script
#
# Builds the Anchor program with the Solana SBF toolchain (cargo-build-sbf)
# and deploys it to devnet with `solana program deploy`.
#
# Note: `anchor build`/`anchor deploy` are avoided because Anchor's workspace
# detection conflicts with the standalone Rust crates in ./crates.
#
# Usage: ./scripts/deploy-devnet.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

echo "╔══════════════════════════════════════════╗"
echo "║  VaultForge Devnet Deployment            ║"
echo "╚══════════════════════════════════════════╝"

# ── Prerequisites ──────────────────────────────────────────────────────

echo ""
echo "Checking prerequisites..."

if ! command -v solana &> /dev/null; then
    echo "ERROR: Solana CLI not found. Install: https://docs.solanalabs.com/cli/install"
    exit 1
fi

if ! command -v cargo-build-sbf &> /dev/null; then
    echo "ERROR: cargo-build-sbf not found. Install the Solana CLI toolchain."
    exit 1
fi

# ── Configuration ──────────────────────────────────────────────────────

CLUSTER="devnet"
WALLET="${HOME}/.config/solana/vaultforge-deployer.json"
PROGRAM_NAME="vault_policy"
PROGRAM_KEYPAIR="$ROOT_DIR/target/deploy/${PROGRAM_NAME}-keypair.json"
PROGRAM_SO="$ROOT_DIR/programs/${PROGRAM_NAME}/target/deploy/${PROGRAM_NAME}.so"
PLATFORM_TOOLS_VERSION="${PLATFORM_TOOLS_VERSION:-v1.52}"

echo "Cluster:     $CLUSTER"
echo "Wallet:      $WALLET"
echo "Program:     $PROGRAM_NAME"
echo "Program key: $PROGRAM_KEYPAIR"

# ── Create deployer wallet if missing ──────────────────────────────────

if [ ! -f "$WALLET" ]; then
    echo ""
    echo "Deployer wallet not found. Creating at $WALLET..."
    solana-keygen new --no-bip39-passphrase -o "$WALLET" --force > /dev/null
    echo "Wallet created."
fi

# ── Configure Solana CLI ───────────────────────────────────────────────

echo ""
echo "Configuring Solana CLI for $CLUSTER..."
solana config set --url "$CLUSTER" --keypair "$WALLET" > /dev/null

# ── Request airdrop (retries; devnet is often rate-limited) ───────────

BALANCE=$(solana balance 2>/dev/null | awk '{print $1}' || echo 0)
echo "Current balance: ${BALANCE:-0} SOL"

if (( $(echo "${BALANCE:-0} < 1" | bc -l 2>/dev/null || echo 1) )); then
    echo "Requesting 2 SOL airdrop (retrying up to 10 times)..."
    AIRDROP_OK=0
    for i in $(seq 1 10); do
        if solana airdrop 2 --url "$CLUSTER" --keypair "$WALLET" 2>/dev/null | grep -q "Signature"; then
            AIRDROP_OK=1
            break
        fi
        echo "  airdrop attempt $i/10 failed (rate limited?), retrying in 30s..."
        sleep 30
    done
    if [ "$AIRDROP_OK" -ne 1 ]; then
        echo "WARNING: Could not airdrop. Fund the wallet manually:"
        echo "  solana airdrop 2 --url devnet --keypair $WALLET"
        echo "  Or use https://faucet.solana.com"
    fi
    sleep 5
fi

BALANCE=$(solana balance 2>/dev/null | awk '{print $1}' || echo 0)
if (( $(echo "${BALANCE:-0} < 0.5" | bc -l 2>/dev/null || echo 1) )); then
    echo "ERROR: Deployer wallet has insufficient SOL (${BALANCE:-0}). Deploy requires ~1 SOL."
    exit 1
fi
echo "Deployer balance: $BALANCE SOL"

# ── Ensure program keypair exists ──────────────────────────────────────

if [ ! -f "$PROGRAM_KEYPAIR" ]; then
    echo ""
    echo "Program keypair not found at $PROGRAM_KEYPAIR."
    echo "Generate it and set declare_id!/Anchor.toml to its pubkey:"
    echo "  solana-keygen new --no-bip39-passphrase -o $PROGRAM_KEYPAIR"
    echo "  solana-keygen pubkey $PROGRAM_KEYPAIR"
    exit 1
fi

PROGRAM_ID=$(solana-keygen pubkey "$PROGRAM_KEYPAIR")
echo "Program ID: $PROGRAM_ID"

# ── Build ──────────────────────────────────────────────────────────────

echo ""
echo "Building Anchor program with cargo-build-sbf..."
(cd "$ROOT_DIR/programs/$PROGRAM_NAME" && \
    cargo-build-sbf --tools-version "$PLATFORM_TOOLS_VERSION" --manifest-path Cargo.toml)

if [ ! -f "$PROGRAM_SO" ]; then
    echo "ERROR: Build output not found at $PROGRAM_SO"
    exit 1
fi
echo "Built: $PROGRAM_SO"

# ── Deploy ─────────────────────────────────────────────────────────────

echo ""
echo "Deploying $PROGRAM_NAME to $CLUSTER..."
solana program deploy "$PROGRAM_SO" \
    --program-id "$PROGRAM_KEYPAIR" \
    --upgrade-authority "$WALLET" \
    --keypair "$WALLET" \
    --url "$CLUSTER"

# ── Verify ─────────────────────────────────────────────────────────────

echo ""
echo "Verifying deployment..."
solana program show "$PROGRAM_ID" --url "$CLUSTER" || true

echo ""
echo "╔══════════════════════════════════════════╗"
echo "║  Deployment Complete!                    ║"
echo "╚══════════════════════════════════════════╝"
echo ""
echo "Program ID: $PROGRAM_ID"
echo "Cluster:    $CLUSTER"
echo "Explorer:   https://explorer.solana.com/address/$PROGRAM_ID?cluster=devnet"
echo ""
echo "Next steps:"
echo "  1. Create test wallets:    ./scripts/create-test-wallets.sh"
echo "  2. Run a real transfer:    ./scripts/e2e-devnet.sh"
echo "  3. Start API server:       make docker-up"
