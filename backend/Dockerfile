FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off

# Copy go workspace config and module files
COPY go.work ./
COPY backend/go.mod backend/go.sum ./backend/
COPY chain/go.mod chain/go.sum ./chain/

# Download dependencies
RUN --mount=type=cache,target=/go/pkg/mod \
    cd backend && go mod download

# Copy source directories
COPY backend ./backend
COPY chain ./chain

# Build binaries
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd backend && \
    go build -o /app/bin/faucet ./module/faucet

FROM debian:trixie-slim
WORKDIR /app

# Install standard dependencies
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*

# Copy chaind and wasmvm libraries from the chain-node image
COPY --from=sovereign-chain-node:latest /usr/local/bin/chaind /usr/local/bin/chaind
COPY --from=sovereign-chain-node:latest /usr/lib/libwasmvm.* /usr/lib/
RUN ldconfig

# Copy compiled faucet binary
COPY --from=builder /app/bin/faucet /app/faucet

ENTRYPOINT ["/app/faucet"]
