FROM --platform=linux/amd64 golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY chain/go.mod chain/go.sum ./chain/
COPY explorer-api/go.mod explorer-api/go.sum ./explorer-api/
RUN cd explorer-api && go mod download
COPY chain ./chain
COPY explorer-api ./explorer-api
RUN cd explorer-api && go build -o /app/api main.go

FROM --platform=linux/amd64 debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates wget && rm -rf /var/lib/apt/lists/*
RUN wget -O /usr/local/bin/solc https://binaries.soliditylang.org/linux-amd64/solc-linux-amd64-v0.8.24+commit.e11b9ed9 && \
    chmod +x /usr/local/bin/solc
COPY --from=builder /app/api /app/api
ENTRYPOINT ["/app/api"]
