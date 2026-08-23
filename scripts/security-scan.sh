#!/usr/bin/env bash
set -euo pipefail

# VaultForge Security Scanner
# Runs cargo-audit (Rust) and gosec (Go) checks

echo "=== VaultForge Security Scanner ==="
echo ""

ERRORS=0

# --- Rust: cargo-audit ---
echo "--- Rust dependency audit (cargo-audit) ---"
if command -v cargo-audit &>/dev/null; then
    for crate_dir in crates/*/; do
        if [ -f "$crate_dir/Cargo.toml" ]; then
            echo "  Auditing $crate_dir..."
            (cd "$crate_dir" && cargo audit 2>&1) || ERRORS=$((ERRORS + 1))
        fi
    done
else
    echo "  cargo-audit not installed. Install: cargo install cargo-audit"
    echo "  Skipping Rust audit."
fi

echo ""

# --- Go: gosec ---
echo "--- Go static analysis (gosec) ---"
if command -v gosec &>/dev/null; then
    gosec -exclude-generated ./services/api/... 2>&1 || ERRORS=$((ERRORS + 1))
else
    echo "  gosec not installed. Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"
    echo "  Skipping Go security scan."
fi

echo ""

# --- Go: govulncheck ---
echo "--- Go vulnerability check (govulncheck) ---"
if command -v govulncheck &>/dev/null; then
    govulncheck ./services/api/... 2>&1 || ERRORS=$((ERRORS + 1))
else
    echo "  govulncheck not installed. Install: go install golang.org/x/vuln/cmd/govulncheck@latest"
    echo "  Skipping Go vulnerability check."
fi

echo ""

# --- Secrets scan ---
echo "--- Secrets scan (hardcoded keys) ---"
SECRETS_FOUND=0
for pattern in \
    "PRIVATE_KEY" \
    "SECRET_KEY" \
    "API_KEY=.*[A-Za-z0-9]{20}" \
    "password=.*[A-Za-z0-9]{8}" \
    "BEGIN RSA PRIVATE KEY" \
    "BEGIN EC PRIVATE KEY"; do
    if grep -r --include="*.go" --include="*.rs" --include="*.toml" --include="*.yaml" --include="*.yml" --include="*.env" "$pattern" . 2>/dev/null | grep -v "_test.go" | grep -v "mocks.go" | grep -v ".env.example" | grep -v "test" | grep -v "dummy" | grep -v "example"; then
        SECRETS_FOUND=1
    fi
done

if [ "$SECRETS_FOUND" -eq 1 ]; then
    echo "  WARNING: Potential secrets found in source code!"
    ERRORS=$((ERRORS + 1))
else
    echo "  No hardcoded secrets detected."
fi

echo ""

if [ "$ERRORS" -gt 0 ]; then
    echo "=== Scan complete: $ERRORS issues found ==="
    exit 1
else
    echo "=== Scan complete: all clean ==="
    exit 0
fi
