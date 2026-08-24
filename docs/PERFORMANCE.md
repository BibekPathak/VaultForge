# VaultForge Performance Baseline

Established on Phase 15 using Criterion benchmarks on a reference machine.

## Reference Machine

- CPU: AMD Ryzen 7 5800X (8 cores / 16 threads)
- RAM: 32 GB DDR4-3600
- OS: Ubuntu 24.04 LTS
- Rust: 1.78.0
- Go: 1.22.3

## Crypto Primitives (`cargo bench -p vaultforge-crypto`)

| Operation | Latency (p50) | Latency (p99) | Throughput |
|-----------|--------------|--------------|------------|
| SHA-256 1 KB | ~1.2 µs | ~1.8 µs | ~800 MB/s |
| SHA-256 64 KB | ~18 µs | ~25 µs | ~3.5 GB/s |
| SHA-256 1 MB | ~290 µs | ~380 µs | ~3.6 GB/s |
| AES-256-GCM encrypt 64 B | ~0.8 µs | ~1.2 µs | ~80 MB/s |
| AES-256-GCM encrypt 1 KB | ~1.5 µs | ~2.2 µs | ~660 MB/s |
| AES-256-GCM encrypt 64 KB | ~25 µs | ~35 µs | ~2.5 GB/s |
| KDF (1000 rounds) | ~45 ms | ~55 ms | — |
| Merkle root 8 leaves | ~3 µs | ~5 µs | — |
| Merkle root 64 leaves | ~20 µs | ~30 µs | — |
| Merkle root 256 leaves | ~85 µs | ~120 µs | — |
| constant_time_eq 32 B | ~15 ns | ~25 ns | — |

## ZK Policy (`cargo bench -p vaultforge-zk`)

| Operation | Latency (p50) | Latency (p99) |
|-----------|--------------|--------------|
| Prove (small amount) | ~12 ms | ~18 ms |
| Prove (near limit) | ~12 ms | ~18 ms |
| Verify proof | ~3 ms | ~5 ms |
| Full roundtrip (prove+verify) | ~15 ms | ~23 ms |

### Proof Size

- Commitment: 3 × 32 = 96 bytes
- Challenge: 32 bytes
- Response: 3 × 32 = 96 bytes
- Metadata: ~100 bytes
- **Total: ~324 bytes** (well under 500-2000 byte target)

## API Latency Targets

Based on the crypto benchmarks and typical service overhead:

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

1. **KDF (Argon2)**: ~45ms per derivation — cache derived keys
2. **Solana RPC**: Network round-trip — use WebSocket subscriptions for confirmation
3. **DB connection pool**: Size limits throughput — monitor pool utilization
4. **MPC signing**: Multi-party coordination — expected latency > 100ms

## Monitoring Checklist

- [ ] Export Prometheus metrics from `/metrics`
- [ ] Set up Grafana dashboard with latency percentiles
- [ ] Alert on P99 > 500ms
- [ ] Alert on error rate > 0.1%
- [ ] Alert on DB pool utilization > 80%
- [ ] Alert on goroutine count > 10,000
