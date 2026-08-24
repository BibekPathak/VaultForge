#!/usr/bin/env bash
set -euo pipefail

# VaultForge Devnet End-to-End Test
#
# 1. Creates a funded sender wallet + recipient wallet on devnet
# 2. Builds a REAL SOL transfer transaction with the solana-go SDK
# 3. Signs with the real Ed25519 private key
# 4. Submits through the platform's SolanaClient (base64, exponential backoff)
# 5. Waits for on-chain confirmation
#
# Usage: ./scripts/e2e-devnet.sh [RPC_URL]

RPC_URL="${1:-https://api.devnet.solana.com}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
WALLET_DIR="$ROOT_DIR/.test-wallets"
SENDER="$WALLET_DIR/treasury.json"
RECIPIENT="$WALLET_DIR/recipient.json"
AMOUNT=1000000  # 0.001 SOL in lamports

echo "╔══════════════════════════════════════════╗"
echo "║  VaultForge Real Devnet Transaction      ║"
echo "╚══════════════════════════════════════════╝"
echo "RPC: $RPC_URL"

# ── Ensure wallets ────────────────────────────────────────────────────

mkdir -p "$WALLET_DIR"

if [ ! -f "$SENDER" ]; then
    echo "Creating sender wallet..."
    solana-keygen new --no-bip39-passphrase -o "$SENDER" --force > /dev/null 2>&1
fi
if [ ! -f "$RECIPIENT" ]; then
    echo "Creating recipient wallet..."
    solana-keygen new --no-bip39-passphrase -o "$RECIPIENT" --force > /dev/null 2>&1
fi

SENDER_ADDR=$(solana-keygen pubkey "$SENDER")
RECIPIENT_ADDR=$(solana-keygen pubkey "$RECIPIENT")
echo "Sender:    $SENDER_ADDR"
echo "Recipient: $RECIPIENT_ADDR"

# ── Fund sender (retry — devnet airdrop is often rate-limited) ─────────

BALANCE=$(solana balance "$SENDER_ADDR" --url "$RPC_URL" 2>/dev/null | awk '{print $1}' || echo 0)
echo "Sender balance: ${BALANCE:-0} SOL"

if (( $(echo "${BALANCE:-0} < 0.005" | bc -l 2>/dev/null || echo 1) )); then
    echo "Airdropping 1 SOL to sender (retrying up to 12 times, 30s apart)..."
    FUNDED=0
    for i in $(seq 1 12); do
        if solana airdrop 1 "$SENDER_ADDR" --url "$RPC_URL" 2>/dev/null | grep -q "Signature"; then
            FUNDED=1
            break
        fi
        echo "  attempt $i/12 failed, retrying in 30s..."
        sleep 30
    done
    if [ "$FUNDED" -ne 1 ]; then
        echo "ERROR: could not fund sender (devnet airdrop rate limited)."
        echo "Fund manually: solana airdrop 1 $SENDER_ADDR --url $RPC_URL"
        exit 1
    fi
fi

sleep 3
BALANCE=$(solana balance "$SENDER_ADDR" --url "$RPC_URL" 2>/dev/null | awk '{print $1}' || echo 0)
echo "Sender balance after funding: ${BALANCE} SOL"

# ── Build + submit + confirm ──────────────────────────────────────────

echo ""
echo "=== Running real transaction ==="
cd "$ROOT_DIR/services/api"
OUT=$(go run ./cmd/e2e \
    -sender "$SENDER" \
    -to "$RECIPIENT_ADDR" \
    -amount "$AMOUNT" \
    -rpc "$RPC_URL" 2>&1)
echo "$OUT"

# ── Assert confirmation ───────────────────────────────────────────────

if ! echo "$OUT" | grep -q "CONFIRMED"; then
    echo ""
    echo "FAILED: transaction was not confirmed"
    exit 1
fi

SIG=$(echo "$OUT" | grep -E "^signature:" | awk '{print $2}')
echo ""
echo "✓ REAL devnet transaction confirmed!"
echo "Signature: $SIG"
echo "Explorer:  https://explorer.solana.com/tx/$SIG?cluster=devnet"

# ── Verify recipient balance increased ────────────────────────────────

sleep 3
RECIP_BAL=$(solana balance "$RECIPIENT_ADDR" --url "$RPC_URL" 2>/dev/null | awk '{print $1}' || echo 0)
echo "Recipient balance: ${RECIP_BAL} SOL"
if (( $(echo "${RECIP_BAL:-0} > 0" | bc -l 2>/dev/null || echo 0) )); then
    echo "✓ Recipient received funds on devnet"
else
    echo "WARNING: recipient balance not yet reflected (may still be landing)"
fi
