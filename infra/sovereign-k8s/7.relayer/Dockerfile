FROM golang:1.25.9-alpine AS builder
WORKDIR /app
COPY go.work ./
COPY relayer ./relayer
COPY chain ./chain
RUN cd relayer && go build -o /app/bin/relayer ./cmd/relayer

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/bin/relayer /usr/local/bin/relayer
EXPOSE 9300
ENTRYPOINT ["/usr/local/bin/relayer"]
