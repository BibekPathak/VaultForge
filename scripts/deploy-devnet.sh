#!/usr/bin/env bash
set -euo pipefail

# VaultForge Devnet Deployment Script
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

if ! command -v anchor &> /dev/null; then
    echo "ERROR: Anchor CLI not found. Install: https://www.anchor-lang.com/docs/installation"
    exit 1
fi

# ── Configuration ──────────────────────────────────────────────────────

CLUSTER="devnet"
WALLET="${HOME}/.config/solana/vaultforge-deployer.json"
PROGRAM_NAME="vault_policy"

echo "Cluster:     $CLUSTER"
echo "Wallet:      $WALLET"
echo "Program:     $PROGRAM_NAME"

# ── Create deployer wallet if missing ──────────────────────────────────

if [ ! -f "$WALLET" ]; then
    echo ""
    echo "Deployer wallet not found. Creating at $WALLET..."
    solana-keygen new --no-bip39-passphrase -o "$WALLET" --force
    echo "Wallet created."
fi

# ── Configure Solana CLI ───────────────────────────────────────────────

echo ""
echo "Configuring Solana CLI for $CLUSTER..."
solana config set --url "$CLUSTER" --keypair "$WALLET"

# ── Request airdrop ───────────────────────────────────────────────────

BALANCE=$(solana balance | awk '{print $1}')
echo "Current balance: $BALANCE SOL"

if (( $(echo "$BALANCE < 1" | bc -l) )); then
    echo "Requesting 2 SOL airdrop..."
    solana airdrop 2 --url "$CLUSTER" || {
        echo "WARNING: Airdrop failed (rate limited). Fund wallet manually:"
        echo "  solana airdrop 2 --url $CLUSTER"
        echo "  Or visit: https://faucet.solana.com"
    }
    sleep 5
fi

# ── Build ──────────────────────────────────────────────────────────────

echo ""
echo "Building Anchor program..."
cd "$ROOT_DIR"
anchor build

# ── Extract program ID ────────────────────────────────────────────────

echo ""
echo "Program ID from Anchor.toml:"
PROGRAM_ID=$(grep -A2 "\[programs.devnet\]" Anchor.toml | grep "$PROGRAM_NAME" | cut -d'"' -f2)
echo "  $PROGRAM_ID"

# ── Deploy ─────────────────────────────────────────────────────────────

echo ""
echo "Deploying $PROGRAM_NAME to $CLUSTER..."
anchor deploy --provider.cluster "$CLUSTER" --program-name "$PROGRAM_NAME" || {
    echo ""
    echo "ERROR: Deployment failed."
    echo ""
    echo "Common issues:"
    echo "  1. Program ID mismatch: Update declare_id! in programs/$PROGRAM_NAME/src/lib.rs"
    echo "     and [programs.devnet] in Anchor.toml"
    echo "  2. Insufficient SOL: Request more airdrop"
    echo "  3. Build errors: Run 'anchor build' first"
    exit 1
}

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
echo "  1. Run integration tests:  make test-integration"
echo "  2. Create test wallets:    ./scripts/create-test-wallets.sh"
echo "  3. Start API server:       make docker-up"
