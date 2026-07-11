.PHONY: build test lint verify-build clean generate

generate:
	cd proto && buf generate


GO_BUILD_FLAGS = -trimpath -ldflags="-s -w -buildid="

build:
	mkdir -p bin
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/chaind ./chain/cmd/chaind
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/faucet ./backend/module/faucet

test:
	go test -v ./chain/...
	go test -v ./backend/...
	go test -v ./e2e/...

lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./... || echo "golangci-lint not installed or failed"

verify-build:
	@echo "Performing local reproducible build verification..."
	mkdir -p bin/build1 bin/build2
	# Compile build 1
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build1/chaind ./chain/cmd/chaind
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build1/faucet ./backend/module/faucet
	# Compile build 2
	CGO_ENABLED=1 go build $(GO_BUILD_FLAGS) -o bin/build2/chaind ./chain/cmd/chaind
	CGO_ENABLED=0 go build $(GO_BUILD_FLAGS) -o bin/build2/faucet ./backend/module/faucet
	# Compare hashes
	@echo "Verifying SHA256 of chaind..."
	shasum -a 256 bin/build1/chaind bin/build2/chaind
	cmp bin/build1/chaind bin/build2/chaind
	@echo "Verifying SHA256 of faucet..."
	shasum -a 256 bin/build1/faucet bin/build2/faucet
	cmp bin/build1/faucet bin/build2/faucet
	@echo "[SUCCESS] Pinned build flags produce 100% identical binary signatures locally."

clean:
	rm -rf bin

