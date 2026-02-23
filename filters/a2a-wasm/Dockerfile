# A2A Wasm Filter - Multi-stage Build
# Stage 1: Build the Wasm module
FROM rust:1.75-slim-bookworm as builder

# Install required dependencies
RUN apt-get update && apt-get install -y \
    llvm-15-dev \
    libclang-15-dev \
    clang-15 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build

# Copy project files
COPY Cargo.toml rust-toolchain.toml ./
COPY .cargo/ ./.cargo/

# Fetch dependencies (cache layer)
RUN cargo fetch

# Copy source code
COPY src/ ./src/

# Build the Wasm module
RUN cargo build --target wasm32-unknown-unknown --release

# Verify the build
RUN ls -la /build/target/wasm32-unknown-unknown/release/*.wasm

# Stage 2: Just the Wasm file
FROM scratch
COPY --from=builder /build/target/wasm32-unknown-unknown/release/a2a_wasm_filter.wasm /filter.wasm
