#!/usr/bin/env bash
set -euo pipefail

# VaultForge Test Wallet Creator
# Creates funded test wallets on devnet for integration testing

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WALLET_DIR="${SCRIPT_DIR}/../.test-wallets"
CLUSTER="devnet"

echo "Creating test wallets on $CLUSTER..."

mkdir -p "$WALLET_DIR"

# Create deployer wallet
echo "Creating deployer wallet..."
solana-keygen new --no-bip39-passphrase -o "$WALLET_DIR/deployer.json" --force 2>/dev/null

# Create 3 MPC signer wallets
for i in 1 2 3; do
    echo "Creating MPC signer $i wallet..."
    solana-keygen new --no-bip39-passphrase -o "$WALLET_DIR/mpc-signer-${i}.json" --force 2>/dev/null
done

# Create treasury wallet
echo "Creating treasury wallet..."
solana-keygen new --no-bip39-passphrase -o "$WALLET_DIR/treasury.json" --force 2>/dev/null

# Fund wallets
echo ""
echo "Funding test wallets..."

PUBKEYS=(
    "$(solana-keygen pubkey "$WALLET_DIR/deployer.json")"
    "$(solana-keygen pubkey "$WALLET_DIR/mpc-signer-1.json")"
    "$(solana-keygen pubkey "$WALLET_DIR/mpc-signer-2.json")"
    "$(solana-keygen pubkey "$WALLET_DIR/mpc-signer-3.json")"
    "$(solana-keygen pubkey "$WALLET_DIR/treasury.json")"
)

for pk in "${PUBKEYS[@]}"; do
    echo "  Funding $pk..."
    solana airdrop 1 "$pk" --url "$CLUSTER" 2>/dev/null || echo "  WARNING: Failed to fund $pk (rate limited)"
done

echo ""
echo "Test wallets created in: $WALLET_DIR"
echo ""
echo "Wallet addresses:"
for f in "$WALLET_DIR"/*.json; do
    name=$(basename "$f" .json)
    addr=$(solana-keygen pubkey "$f")
    echo "  $name: $addr"
done
echo ""
echo "IMPORTANT: These wallets are for testing only. Never use them with real funds."
