FROM golang:1.25.9 AS builder
WORKDIR /app
COPY go.work go.work.sum ./
RUN mkdir -p backend e2e oracle explorer-indexer explorer-api && \
    echo "module github.com/sovereign-l1/backend" > backend/go.mod && \
    echo "module github.com/sovereign-l1/e2e" > e2e/go.mod && \
    echo "module github.com/sovereign-l1/oracle" > oracle/go.mod && \
    echo "module github.com/sovereign-l1/explorer-indexer" > explorer-indexer/go.mod && \
    echo "module github.com/sovereign-l1/explorer-api" > explorer-api/go.mod
COPY relayer ./relayer
COPY chain ./chain
RUN cd relayer && go get github.com/nats-io/nats.go@v1.15.0 && CGO_ENABLED=1 go build -o /app/bin/relayer ./cmd/relayer

FROM debian:trixie-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/bin/relayer /usr/local/bin/relayer
EXPOSE 9300
ENTRYPOINT ["/usr/local/bin/relayer"]
