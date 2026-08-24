.PHONY: build test lint docker deploy clean help

# Go API
GO_API_DIR := ./services/api
GO_BINARY := vaultforge-api

# Rust crates
CRATES := crypto mpc zk policy types transaction solana

# Solana program
PROGRAM_NAME := vault_policy

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Build ──────────────────────────────────────────────

build: build-go build-solana ## Build all binaries

build-go: ## Build Go API binary
	cd $(GO_API_DIR) && go build -o ../$(GO_BINARY) .
	@echo "Built: $(GO_BINARY)"

build-rust: ## Build all Rust crates
	@for crate in $(CRATES); do \
		echo "Building $$crate..."; \
		cd crates/$$crate && cargo build && cd ../..; \
	done

build-solana: ## Build Solana program
	anchor build
	@echo "Built: $(PROGRAM_NAME)"

# ── Test ───────────────────────────────────────────────

test: test-go test-rust ## Run all tests

test-go: ## Run Go tests
	cd $(GO_API_DIR) && go test -v -count=1 ./...

test-rust: ## Run all Rust crate tests
	@for crate in $(CRATES); do \
		echo "=== Testing $$crate ==="; \
		cd crates/$$crate && cargo test && cd ../..; \
	done

test-integration: ## Run integration tests
	./scripts/run-integration-tests.sh

test-all: test-go test-rust test-integration ## Run all tests including integration

test-race: ## Run Go tests with race detector
	cd $(GO_API_DIR) && go test -v -count=1 -race ./...

test-coverage: ## Run Go tests with coverage report
	cd $(GO_API_DIR) && go test -count=1 -coverprofile=coverage.out ./... && \
		go tool cover -func=coverage.out && \
		go tool cover -html=coverage.out -o coverage.html && \
		echo "Coverage report: coverage.html"

test-coverage-open: ## Run tests and open coverage report
	cd $(GO_API_DIR) && go test -count=1 -coverprofile=coverage.out ./... && \
		go tool cover -func=coverage.out && \
		go tool cover -html=coverage.out -o coverage.html && \
		open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || echo "Open coverage.html manually"

# ── Lint ───────────────────────────────────────────────

lint: lint-go lint-rust ## Run all linters

lint-go: ## Run Go linter
	cd $(GO_API_DIR) && go vet ./...

lint-rust: ## Run Rust clippy
	@for crate in $(CRATES); do \
		echo "=== Clippy $$crate ==="; \
		cd crates/$$crate && cargo clippy -- -D warnings && cd ../..; \
	done

fmt: fmt-go fmt-rust ## Format all code

fmt-go: ## Format Go code
	cd $(GO_API_DIR) && gofmt -s -w .

fmt-rust: ## Format Rust code
	@for crate in $(CRATES); do \
		cd crates/$$crate && cargo fmt && cd ../..; \
	done

# ── Docker ─────────────────────────────────────────────

docker-build: ## Build Docker image
	docker build -f docker/Dockerfile.api -t vaultforge-api:latest ..

docker-scan: docker-build ## Build and scan Docker image for vulnerabilities
	@echo "=== Docker Image Security Scan ==="
	@echo "Image: vaultforge-api:latest"
	@echo ""
	@command -v trivy >/dev/null 2>&1 && \
		trivy image --severity HIGH,CRITICAL vaultforge-api:latest || \
		echo "Trivy not installed. Install: brew install trivy"
	@echo ""
	@command -v grype >/dev/null 2>&1 && \
		grype vaultforge-api:latest || \
		echo "Grype not installed. Install: brew install grype"

docker-up: ## Start services with docker-compose
	cd docker && docker compose up -d

docker-down: ## Stop services with docker-compose
	cd docker && docker compose down

docker-logs: ## View docker-compose logs
	cd docker && docker compose logs -f

# ── Deploy ─────────────────────────────────────────────

deploy-devnet: build-solana ## Deploy program to Solana devnet
	./scripts/deploy-devnet.sh

create-wallets: ## Create test wallets for devnet
	./scripts/create-test-wallets.sh

e2e-devnet: ## Execute a real SOL transfer on devnet (builds tx with solana-go)
	./scripts/e2e-devnet.sh

# ── Operations ──────────────────────────────────────────

security-scan: ## Run security audit (cargo-audit + gosec + govulncheck)
	./scripts/security-scan.sh

security-scan-all: security-scan ## Run full security scan including Docker
	./scripts/security-scan.sh
	$(MAKE) docker-scan

verify: test-race security-scan ## Full verification (tests + security)
	@echo "=== Full Verification Complete ==="
	@echo "All tests passed with race detector."
	@echo "Security scan complete."

helm-template: ## Render Helm templates locally
	helm template vaultforge deploy/helm/vaultforge

helm-lint: ## Lint Helm chart
	helm lint deploy/helm/vaultforge

seed-db: ## Seed database with test data
	./scripts/seed-db.sh

backup-db: ## Backup database (full)
	./scripts/backup-db.sh /var/backups/vaultforge full

restart: ## Graceful restart (systemd)
	./scripts/restart.sh

restart-docker: ## Graceful restart (docker-compose)
	./scripts/restart.sh --docker

status: ## Show service status
	./scripts/status.sh

# ── Testing ────────────────────────────────────────────

load-test: ## Run load tests (requires: hey)
	./scripts/load-test.sh

integration-test: ## Run integration tests against devnet
	./scripts/integration-tests.sh

bench: ## Run Rust benchmarks (requires nightly)
	@for crate in crypto zk; do \
		echo "=== Benchmarks: $$crate ==="; \
		cd crates/$$crate && cargo bench 2>&1; \
		cd ../..; \
	done

# ── Clean ──────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -f $(GO_BINARY)
	@for crate in $(CRATES); do \
		cd crates/$$crate && cargo clean && cd ../..; \
	done
	anchor clean 2>/dev/null || true
	@echo "Cleaned all build artifacts"
