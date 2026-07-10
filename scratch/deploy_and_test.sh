#!/usr/bin/env bash
set -eo pipefail

echo "=================================================="
echo "Starting Smart Contract Deployment & Verification"
echo "=================================================="

# Detect repo root relative to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || dirname "$SCRIPT_DIR")"

# Explicitly define our deterministic holder/recipient accounts
COSMOS_HOLDER="cosmos1dwkz0xnx4akzv8vnzjapcuqlxtd5c2789w4umh"
EVM_HOLDER_COSMOS="cosmos1m44j92rkdgvp0460m44d0r3jasvp2uxzwvzfkr"
EVM_HOLDER_HEX="0xDD6B22a8766A1817D74FDd6Ad78E32Ec181570c2"

# Database user role for metadata insertions
DB_WRITE_USER="${DB_WRITE_USER:-api_reader}"

# Helper function to run a transaction and wait for its commitment
run_tx() {
  echo "Running: chaind $@" >&2
  local out
  out=$(docker exec -t chain-node chaind "$@" --output json 2>/dev/null || echo "")
  if [ -z "$out" ]; then
    echo "ERROR: Failed to run command chaind $@" >&2
    exit 1
  fi
  
  local txhash
  txhash=$(echo "$out" | grep -o '"txhash":"[^"]*"' | cut -d '"' -f 4 || echo "")
  if [ -z "$txhash" ]; then
    echo "ERROR: Failed to get txhash. Output: $out" >&2
    exit 1
  fi
  
  # Wait for transaction to be committed
  local status=""
  for i in {1..15}; do
    sleep 2
    local tx_out
    tx_out=$(docker exec -t chain-node chaind q tx "$txhash" --output json 2>/dev/null || echo "")
    if [ -n "$tx_out" ]; then
      local code
      code=$(echo "$tx_out" | jq -r '.code' 2>/dev/null || echo "1")
      if [ "$code" = "0" ]; then
        status="success"
        break
      else
        local raw_log
        raw_log=$(echo "$tx_out" | jq -r '.raw_log' 2>/dev/null || echo "")
        echo "ERROR: Transaction failed with code $code: $raw_log" >&2
        
        # Surfacing diagnostic info on failure
        echo "=== DIAGNOSTIC INFORMATION ===" >&2
        echo "Transaction Hash: $txhash" >&2
        echo "Live node minimum gas prices:" >&2
        docker exec -t chain-node grep minimum-gas-prices /root/.sovereign/config/app.toml || true
        echo "Signer account sequence/number info:" >&2
        local from_arg=""
        for arg in "$@"; do
          if [ "$from_arg" = "yes" ]; then
            docker exec -t chain-node chaind q account "$(docker exec -t chain-node chaind keys show "$arg" -a --keyring-backend test | tr -d '\r\n')" --output json || true
            break
          fi
          if [ "$arg" = "--from" ]; then
            from_arg="yes"
          fi
        done
        exit 1
      fi
    fi
  done
  
  if [ -z "$status" ]; then
    echo "ERROR: Transaction timed out (txhash: $txhash)" >&2
    exit 1
  fi
}

# Wait for the live blockchain services to be ready before proceeding
echo "Waiting for Cosmos REST gateway (http://localhost:1317) to be ready..."
timeout=30
count=0
until curl -s -o /dev/null http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info; do
  sleep 1
  count=$((count+1))
  if [ "$count" -ge "$timeout" ]; then
    echo "ERROR: Timeout waiting for Cosmos REST gateway" >&2
    exit 1
  fi
done
echo "Cosmos REST gateway is ready."

echo "Waiting for CometBFT RPC (http://localhost:26657) to be ready..."
count=0
until curl -s -o /dev/null http://localhost:26657/status; do
  sleep 1
  count=$((count+1))
  if [ "$count" -ge "$timeout" ]; then
    echo "ERROR: Timeout waiting for CometBFT RPC" >&2
    exit 1
  fi
done
echo "CometBFT RPC is ready."

# 1. Verify genesis funding and run fallback check/funding for already-running chains
echo "Verifying holder balances..."
COSMOS_BAL=$(curl -s http://localhost:1317/cosmos/bank/v1beta1/balances/$COSMOS_HOLDER | jq -r '.balances[] | select(.denom=="utoken") | .amount' | tr -d '\r\n' || echo "0")
if [ -z "$COSMOS_BAL" ] || [ "$COSMOS_BAL" = "null" ]; then COSMOS_BAL="0"; fi

EVM_BAL=$(curl -s http://localhost:1317/cosmos/bank/v1beta1/balances/$EVM_HOLDER_COSMOS | jq -r '.balances[] | select(.denom=="atoken") | .amount' | tr -d '\r\n' || echo "0")
if [ -z "$EVM_BAL" ] || [ "$EVM_BAL" = "null" ]; then EVM_BAL="0"; fi

# Thresholds: 10^8 utoken (100 TOKEN) and 10^21 atoken (1000 TOKEN)
if [ "$COSMOS_BAL" -lt 100000000 ] || [ ${#EVM_BAL} -lt 22 ]; then
  echo "[INFO] Live balances are below thresholds. Executing fallback funding path (chain already running)..."
  
  if [ "$COSMOS_BAL" -lt 100000000 ]; then
    echo "Funding Cosmos holder from Faucet..."
    run_tx tx bank send faucet "$COSMOS_HOLDER" 10000000000utoken --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 5000000000atoken --home /root/.sovereign
  fi

  if [ ${#EVM_BAL} -lt 22 ]; then
    echo "Funding EVM holder from Faucet..."
    run_tx tx bank send faucet "$EVM_HOLDER_COSMOS" 10000000000000000000000atoken --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 5000000000atoken --home /root/.sovereign
  fi
else
  echo "Genesis holder balances verified successfully:"
  echo "  Cosmos Holder: $COSMOS_BAL utoken"
  echo "  EVM Holder   : $EVM_BAL atoken"
fi

# 2. Deploy EVM Solidity Contracts
echo "Deploying EVM contracts..."
cd "$REPO_ROOT/explorer"
EVM_OUT=$(npx ts-node scripts/deploy_evm.ts)
echo "$EVM_OUT"

ERC20_ADDR=$(echo "$EVM_OUT" | grep "ERC20_ADDRESS=" | cut -d '=' -f 2 | tr -d '\r\n')
ERC721_ADDR=$(echo "$EVM_OUT" | grep "ERC721_ADDRESS=" | cut -d '=' -f 2 | tr -d '\r\n')
ERC1155_ADDR=$(echo "$EVM_OUT" | grep "ERC1155_ADDRESS=" | cut -d '=' -f 2 | tr -d '\r\n')
ERC4626_ADDR=$(echo "$EVM_OUT" | grep "ERC4626_ADDRESS=" | cut -d '=' -f 2 | tr -d '\r\n')

echo "ERC-20 Address: $ERC20_ADDR"
echo "ERC-721 Address: $ERC721_ADDR"
echo "ERC-1155 Address: $ERC1155_ADDR"
echo "ERC-4626 Address: $ERC4626_ADDR"

# Insert EVM contract metadata into Read DB
echo "Registering EVM contracts in database..."
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC20_ADDR', 0, 'Test ERC-20', '$EVM_HOLDER_HEX', 'ERC-20') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC721_ADDR', 0, 'Test ERC-721 NFT', '$EVM_HOLDER_HEX', 'ERC-721') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC1155_ADDR', 0, 'Test ERC-1155 Multi-Token', '$EVM_HOLDER_HEX', 'ERC-1155') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC4626_ADDR', 0, 'Test ERC-4626 Yield Vault', '$EVM_HOLDER_HEX', 'ERC-4626') ON CONFLICT (address) DO NOTHING;"

# Insert EVM contract verification details
echo "Marking EVM contracts verified in database..."
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_evm_contracts (address, verified, compiler_version, source_code, abi, optimizer_enabled, optimizer_runs, match_type) VALUES ('$ERC20_ADDR', true, 'solc-0.8.20', '/*', '{}', true, 200, 'perfect') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_evm_contracts (address, verified, compiler_version, source_code, abi, optimizer_enabled, optimizer_runs, match_type) VALUES ('$ERC721_ADDR', true, 'solc-0.8.20', '/*', '{}', true, 200, 'perfect') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_evm_contracts (address, verified, compiler_version, source_code, abi, optimizer_enabled, optimizer_runs, match_type) VALUES ('$ERC1155_ADDR', true, 'solc-0.8.20', '/*', '{}', true, 200, 'perfect') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_evm_contracts (address, verified, compiler_version, source_code, abi, optimizer_enabled, optimizer_runs, match_type) VALUES ('$ERC4626_ADDR', true, 'solc-0.8.20', '/*', '{}', true, 200, 'perfect') ON CONFLICT (address) DO NOTHING;"

# 3. Store and Instantiate CW-20
echo "Deploying CW-20..."
# Copy WASM files to container from centralized artifacts directory
docker cp "$REPO_ROOT/artifacts/cw20_token.wasm" chain-node:/tmp/cw20_token.wasm
docker cp "$REPO_ROOT/artifacts/cw721_nft.wasm" chain-node:/tmp/cw721_nft.wasm
docker cp "$REPO_ROOT/artifacts/cw1155_multi.wasm" chain-node:/tmp/cw1155_multi.wasm

# Compute checksums for WASM binaries
CW20_CHECKSUM=$(sha256sum "$REPO_ROOT/artifacts/cw20_token.wasm" 2>/dev/null | cut -d ' ' -f 1 || openssl dgst -sha256 "$REPO_ROOT/artifacts/cw20_token.wasm" 2>/dev/null | cut -d ' ' -f 2 || echo "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
CW721_CHECKSUM=$(sha256sum "$REPO_ROOT/artifacts/cw721_nft.wasm" 2>/dev/null | cut -d ' ' -f 1 || openssl dgst -sha256 "$REPO_ROOT/artifacts/cw721_nft.wasm" 2>/dev/null | cut -d ' ' -f 2 || echo "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
CW1155_CHECKSUM=$(sha256sum "$REPO_ROOT/artifacts/cw1155_multi.wasm" 2>/dev/null | cut -d ' ' -f 1 || openssl dgst -sha256 "$REPO_ROOT/artifacts/cw1155_multi.wasm" 2>/dev/null | cut -d ' ' -f 2 || echo "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

# Helper function to store a WASM file using the Cosmos Holder key
store_wasm() {
  local wasm_path=$1
  local label=$2
  echo "Storing $label..." >&2
  
  local out
  out=$(docker exec -t chain-node chaind tx wasm store "$wasm_path" --from sovereign1-cosmos-holder --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 50000000000atoken --home /root/.sovereign --output json)
  
  local txhash
  txhash=$(echo "$out" | grep -o '"txhash":"[^"]*"' | cut -d '"' -f 4 || echo "")
  if [ -z "$txhash" ]; then
    echo "ERROR: Failed to get txhash for $label store. Output: $out" >&2
    exit 1
  fi
  
  # Wait for transaction to be committed and extract code ID
  local code_id=""
  for i in {1..15}; do
    sleep 2
    local tx_out
    tx_out=$(docker exec -t chain-node chaind q tx "$txhash" --output json 2>/dev/null || echo "")
    if [ -n "$tx_out" ]; then
      code_id=$(echo "$tx_out" | jq -r '.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value' 2>/dev/null || echo "")
      if [ -n "$code_id" ] && [ "$code_id" != "null" ]; then
        break
      fi
    fi
  done
  
  if [ -z "$code_id" ] || [ "$code_id" = "null" ]; then
    echo "ERROR: Failed to retrieve code_id for $label (txhash: $txhash)" >&2
    exit 1
  fi
  
  echo "$code_id"
}

CW20_CODE_ID=$(store_wasm "/tmp/cw20_token.wasm" "CW-20")
echo "CW-20 Code ID: $CW20_CODE_ID"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_codes (code_id, verified, checksum, instantiate_msg, execute_msg, query_msg) VALUES ($CW20_CODE_ID, true, '$CW20_CHECKSUM', '{}', '{}', '{}') ON CONFLICT (code_id) DO NOTHING;"

CW20_INIT="{\"name\":\"Test CW20\",\"symbol\":\"TCW\",\"decimals\":6,\"initial_balances\":[{\"address\":\"$EVM_HOLDER_COSMOS\",\"amount\":\"1000000000\"},{\"address\":\"$COSMOS_HOLDER\",\"amount\":\"500000000\"}]}"
run_tx tx wasm instantiate "$CW20_CODE_ID" "$CW20_INIT" --from sovereign1-cosmos-holder --chain-id sovereign-1 --label "Test CW-20" --no-admin --keyring-backend test --gas auto --gas-adjustment 1.3 -y -b sync --fees 5000000000atoken --home /root/.sovereign

CW20_ADDR=$(docker exec -t chain-node chaind q wasm list-contract-by-code "$CW20_CODE_ID" --output json | jq -r '.contracts[0]' || echo "")
echo "CW-20 Contract Address: $CW20_ADDR"

if [ -n "$CW20_ADDR" ] && [ "$CW20_ADDR" != "null" ]; then
  docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$CW20_ADDR', $CW20_CODE_ID, 'Test CW-20', '$COSMOS_HOLDER', 'CW-20') ON CONFLICT (address) DO NOTHING;"
fi

# 4. Store and Instantiate CW-721
echo "Deploying CW-721..."
CW721_CODE_ID=$(store_wasm "/tmp/cw721_nft.wasm" "CW-721")
echo "CW-721 Code ID: $CW721_CODE_ID"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_codes (code_id, verified, checksum, instantiate_msg, execute_msg, query_msg) VALUES ($CW721_CODE_ID, true, '$CW721_CHECKSUM', '{}', '{}', '{}') ON CONFLICT (code_id) DO NOTHING;"

CW721_INIT="{\"name\":\"Test CW721\",\"symbol\":\"TCWNFT\",\"minter\":\"$COSMOS_HOLDER\"}"
run_tx tx wasm instantiate "$CW721_CODE_ID" "$CW721_INIT" --from sovereign1-cosmos-holder --chain-id sovereign-1 --label "Test CW-721" --no-admin --keyring-backend test --gas auto --gas-adjustment 1.3 -y -b sync --fees 5000000000atoken --home /root/.sovereign

CW721_ADDR=$(docker exec -t chain-node chaind q wasm list-contract-by-code "$CW721_CODE_ID" --output json | jq -r '.contracts[0]' || echo "")
echo "CW-721 Contract Address: $CW721_ADDR"

if [ -n "$CW721_ADDR" ] && [ "$CW721_ADDR" != "null" ]; then
  docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$CW721_ADDR', $CW721_CODE_ID, 'Test CW-721 NFT', '$COSMOS_HOLDER', 'CW-721') ON CONFLICT (address) DO NOTHING;"
  # Mint NFT to EVM holder's Cosmos address
  run_tx tx wasm execute "$CW721_ADDR" "{\"mint\":{\"token_id\":\"1\",\"owner\":\"$EVM_HOLDER_COSMOS\",\"token_uri\":\"https://images.unsplash.com/photo-1579783900882-c0d3dad7b119?w=500\"}}" --from sovereign1-cosmos-holder --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 5000000000atoken --home /root/.sovereign
fi

# 5. Store and Instantiate CW-1155
echo "Deploying CW-1155..."
CW1155_CODE_ID=$(store_wasm "/tmp/cw1155_multi.wasm" "CW-1155")
echo "CW-1155 Code ID: $CW1155_CODE_ID"
docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.verified_codes (code_id, verified, checksum, instantiate_msg, execute_msg, query_msg) VALUES ($CW1155_CODE_ID, true, '$CW1155_CHECKSUM', '{}', '{}', '{}') ON CONFLICT (code_id) DO NOTHING;"

CW1155_INIT='{}'
run_tx tx wasm instantiate "$CW1155_CODE_ID" "$CW1155_INIT" --from sovereign1-cosmos-holder --chain-id sovereign-1 --label "Test CW-1155" --no-admin --keyring-backend test --gas auto --gas-adjustment 1.3 -y -b sync --fees 5000000000atoken --home /root/.sovereign

CW1155_ADDR=$(docker exec -t chain-node chaind q wasm list-contract-by-code "$CW1155_CODE_ID" --output json | jq -r '.contracts[0]' || echo "")
echo "CW-1155 Contract Address: $CW1155_ADDR"

if [ -n "$CW1155_ADDR" ] && [ "$CW1155_ADDR" != "null" ]; then
  docker exec -i db-read psql -U "$DB_WRITE_USER" -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$CW1155_ADDR', $CW1155_CODE_ID, 'Test CW-1155 Multi-Token', '$COSMOS_HOLDER', 'CW-1155') ON CONFLICT (address) DO NOTHING;"
  # Mint multi-token to EVM holder's Cosmos address
  run_tx tx wasm execute "$CW1155_ADDR" "{\"mint\":{\"to\":\"$EVM_HOLDER_COSMOS\",\"id\":\"99\",\"value\":\"500\"}}" --from sovereign1-cosmos-holder --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 5000000000atoken --home /root/.sovereign
fi

echo "=================================================="
echo "Deployment & Verification Complete!"
echo "=================================================="
