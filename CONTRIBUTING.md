# Contributing to VaultForge

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USER/vaultforge.git`
3. Create a branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Commit and push
7. Open a Pull Request

## Development Environment

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for full setup instructions.

Quick version:

```bash
make docker-up          # Start PostgreSQL
cd services/api && go run .  # Start API
make test               # Run all tests
```

## Code Style

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- Run `goimports -w .` before committing
- Run `go vet ./...` and `golangci-lint run` before pushing
- All exported functions must have doc comments
- No unused imports or variables

### Rust

- Follow standard `rustfmt` formatting
- Run `cargo fmt --check` before committing
- Run `cargo clippy -- -D warnings` before pushing
- All public items must have doc comments (`///`)
- No `unwrap()` in production code — use `?` or explicit error handling

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new policy rule type for velocity checks
fix: correct Merkle tree leaf ordering for odd counts
docs: update API reference for new endpoints
test: add benchmark for ZK proof generation
refactor: extract rate limiter into separate middleware
chore: update dependencies
```

## Pull Request Guidelines

### Before Opening a PR

- [ ] All tests pass (`make test`)
- [ ] `go vet` and `cargo clippy` clean
- [ ] `goimports` / `cargo fmt` applied
- [ ] New code has tests
- [ ] Documentation updated if needed
- [ ] CHANGELOG.md updated with entry

### PR Description Template

```markdown
## What

Brief description of the change.

## Why

Why this change is needed.

## How

Implementation details (if non-obvious).

## Testing

How this was tested.

## Checklist

- [ ] Tests pass
- [ ] Docs updated
- [ ] CHANGELOG updated
- [ ] No breaking API changes (or documented)
```

## Invariants

All code must respect the 10 system invariants in [docs/invariants.md](docs/invariants.md). PRs that violate an invariant will be rejected.

Key invariants:
- **I1**: No transfer without valid, approved intent
- **I2**: No intent approval without valid policy check
- **I3**: No on-chain transaction without MPC threshold signature
- **I5**: Replay protection via nonce + intent hash
- **I9**: Every state transition must be audit-logged

## Testing Requirements

### Unit Tests

- All new functions must have unit tests
- Mock external dependencies (Solana RPC, DB) using interfaces
- Test both success and error paths

### Integration Tests

- Test full lifecycle flows
- Test error recovery paths
- Test concurrent access patterns

### Benchmarks

- New crypto operations must have benchmarks
- Performance-sensitive code must show no regression

## Security

### Reporting Vulnerabilities

**Do NOT open a public issue for security vulnerabilities.**

Email security@vaultforge.io with:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Security Review Checklist

- [ ] No secrets in code or commits
- [ ] All DB queries parameterized (no string interpolation)
- [ ] Input validated before use
- [ ] Error messages don't leak internal state
- [ ] Rate limiting applied to all endpoints
- [ ] Auth checks on all protected routes

## Architecture Decisions

Significant changes require an Architecture Decision Record (ADR). See [docs/adr/](docs/adr/) for existing ADRs.

To create a new ADR:

```bash
cp docs/adr/000-template.md docs/adr/NNN-title.md
# Fill in the template
```

## Release Process

1. Update CHANGELOG.md with all changes
2. Bump version in Cargo.toml files and Chart.yaml
3. Create a git tag: `git tag v0.X.0`
4. Push: `git push origin v0.X.0`
5. CI builds and pushes Docker image
6. Deploy to staging, then production

## Code of Conduct

- Be respectful and constructive
- Focus on the code, not the person
- Welcome newcomers and help them get started
- Disagreements are fine; personal attacks are not
