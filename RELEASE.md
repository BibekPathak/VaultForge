# Release Guide — VaultForge 1.0.0

Step-by-step instructions to cut the 1.0.0 release.

## Pre-Release Checklist

```bash
# 1. Full test suite passes
make test

# 2. Race detector clean
make test-race

# 3. Security scan clean
make security-scan

# 4. Lint clean
make lint

# 5. Docker builds
make docker-build

# 6. Helm chart lints
make helm-lint
```

## Release Steps

### 1. Final Commit

```bash
git add -A
git commit -m "chore: VaultForge 1.0.0 release

- Version bump to 1.0.0 across all components
- Docker-compose hardened (read_only, no-new-privileges, resource limits)
- .env.example finalized with all config options
- 150 tests (102 Go + 48 Rust) all passing
- CHANGELOG finalized as 1.0.0"
```

### 2. Tag

```bash
git tag -a v1.0.0 -m "VaultForge 1.0.0 — MPC-threshold institutional treasury for Solana"
git push origin v1.0.0
```

### 3. Automated Release (CI)

The `.github/workflows/release.yml` pipeline will automatically:
1. Run Go + Rust tests
2. Build Docker image (distroless, multi-arch)
3. Push to GitHub Container Registry (ghcr.io)
4. Create GitHub Release with auto-generated notes

### 4. Manual Verification

```bash
# Pull and run the released image
docker pull ghcr.io/vaultforge/vaultforge:1.0.0
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="..." \
  -e SOLANA_RPC_URL="https://api.devnet.solana.com" \
  ghcr.io/vaultforge/vaultforge:1.0.0

# Verify
curl http://localhost:8080/v1/version
# {"version":"1.0.0","environment":"development","runtime":"go1.25","goos":"linux","goarch":"amd64"}
```

### 5. Deploy to Staging

```bash
# Render Helm chart
helm template vaultforge deploy/helm/vaultforge \
  --set image.tag=1.0.0 \
  --set env.VAULTFORGE_ENV=staging \
  > staging.yaml

# Apply to staging cluster
kubectl apply -f staging.yaml

# Run integration tests against staging
API_URL=https://staging-api.vaultforge.io ./scripts/integration-tests.sh
```

### 6. Deploy to Production

```bash
# Render production Helm chart
helm template vaultforge deploy/helm/vaultforge \
  --set image.tag=1.0.0 \
  --set env.VAULTFORGE_ENV=production \
  --set secrets.DATABASE_URL="$DATABASE_URL" \
  --set secrets.JWT_SECRET="$JWT_SECRET" \
  --set secrets.WEBHOOK_SECRET="$WEBHOOK_SECRET" \
  > production.yaml

# Apply to production cluster
kubectl apply -f production.yaml

# Verify
curl https://api.vaultforge.io/v1/version
curl https://api.vaultforge.io/ready
```

### 7. Post-Release

```bash
# Monitor for 24 hours
# - Grafana dashboard: check latency, error rate, goroutines
# - Prometheus alerts: verify no firing alerts
# - Logs: check for unexpected errors

# If issues found, rollback:
kubectl rollout undo deployment/vaultforge
```

## Version Matrix

| Component | Version | File |
|-----------|---------|------|
| VERSION | 1.0.0 | `VERSION` |
| Go API | 1.0.0 | `VERSION` (injected at build) |
| Rust crates | 1.0.0 | `crates/*/Cargo.toml` |
| Helm chart | 1.0.0 | `deploy/helm/vaultforge/Chart.yaml` |
| Docker image | 1.0.0 | Tagged by CI |
| Anchor program | 1.0.0 | `programs/vault_policy/Cargo.toml` |

## Rollback Procedure

If critical issues are found after production deploy:

```bash
# 1. Immediate: rollback Kubernetes deployment
kubectl rollout undo deployment/vaultforge

# 2. If DB migration was involved:
# DO NOT rollback DB — fix forward with a new release

# 3. Notify team
# Post in #incidents Slack channel

# 4. Investigate
kubectl logs -l app=vaultforge --tail=100
```

## Release Notes Template

```markdown
# VaultForge v1.0.0

## Highlights
- MPC threshold signing (FROST 2-of-3)
- ZK policy verification
- 7-rule policy engine
- Full audit trail
- 150 tests

## Breaking Changes
None — first stable release.

## Upgrading
First release — no prior version to upgrade from.

## Assets
- Docker: `ghcr.io/vaultforge/vaultforge:1.0.0`
- Helm: `deploy/helm/vaultforge` (version 1.0.0)
- API Spec: `deploy/openapi.yaml`
```
