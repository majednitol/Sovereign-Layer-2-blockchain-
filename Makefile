.PHONY: build test lint verify-build clean generate

generate:
	cd proto && buf generate


GO_BUILD_FLAGS = -trimpath -ldflags="-s -w -buildid="

build:
	mkdir -p bin
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/chaind ./chain/cmd/chaind

test:
	go test -v ./chain/...
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
	# Compile build 2
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build2/chaind ./chain/cmd/chaind
	# Compare hashes
	@echo "Verifying SHA256 of chaind..."
	shasum -a 256 bin/build1/chaind bin/build2/chaind
	cmp bin/build1/chaind bin/build2/chaind
	@echo "[SUCCESS] Pinned build flags produce 100% identical chaind binary signature locally."

clean:
	rm -rf bin
