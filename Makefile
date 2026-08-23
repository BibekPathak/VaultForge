.PHONY: build test lint docker clean help

# Go API
GO_API_DIR := ./services/api
GO_BINARY := vaultforge-api

# Rust crates
CRATES := crypto mpc zk policy types transaction solana

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ── Build ──────────────────────────────────────────────

build: build-go ## Build all binaries

build-go: ## Build Go API binary
	cd $(GO_API_DIR) && go build -o ../$(GO_BINARY) .
	@echo "Built: $(GO_BINARY)"

build-rust: ## Build all Rust crates
	@for crate in $(CRATES); do \
		echo "Building $$crate..."; \
		cd crates/$$crate && cargo build && cd ../..; \
	done

# ── Test ───────────────────────────────────────────────

test: test-go test-rust ## Run all tests

test-go: ## Run Go tests
	cd $(GO_API_DIR) && go test -v -count=1 ./...

test-rust: ## Run all Rust crate tests
	@for crate in $(CRATES); do \
		echo "=== Testing $$crate ==="; \
		cd crates/$$crate && cargo test && cd ../..; \
	done

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

docker-up: ## Start services with docker-compose
	cd docker && docker compose up -d

docker-down: ## Stop services with docker-compose
	cd docker && docker compose down

docker-logs: ## View docker-compose logs
	cd docker && docker compose logs -f

# ── Clean ──────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -f $(GO_BINARY)
	@for crate in $(CRATES); do \
		cd crates/$$crate && cargo clean && cd ../..; \
	done
	@echo "Cleaned all build artifacts"
