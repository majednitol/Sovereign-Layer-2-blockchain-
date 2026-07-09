#!/usr/bin/env bash
set -eo pipefail

echo "=================================================="
echo "Starting Smart Contract Deployment & Verification"
echo "=================================================="

# Helper function to run a transaction and wait for its commitment
run_tx() {
  echo "Running: chaind $@" >&2
  local out
  out=$(docker exec -t chain-node chaind "$@" --output json)
  
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
        exit 1
      fi
    fi
  done
  
  if [ -z "$status" ]; then
    echo "ERROR: Transaction timed out (txhash: $txhash)" >&2
    exit 1
  fi
}

# 1. Fund the EVM signer address using chaind tx bank send
echo "Funding EVM faucet address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266..."
TARGET_BAL=$(curl -s http://localhost:1317/cosmos/bank/v1beta1/balances/cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq | jq -r '.balances[] | select(.denom=="atoken") | .amount' | tr -d '\r\n')
if [ -z "$TARGET_BAL" ] || [ "$TARGET_BAL" = "null" ]; then TARGET_BAL="0"; fi
# 10^21 atoken = 1000 tokens (length 22). If length is less than 22, fund it.
if [ ${#TARGET_BAL} -lt 22 ]; then
  run_tx tx bank send faucet cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq 10000000000000000000000atoken --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 10000atoken --home /root/.sovereign
else
  echo "EVM faucet balance is sufficient ($TARGET_BAL atoken). Skipping transfer."
fi

# 2. Deploy EVM Solidity Contracts
echo "Deploying EVM contracts..."
cd /Users/majedurrahman/Sovereign/explorer
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
docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC20_ADDR', 0, 'Test ERC-20', '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266', 'ERC-20') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC721_ADDR', 0, 'Test ERC-721 NFT', '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266', 'ERC-721') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC1155_ADDR', 0, 'Test ERC-1155 Multi-Token', '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266', 'ERC-1155') ON CONFLICT (address) DO NOTHING;"
docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$ERC4626_ADDR', 0, 'Test ERC-4626 Yield Vault', '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266', 'ERC-4626') ON CONFLICT (address) DO NOTHING;"

# 3. Store and Instantiate CW-20
echo "Deploying CW-20..."
# Copy WASM files to container
docker cp /Users/majedurrahman/Sovereign/contracts/target/wasm32-unknown-unknown/release/cw20_token.wasm chain-node:/tmp/cw20_token.wasm
docker cp /Users/majedurrahman/Sovereign/contracts/target/wasm32-unknown-unknown/release/cw721_nft.wasm chain-node:/tmp/cw721_nft.wasm
docker cp /Users/majedurrahman/Sovereign/contracts/target/wasm32-unknown-unknown/release/cw1155_multi.wasm chain-node:/tmp/cw1155_multi.wasm

# Helper function to store a WASM file and get its Code ID
store_wasm() {
  local wasm_path=$1
  local label=$2
  echo "Storing $label..." >&2
  
  local out
  out=$(docker exec -t chain-node chaind tx wasm store "$wasm_path" --from faucet --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 1000000atoken --home /root/.sovereign --output json)
  
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

CW20_INIT='{"name":"Test CW20","symbol":"TCW","decimals":6,"initial_balances":[{"address":"cosmos1u8c62zfj2je4p324znutnka5pwvkzuxyyk63dz","amount":"1000000000"},{"address":"cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq","amount":"500000000"}]}'
run_tx tx wasm instantiate "$CW20_CODE_ID" "$CW20_INIT" --from faucet --chain-id sovereign-1 --label "Test CW-20" --no-admin --keyring-backend test --gas auto --gas-adjustment 1.3 -y -b sync --fees 100000atoken --home /root/.sovereign

CW20_ADDR=$(docker exec -t chain-node chaind q wasm list-contract-by-code "$CW20_CODE_ID" --output json | jq -r '.contracts[0]' || echo "")
echo "CW-20 Contract Address: $CW20_ADDR"

if [ -n "$CW20_ADDR" ] && [ "$CW20_ADDR" != "null" ]; then
  docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$CW20_ADDR', $CW20_CODE_ID, 'Test CW-20', 'cosmos1srqfdy5fwu2ek25kswwvpj647dl30nwfg7gcf8', 'CW-20') ON CONFLICT (address) DO NOTHING;"
fi

# 4. Store and Instantiate CW-721
echo "Deploying CW-721..."
CW721_CODE_ID=$(store_wasm "/tmp/cw721_nft.wasm" "CW-721")
echo "CW-721 Code ID: $CW721_CODE_ID"

CW721_INIT='{"name":"Test CW721","symbol":"TCWNFT","minter":"cosmos1srqfdy5fwu2ek25kswwvpj647dl30nwfg7gcf8"}'
run_tx tx wasm instantiate "$CW721_CODE_ID" "$CW721_INIT" --from faucet --chain-id sovereign-1 --label "Test CW-721" --no-admin --keyring-backend test --gas auto --gas-adjustment 1.3 -y -b sync --fees 100000atoken --home /root/.sovereign

CW721_ADDR=$(docker exec -t chain-node chaind q wasm list-contract-by-code "$CW721_CODE_ID" --output json | jq -r '.contracts[0]' || echo "")
echo "CW-721 Contract Address: $CW721_ADDR"

if [ -n "$CW721_ADDR" ] && [ "$CW721_ADDR" != "null" ]; then
  docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$CW721_ADDR', $CW721_CODE_ID, 'Test CW-721 NFT', 'cosmos1srqfdy5fwu2ek25kswwvpj647dl30nwfg7gcf8', 'CW-721') ON CONFLICT (address) DO NOTHING;"
  # Mint NFT to test address
  run_tx tx wasm execute "$CW721_ADDR" '{"mint":{"token_id":"1","owner":"cosmos1u8c62zfj2je4p324znutnka5pwvkzuxyyk63dz","token_uri":"https://images.unsplash.com/photo-1579783900882-c0d3dad7b119?w=500"}}' --from faucet --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 10000atoken --home /root/.sovereign
fi

# 5. Store and Instantiate CW-1155
echo "Deploying CW-1155..."
CW1155_CODE_ID=$(store_wasm "/tmp/cw1155_multi.wasm" "CW-1155")
echo "CW-1155 Code ID: $CW1155_CODE_ID"

CW1155_INIT='{}'
run_tx tx wasm instantiate "$CW1155_CODE_ID" "$CW1155_INIT" --from faucet --chain-id sovereign-1 --label "Test CW-1155" --no-admin --keyring-backend test --gas auto --gas-adjustment 1.3 -y -b sync --fees 100000atoken --home /root/.sovereign

CW1155_ADDR=$(docker exec -t chain-node chaind q wasm list-contract-by-code "$CW1155_CODE_ID" --output json | jq -r '.contracts[0]' || echo "")
echo "CW-1155 Contract Address: $CW1155_ADDR"

if [ -n "$CW1155_ADDR" ] && [ "$CW1155_ADDR" != "null" ]; then
  docker exec -i db-read psql -U api_reader -d sovereign_read -c "INSERT INTO explorer.contracts (address, code_id, label, creator, type_badge) VALUES ('$CW1155_ADDR', $CW1155_CODE_ID, 'Test CW-1155 Multi-Token', 'cosmos1srqfdy5fwu2ek25kswwvpj647dl30nwfg7gcf8', 'CW-1155') ON CONFLICT (address) DO NOTHING;"
  # Mint multi-token to test address
  run_tx tx wasm execute "$CW1155_ADDR" '{"mint":{"to":"cosmos1u8c62zfj2je4p324znutnka5pwvkzuxyyk63dz","id":"99","value":"500"}}' --from faucet --keyring-backend test --gas auto --gas-adjustment 1.3 --chain-id sovereign-1 -y -b sync --fees 10000atoken --home /root/.sovereign
fi

echo "=================================================="
echo "Deployment & Verification Complete!"
echo "=================================================="
