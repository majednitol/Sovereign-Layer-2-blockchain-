FROM golang:1.25.9 AS builder
WORKDIR /app
COPY go.work go.work.sum ./
RUN mkdir -p backend e2e relayer explorer-indexer explorer-api && \
    echo "module github.com/sovereign-l1/backend" > backend/go.mod && \
    echo "module github.com/sovereign-l1/e2e" > e2e/go.mod && \
    echo "module github.com/sovereign-l1/relayer" > relayer/go.mod && \
    echo "module github.com/sovereign-l1/explorer-indexer" > explorer-indexer/go.mod && \
    echo "module github.com/sovereign-l1/explorer-api" > explorer-api/go.mod
COPY oracle ./oracle
COPY chain ./chain
RUN cd oracle && CGO_ENABLED=1 go build -o /app/bin/oracle .

FROM debian:trixie-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/bin/oracle /usr/local/bin/oracle
EXPOSE 9200
ENTRYPOINT ["/usr/local/bin/oracle"]
