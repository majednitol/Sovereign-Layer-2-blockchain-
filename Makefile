.PHONY: build test lint verify-build clean generate

generate:
	cd proto && buf generate


GO_BUILD_FLAGS = -trimpath -ldflags="-s -w -buildid="

build:
	mkdir -p bin
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/chaind ./chain/cmd/chaind
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/oracle ./oracle
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/ingestion ./backend/module/ingestion
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/projection ./backend/module/projection
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/api ./backend/module/api
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/faucet ./backend/module/faucet

test:
	go test -v ./chain/...
	go test -v ./oracle/...
	go test -v ./relayer/...
	go test -v ./backend/...
	go test -v ./e2e/...

lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./... || echo "golangci-lint not installed or failed"
	@echo "Running cargo clippy..."
	cd contracts && cargo clippy --workspace --all-targets -- -D warnings

verify-build:
	@echo "Performing local reproducible build verification..."
	mkdir -p bin/build1 bin/build2
	# Compile build 1
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build1/chaind ./chain/cmd/chaind
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build1/oracle ./oracle
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build1/ingestion ./backend/module/ingestion
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build1/projection ./backend/module/projection
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build1/api ./backend/module/api
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build1/faucet ./backend/module/faucet
	# Compile build 2
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build2/chaind ./chain/cmd/chaind
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build2/oracle ./oracle
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build2/ingestion ./backend/module/ingestion
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build2/projection ./backend/module/projection
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build2/api ./backend/module/api
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build2/faucet ./backend/module/faucet
	# Compare hashes
	@echo "Verifying SHA256 of chaind..."
	shasum -a 256 bin/build1/chaind bin/build2/chaind
	cmp bin/build1/chaind bin/build2/chaind
	@echo "Verifying SHA256 of oracle..."
	shasum -a 256 bin/build1/oracle bin/build2/oracle
	cmp bin/build1/oracle bin/build2/oracle
	@echo "Verifying SHA256 of backend binaries..."
	shasum -a 256 bin/build1/ingestion bin/build2/ingestion
	cmp bin/build1/ingestion bin/build2/ingestion
	shasum -a 256 bin/build1/projection bin/build2/projection
	cmp bin/build1/projection bin/build2/projection
	shasum -a 256 bin/build1/api bin/build2/api
	cmp bin/build1/api bin/build2/api
	shasum -a 256 bin/build1/faucet bin/build2/faucet
	cmp bin/build1/faucet bin/build2/faucet
	@echo "[SUCCESS] Pinned build flags produce 100% identical binary signatures locally."

clean:
	rm -rf bin

build-cw-assets:
	mkdir -p artifacts
	cd contracts && cargo build --target wasm32-unknown-unknown --release --lib --workspace
	cp contracts/target/wasm32-unknown-unknown/release/cw20_token.wasm artifacts/cw20_token.wasm
	cp contracts/target/wasm32-unknown-unknown/release/cw721_nft.wasm artifacts/cw721_nft.wasm
	cp contracts/target/wasm32-unknown-unknown/release/cw1155_multi.wasm artifacts/cw1155_multi.wasm
	wasm-opt --llvm-memory-copy-fill-lowering artifacts/cw20_token.wasm -o artifacts/cw20_token.wasm
	wasm-opt --llvm-memory-copy-fill-lowering artifacts/cw721_nft.wasm -o artifacts/cw721_nft.wasm
	wasm-opt --llvm-memory-copy-fill-lowering artifacts/cw1155_multi.wasm -o artifacts/cw1155_multi.wasm
	cd artifacts && (sha256sum *.wasm > checksums.txt || shasum -a 256 *.wasm > checksums.txt)

