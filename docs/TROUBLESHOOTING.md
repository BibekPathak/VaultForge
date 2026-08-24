# Troubleshooting & FAQ

## Common Issues

### "connection refused" when starting API

**Cause:** PostgreSQL is not running.

```bash
# Start PostgreSQL via Docker
make docker-up

# Verify it's running
docker ps | grep vaultforge
```

### "pq: password authentication failed"

**Cause:** Database credentials don't match.

```bash
# Check .env file
cat .env | grep DATABASE_URL

# Reset PostgreSQL password
docker exec vaultforge-pg psql -U postgres -c \
  "ALTER USER vaultforge PASSWORD 'vaultforge';"
```

### "context deadline exceeded" on Solana RPC

**Cause:** Network issue or Solana RPC rate limiting.

```bash
# Test Solana connectivity
curl -X POST https://api.devnet.solana.com \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getHealth"}'

# Switch to a different RPC endpoint
export SOLANA_RPC_URL="https://api.mainnet-beta.solana.com"
```

### "too many connections" error

**Cause:** Database connection pool exhausted.

```bash
# Check active connections
docker exec vaultforge-pg psql -U vaultforge -d vaultforge \
  -c "SELECT count(*) FROM pg_stat_activity WHERE datname='vaultforge';"

# Increase pool size or restart API
```

### Rust tests fail with "thread 'main' panicked"

**Cause:** Missing system dependency or version mismatch.

```bash
# Update Rust toolchain
rustup update stable

# Rebuild all crates
cargo clean && cargo build
```

### "address already in use" on port 8080

**Cause:** Another process is using the port.

```bash
# Find the process
lsof -i :8080

# Kill it or use a different port
export PORT=8081
```

### Benchmarks fail to compile

**Cause:** Criterion requires nightly or specific features.

```bash
# Benchmarks use stable Rust but need criterion
cd crates/crypto && cargo bench
```

### Docker build fails

**Cause:** Out of disk space or Docker daemon issues.

```bash
# Clean Docker cache
docker system prune -a

# Rebuild
make docker-build
```

## Performance Issues

### High latency on /v1/intents/:id/execute

**Expected:** This endpoint calls MPC signing + Solana submission. Typical latency is 200-500ms.

**If higher:**
1. Check Solana RPC latency: `curl https://api.devnet.solana.com`
2. Check DB query performance: enable slow query logging
3. Check goroutine count in `/metrics`

### Memory usage growing over time

**Check:**
1. `/metrics` for goroutine count
2. DB connection pool utilization
3. Log volume (structured logs can be large)

**Fix:** Restart the service, investigate goroutine leaks.

### Rate limiting too aggressive

**Adjust:** Edit `RateLimitMiddleware` in `services/api/routes/middleware.go`:

```go
// Default: 30 requests/second per tenant
limiter := core.NewRateLimiter(30, 60)
```

## FAQ

### Q: How many tests are there?

- **92 Go tests** (71 core + 21 routes)
- **48 Rust tests** (6 crypto + 12 mpc + 7 policy + 4 transaction + 4 solana + 15 zk)
- **140 total**

### Q: What Solana clusters are supported?

- Devnet (default)
- Testnet
- Mainnet Beta

Configure via `SOLANA_RPC_URL` environment variable.

### Q: Can I use SQLite instead of PostgreSQL?

Not recommended. The service uses PostgreSQL-specific features (JSONB, WAL mode). SQLite would require significant store rewrites.

### Q: How do I add a new policy rule type?

1. Add the rule type constant in `core/policy_engine.go`
2. Add validation logic in `PolicyEngine.Evaluate()`
3. Add tests in `core/policy_engine_test.go`
4. Update the policy config JSON schema in docs

### Q: How do I deploy to mainnet?

1. Update `Anchor.toml` cluster to `mainnet-beta`
2. Set `SOLANA_RPC_URL` to a mainnet RPC endpoint
3. Use a production PostgreSQL instance
4. Follow [docs/DEPLOYMENT.md](DEPLOYMENT.md) for production configs

### Q: What's the proof size for ZK verification?

~324 bytes (3 commitments + challenge + response). Well under the 500-2000 byte target.

### Q: How does the reconciler work?

The reconciler polls Solana RPC every 2 seconds (up to 60 attempts) to check if a submitted transaction has been confirmed. On confirmation, it updates the intent status and sends audit + webhook notifications.

### Q: Can I run without MPC signing?

Yes. Set `MPC_ENABLED=false` in your environment. The system will skip MPC signing and submit transactions directly. **Not recommended for production.**

## Getting Help

- Check [docs/](.) for architecture, threat model, and invariants
- Open an issue on GitHub
- Join our Discord (link in README)
