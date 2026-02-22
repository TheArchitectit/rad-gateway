# A2A Wasm Filter - Build Automation
.PHONY: build test clean image push lint check

FILTER_NAME := a2a-wasm-filter
REGISTRY := ghcr.io/radgateway
TAG := latest
IMAGE := $(REGISTRY)/$(FILTER_NAME):$(TAG)

# Build the Wasm module
build:
	cargo build --target wasm32-unknown-unknown --release
	@echo "Wasm module built successfully"
	@ls -lh target/wasm32-unknown-unknown/release/*.wasm

# Run tests
test:
	cargo test --target wasm32-unknown-unknown

# Lint and format
lint:
	cargo clippy --target wasm32-unknown-unknown -- -D warnings
	cargo fmt -- --check

# Clean build artifacts
clean:
	cargo clean
	rm -rf dist/

# Build Docker image
image: build
	docker build -t $(IMAGE) .
	@echo "Docker image built: $(IMAGE)"

# Push to registry
push: image
	docker push $(IMAGE)
	@echo "Docker image pushed: $(IMAGE)"

# Full build pipeline
all: lint test build

# Development: fast build without optimizations
dev:
	cargo build --target wasm32-unknown-unknown

# Check compilation only
check:
	cargo check --target wasm32-unknown-unknown
