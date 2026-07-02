FROM golang:alpine AS builder
WORKDIR /app
ENV GOWORK=off
COPY chain/go.mod chain/go.sum ./chain/
COPY explorer-indexer/go.mod explorer-indexer/go.sum ./explorer-indexer/
COPY chain ./chain
COPY explorer-indexer ./explorer-indexer
RUN cd explorer-indexer && go build -o /app/indexer main.go

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/indexer /app/indexer
ENTRYPOINT ["/app/indexer"]
