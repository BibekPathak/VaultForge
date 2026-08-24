# VaultForge Performance Baseline

Measured with [Criterion](https://github.com/bheisler/criterion.rs) on 2026-08-24.
All numbers are **measured** (mean of 100 samples, Criterion statistical estimator), not estimates.

## Reproducing These Numbers

```bash
# Crypto primitives
cd crates/crypto && cargo bench --bench benchmarks -- --save-baseline v1.0.0

# ZK policy
cd crates/zk && cargo bench --bench benchmarks -- --save-baseline v1.0.0

# Compare against a stored baseline
cargo bench --bench benchmarks -- --baseline v1.0.0
```

Raw Criterion output is committed under `benchmarks/results/`.

> Note: benchmark closures use `black_box()` on inputs **and** outputs to prevent
> dead-code elimination and constant hoisting (verified against a fast-fail bug
> in `zk_prove_near_limit`).

## Reference Machine

| Property | Value |
|----------|-------|
| CPU | Intel Core 5 120U, 12 CPUs (10 cores / 12 threads), up to 5.0 GHz |
| RAM | 16 GB (15 GiB usable) |
| OS | Linux 6.17.13-hardened1 (Arch) |
| Rust | 1.93.1 |
| Cargo | 1.93.1 |
| Go | 1.25.13 |
| Criterion | 0.5.1 |

## Crypto Primitives (`cargo bench -p vaultforge-crypto`)

Mean latency (100 samples). Throughput computed from mean time / payload size.

| Operation | Mean latency | Throughput |
|-----------|-------------|------------|
| SHA-256 1 KB | 715 ns | ~1.4 GB/s |
| SHA-256 64 KB | 40.5 µs | ~1.6 GB/s |
| SHA-256 1 MB | 641 µs | ~1.6 GB/s |
| AES-256-GCM encrypt 64 B | 454 ns | ~141 MB/s |
| AES-256-GCM encrypt 1 KB | 1.64 µs | ~625 MB/s |
| AES-256-GCM encrypt 64 KB | 83.5 µs | ~785 MB/s |
| KDF (single SHA-256 key derivation) | 82.7 ns | — |
| Merkle root 8 leaves | 905 ns | — |
| Merkle root 64 leaves | 7.77 µs | — |
| Merkle root 256 leaves | 31.5 µs | — |
| constant_time_eq 32 B (equal) | 5.3 ns | — |
| constant_time_eq 32 B (not equal) | 6.0 ns | — |

> Note: `SimpleKdf::derive_key` performs a single SHA-256 over (password || salt),
> **not** an iterated rounds KDF. The `output_length` argument caps at 32 bytes.
> If an iterated (e.g. PBKDF2/Argon2) KDF is added, re-benchmark this entry.

## ZK Policy (`cargo bench -p vaultforge-zk`)

Mean latency (100 samples).

| Operation | Mean latency |
|-----------|-------------|
| Prove (amount 25,000) | 1.28 µs |
| Prove (amount 49,999, near per-wallet limit) | 1.29 µs |
| Verify proof | 775 ns |
| Full roundtrip (prove + verify) | 2.06 µs |

### Proof Size

- Commitment: 3 × 32 = 96 bytes
- Challenge: 32 bytes
- Response: 3 × 32 = 96 bytes
- Metadata: ~100 bytes
- **Total: ~324 bytes** (well under 500-2000 byte target)

## API Latency Targets

Targets below are design goals to be validated with load tests (`scripts/load-test.sh`):

| Endpoint | Target p50 | Target p99 | Notes |
|----------|-----------|-----------|-------|
| `GET /health` | < 1 ms | < 5 ms | No DB query |
| `GET /ready` | < 50 ms | < 200 ms | DB ping + Solana RPC |
| `POST /v1/intents` | < 10 ms | < 50 ms | DB write + audit |
| `GET /v1/intents` | < 5 ms | < 20 ms | DB read |
| `POST /v1/intents/:id/approve` | < 30 ms | < 100 ms | Policy check + ZK verify |
| `POST /v1/intents/:id/execute` | < 200 ms | < 500 ms | MPC sign + Solana submit |
| `GET /v1/audit-events` | < 5 ms | < 20 ms | DB read |

## Throughput Targets

| Metric | Target | Notes |
|--------|--------|-------|
| Sustained RPS (read) | > 1,000 | GET endpoints |
| Sustained RPS (write) | > 200 | POST endpoints |
| Burst RPS | > 5,000 | Health/metrics |
| Concurrent connections | > 100 | With keepalive |
| P99 latency under load | < 100 ms | Read endpoints |

## Scaling Thresholds

| Metric | Scale Up | Scale Down |
|--------|----------|------------|
| CPU avg | > 70% | < 30% |
| Memory avg | > 80% | < 40% |
| P99 latency | > 200 ms | < 50 ms |
| Error rate | > 0.1% | < 0.01% |
| DB pool utilization | > 80% | < 40% |
| Goroutines | > 10,000 | < 1,000 |

## Load Test Scenarios

| Scenario | Duration | Concurrency | Target |
|----------|----------|-------------|--------|
| Light load | 60s | 5 | Baseline latency |
| Normal load | 120s | 20 | Sustain 200 RPS |
| Stress test | 120s | 100 | Find breaking point |
| Soak test | 30 min | 20 | Memory leak detection |
| Spike test | 60s | 5→100→5 | Recovery time |

## Known Bottlenecks

1. **Solana RPC**: Network round-trip dominates tx latency — use WebSocket subscriptions for confirmation
2. **DB connection pool**: Size limits throughput — monitor pool utilization
3. **MPC signing**: Multi-party coordination — expected latency > 100ms

## Monitoring Checklist

- [ ] Export Prometheus metrics from `/metrics`
- [ ] Set up Grafana dashboard with latency percentiles
- [ ] Alert on P99 > 500ms
- [ ] Alert on error rate > 0.1%
- [ ] Alert on DB pool utilization > 80%
- [ ] Alert on goroutine count > 10,000
