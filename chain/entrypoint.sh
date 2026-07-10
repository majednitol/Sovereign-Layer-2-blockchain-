#!/bin/sh
set -eu

CHAIN_HOME="${CHAIN_HOME:-/root/.sovereign}"
CHAIN_ID="${CHAIN_ID:-sovereign-testnet-1}"
MONIKER="${MONIKER:-sovereign-node}"

# Always initialize fresh for devnet (volume-mounted genesis.json overrides)
if [ ! -f "${CHAIN_HOME}/config/config.toml" ]; then
  echo "Initializing chain home at ${CHAIN_HOME}..."
  chaind init "${MONIKER}" --chain-id "${CHAIN_ID}" --home "${CHAIN_HOME}"

  # Copy custom genesis.json if mounted
  if [ -f /root/genesis.json ]; then
    echo "Overwriting default genesis with custom /root/genesis.json..."
    cp /root/genesis.json "${CHAIN_HOME}/config/genesis.json"
    # Re-read chain_id from custom genesis so the gentx is signed with the correct chain-id
    CUSTOM_CHAIN_ID=$(grep '"chain_id"' "${CHAIN_HOME}/config/genesis.json" | head -n 1 | cut -d '"' -f 4)
    if [ -n "${CUSTOM_CHAIN_ID}" ]; then
      echo "Detected chain_id from custom genesis: ${CUSTOM_CHAIN_ID}"
      CHAIN_ID="${CUSTOM_CHAIN_ID}"
    fi
  fi

  echo "Adding validator credentials to test keyring..."
  # Add validator key to test keyring
  chaind keys add validator --keyring-backend test --home "${CHAIN_HOME}"
  VAL_ADDR=$(chaind keys show validator -a --keyring-backend test --home "${CHAIN_HOME}")
  echo "Validator address: ${VAL_ADDR}"

  echo "Adding faucet credentials to test keyring..."
  echo "test test test test test test test test test test test junk" | chaind keys add faucet --recover --keyring-backend test --home "${CHAIN_HOME}"
  FAUCET_ADDR=$(chaind keys show faucet -a --keyring-backend test --home "${CHAIN_HOME}")
  echo "Faucet address: ${FAUCET_ADDR}"

  echo "Adding evm-holder credentials to test keyring..."
  echo "must motion super wedding record raccoon toast public dance dial index embrace" | chaind keys add sovereign1-evm-holder --recover --keyring-backend test --algo eth_secp256k1 --home "${CHAIN_HOME}"
  EVM_HOLDER_ADDR=$(chaind keys show sovereign1-evm-holder -a --keyring-backend test --home "${CHAIN_HOME}")
  echo "EVM holder address: ${EVM_HOLDER_ADDR}"

  echo "Adding cosmos-holder credentials to test keyring..."
  echo "must motion super wedding record raccoon toast public dance dial index embrace" | chaind keys add sovereign1-cosmos-holder --recover --keyring-backend test --algo secp256k1 --home "${CHAIN_HOME}"
  COSMOS_HOLDER_ADDR=$(chaind keys show sovereign1-cosmos-holder -a --keyring-backend test --home "${CHAIN_HOME}")
  echo "Cosmos holder address: ${COSMOS_HOLDER_ADDR}"

  echo "Funding validator, faucet and relayer accounts in genesis..."
  # Fund validator in genesis (500,000,000 CSOV = 500,000,000,000,000 ucsov, and 1,000,000 ESOV for EVM)
  chaind genesis add-genesis-account "${VAL_ADDR}" 1000000000000000000000000aesov,500000000000000ucsov --home "${CHAIN_HOME}"

  # Fund faucet in genesis (1,000,000,000 WSOV = 1,000,000,000,000,000 uwsov, 1,000,000,000 CSOV = 1,000,000,000,000,000 ucsov for gas, and 1,000,000 ESOV for EVM)
  chaind genesis add-genesis-account "${FAUCET_ADDR}" 1000000000000000000000000aesov,1000000000000000ucsov,1000000000000000uwsov --home "${CHAIN_HOME}"

  # Extract relayer address from genesis and fund it
  RELAYER_ADDR=$(grep -A 2 '"relayers"' "${CHAIN_HOME}/config/genesis.json" | grep '"address"' | head -n 1 | cut -d '"' -f 4 || true)
  if [ -n "${RELAYER_ADDR}" ]; then
    echo "Detected Relayer address: ${RELAYER_ADDR}. Funding..."
    chaind genesis add-genesis-account "${RELAYER_ADDR}" 100000000000000ucsov --home "${CHAIN_HOME}"
  fi

  echo "Generating validator genesis transaction (gentx)..."
  # Generate gentx (delegate 400,000,000 CSOV = 400,000,000,000,000 ucsov to validator)
  chaind genesis gentx validator 400000000000000ucsov --fees 5000aesov --keyring-backend test --chain-id "${CHAIN_ID}" --home "${CHAIN_HOME}"

  echo "Collecting gentxs..."
  # Collect gentxs into genesis.json
  chaind genesis collect-gentxs --home "${CHAIN_HOME}"
fi

# --- Patch app.toml ---
APP_TOML="${CHAIN_HOME}/config/app.toml"
if [ -f "${APP_TOML}" ]; then
  # gRPC: listen on all interfaces
  sed -i 's|^address = "localhost:9090"|address = "0.0.0.0:9090"|' "${APP_TOML}"
  # API: enable and listen on all interfaces
  sed -i '/^\[api\]/,/^\[/{s|^enable = false|enable = true|}' "${APP_TOML}"
  sed -i 's|^address = "tcp://localhost:1317"|address = "tcp://0.0.0.0:1317"|' "${APP_TOML}"
  # JSON-RPC: enable and listen on all interfaces
  sed -i '/^\[json-rpc\]/,/^\[/{s|^enable = false|enable = true|}' "${APP_TOML}"
  sed -i 's|^address = "127.0.0.1:8545"|address = "0.0.0.0:8545"|' "${APP_TOML}"
  sed -i 's|^ws-address = "127.0.0.1:8546"|ws-address = "0.0.0.0:8546"|' "${APP_TOML}"
  # Set minimum gas prices
  sed -i 's|^minimum-gas-prices =.*|minimum-gas-prices = "0aesov,0.025ucsov"|' "${APP_TOML}"
  # Enable EVM indexer for transaction receipts/queries
  sed -i 's|^enable-indexer = false|enable-indexer = true|' "${APP_TOML}"
  # Enable app-side mempool (required when mempool.type = "app" in config.toml)
  if grep -q "^max-txs =" "${APP_TOML}"; then
    sed -i 's|^max-txs =.*|max-txs = 5000|' "${APP_TOML}"
  else
    sed -i 's|^max-txs = -1|max-txs = 5000|' "${APP_TOML}"
  fi
  # Configure EVM mempool insert queue size
  if grep -q "insert-queue-size =" "${APP_TOML}"; then
    sed -i 's|.*insert-queue-size =.*|insert-queue-size = 5000|' "${APP_TOML}"
  else
    sed -i '/^\[evm.mempool\]/a insert-queue-size = 5000' "${APP_TOML}"
  fi
fi

# --- Patch config.toml ---
CONFIG_TOML="${CHAIN_HOME}/config/config.toml"
if [ -f "${CONFIG_TOML}" ]; then
  # CometBFT RPC: listen on all interfaces
  sed -i 's|^laddr = "tcp://127.0.0.1:26657"|laddr = "tcp://0.0.0.0:26657"|' "${CONFIG_TOML}"
  # Allow CORS for local dev
  sed -i 's|^cors_allowed_origins = \[\]|cors_allowed_origins = ["*"]|' "${CONFIG_TOML}"
  # EVM mempool requires CometBFT mempool.type = "app" (not the default "flood")
  sed -i 's|^type = "flood"|type = "app"|' "${CONFIG_TOML}"
  # Optimize block commit settings for production-grade speed and stability (Cosmos SDK v0.54)
  sed -i 's|^timeout_commit =.*|timeout_commit = "1s"|' "${CONFIG_TOML}"
  sed -i 's|^skip_timeout_commit =.*|skip_timeout_commit = false|' "${CONFIG_TOML}"
fi

# Extract chain ID dynamically from genesis.json if it exists
if [ -f "${CHAIN_HOME}/config/genesis.json" ]; then
  CHAIN_ID=$(grep '"chain_id"' "${CHAIN_HOME}/config/genesis.json" | head -n 1 | cut -d '"' -f 4)
  echo "Detected Chain ID from genesis: ${CHAIN_ID}"
fi

echo "Starting chaind..."
exec chaind start --home "${CHAIN_HOME}" --chain-id "${CHAIN_ID}" "$@"
