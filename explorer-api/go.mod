module github.com/sovereign-l1/explorer-api

go 1.25.9

replace github.com/sovereign-l1/chain => ../chain

require (
	github.com/cosmos/cosmos-sdk v0.54.3
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0
	github.com/jackc/pgx/v5 v5.7.5
	github.com/nats-io/nats.go v1.52.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/sovereign-l1/chain v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.82.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/nats-io/nkeys v0.4.15 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260511170946-3700d4141b60 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260511170946-3700d4141b60 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
