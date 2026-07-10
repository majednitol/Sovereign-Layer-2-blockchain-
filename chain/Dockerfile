# --- Build Stage ---
# Pinned for production verification:
# golang:1.25.9@sha256:d8c558f623b3f2df9547d25e01eb4a86b5155f9a7adfb2cd33a2072f9dcfbe2a
# debian:trixie-slim@sha256:1f7a6b8c946bc77b9492167d7162a8cb9815ea7e31b67272828b610c14b9a9f2
FROM golang:1.25.9 AS builder

WORKDIR /app

# Copy dependency files and codebase
COPY chain ./chain

WORKDIR /app/chain

# Compile chain daemon and stage wasmvm shared library
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go build -ldflags "-s -w" -o /bin/chaind ./cmd/chaind && \
    WASMVM_V3=$(go list -f '{{.Dir}}' github.com/CosmWasm/wasmvm/v3/internal/api 2>/dev/null || echo "failed_v3") && \
    WASMVM_V2=$(go list -f '{{.Dir}}' github.com/CosmWasm/wasmvm/v2/internal/api 2>/dev/null || echo "failed_v2") && \
    echo "WASMVM_V3 dir: $WASMVM_V3" && \
    echo "WASMVM_V2 dir: $WASMVM_V2" && \
    (cp ${WASMVM_V3}/libwasmvm.*.so /lib/ 2>/dev/null || cp ${WASMVM_V2}/libwasmvm.*.so /lib/ || true)

# --- Final Stage ---
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl sed && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/chaind /usr/local/bin/chaind
# Copy wasmvm shared libraries
COPY --from=builder /lib/libwasmvm.* /usr/lib/
RUN ldconfig

COPY chain/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

EXPOSE 26656 26657 9090 1317

ENTRYPOINT ["/entrypoint.sh"]

