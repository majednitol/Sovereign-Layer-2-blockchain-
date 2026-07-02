#!/bin/bash
set -euo pipefail

# Script to run the Sovereign L1 blockchain node natively on the host machine.
# This script initializes the environment, funds validator/faucet/relayer accounts,
# and starts the block producer node.

CHAIN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${CHAIN_DIR}/bin/chaind"
CHAIN_HOME="${CHAIN_DIR}/.sovereign-local"
CHAIN_ID="sovereign-1"
MONIKER="sovereign-node"

# 1. Compile the binary
echo "Compiling chaind daemon..."
cd "$CHAIN_DIR"
make build
cd - > /dev/null

# 2. Reset and initialize fresh state
echo "Resetting previous state at ${CHAIN_HOME}..."
rm -rf "${CHAIN_HOME}"

echo "Initializing chain home..."
"$BIN" init "${MONIKER}" --chain-id "${CHAIN_ID}" --home "${CHAIN_HOME}"

# Copy custom genesis.json
if [ -f "${CHAIN_DIR}/chain/genesis.json" ]; then
  echo "Using custom genesis.json..."
  cp "${CHAIN_DIR}/chain/genesis.json" "${CHAIN_HOME}/config/genesis.json"
fi

echo "Adding validator credentials to test keyring..."
"$BIN" keys add validator --keyring-backend test --home "${CHAIN_HOME}"
VAL_ADDR=$("$BIN" keys show validator -a --keyring-backend test --home "${CHAIN_HOME}")
echo "Validator address: ${VAL_ADDR}"

echo "Adding faucet credentials to test keyring..."
echo "test test test test test test test test test test test junk" | "$BIN" keys add faucet --recover --keyring-backend test --home "${CHAIN_HOME}"
FAUCET_ADDR=$("$BIN" keys show faucet -a --keyring-backend test --home "${CHAIN_HOME}")
echo "Faucet address: ${FAUCET_ADDR}"

echo "Funding validator, faucet and relayer accounts in genesis..."
# Fund validator in genesis (500,000,000 TOKEN = 500,000,000,000,000 utoken, and 1,000,000 atoken for EVM)
"$BIN" genesis add-genesis-account "${VAL_ADDR}" 500000000000000utoken,1000000000000000000000000atoken --home "${CHAIN_HOME}"

# Fund faucet in genesis (1,000,000,000 SOV = 1,000,000,000,000,000 usov, 1,000,000,000 TOKEN = 1,000,000,000,000,000 utoken for gas, and 1,000,000 atoken for EVM)
"$BIN" genesis add-genesis-account "${FAUCET_ADDR}" 1000000000000000usov,1000000000000000utoken,1000000000000000000000000atoken --home "${CHAIN_HOME}"

# Extract relayer address from genesis and fund it
RELAYER_ADDR=$(grep -A 2 '"relayers"' "${CHAIN_HOME}/config/genesis.json" | grep '"address"' | head -n 1 | cut -d '"' -f 4 || true)
if [ -n "${RELAYER_ADDR}" ]; then
  echo "Detected Relayer address: ${RELAYER_ADDR}. Funding..."
  "$BIN" genesis add-genesis-account "${RELAYER_ADDR}" 100000000000000utoken --home "${CHAIN_HOME}"
fi

echo "Generating validator genesis transaction (gentx)..."
"$BIN" genesis gentx validator 400000000000000utoken --keyring-backend test --chain-id "${CHAIN_ID}" --home "${CHAIN_HOME}"

echo "Collecting gentxs..."
"$BIN" genesis collect-gentxs --home "${CHAIN_HOME}"

# --- Patch app.toml config ---
APP_TOML="${CHAIN_HOME}/config/app.toml"
if [ -f "${APP_TOML}" ]; then
  echo "Patching app.toml..."
  python3 -c '
import sys, re
path = sys.argv[1]
with open(path, "r") as f:
    content = f.read()

# Enable API
api_match = re.search(r"(\[api\]\n(?:.*\n)*?enable\s*=\s*)(false|true)", content)
if api_match:
    content = content.replace(api_match.group(0), api_match.group(1) + "true")

content = re.sub(r"address\s*=\s*\"tcp://localhost:1317\"", "address = \"tcp://0.0.0.0:1317\"", content)
content = re.sub(r"address\s*=\s*\"localhost:9090\"", "address = \"0.0.0.0:9090\"", content)

# Enable JSON-RPC
rpc_match = re.search(r"(\[json-rpc\]\n(?:.*\n)*?enable\s*=\s*)(false|true)", content)
if rpc_match:
    content = content.replace(rpc_match.group(0), rpc_match.group(1) + "true")

content = re.sub(r"address\s*=\s*\"127.0.0.1:8545\"", "address = \"0.0.0.0:8545\"", content)
content = re.sub(r"ws-address\s*=\s*\"127.0.0.1:8546\"", "ws-address = \"0.0.0.0:8546\"", content)

# Set minimum gas prices
content = re.sub(r"minimum-gas-prices\s*=\s*\".*\"", "minimum-gas-prices = \"0atoken\"", content)
content = re.sub(r"minimum-gas-prices\s*=\s*'\x27\x27'", "minimum-gas-prices = \"0atoken\"", content)

# Enable EVM indexer
content = re.sub(r"enable-indexer\s*=\s*false", "enable-indexer = true", content)

# Configure max-txs
if "max-txs =" in content:
    content = re.sub(r"max-txs\s*=\s*.*", "max-txs = 5000", content)
else:
    content = content.replace("[mempool]", "[mempool]\nmax-txs = 5000")

# Configure insert-queue-size
if "insert-queue-size =" in content:
    content = re.sub(r"insert-queue-size\s*=\s*.*", "insert-queue-size = 5000", content)
else:
    if "[evm.mempool]" in content:
        content = content.replace("[evm.mempool]", "[evm.mempool]\ninsert-queue-size = 5000")

with open(path, "w") as f:
    f.write(content)
' "${APP_TOML}"
fi

# --- Patch config.toml config ---
CONFIG_TOML="${CHAIN_HOME}/config/config.toml"
if [ -f "${CONFIG_TOML}" ]; then
  echo "Patching config.toml..."
  python3 -c '
import sys, re
path = sys.argv[1]
with open(path, "r") as f:
    content = f.read()

content = re.sub(r"laddr\s*=\s*\"tcp://127.0.0.1:26657\"", "laddr = \"tcp://0.0.0.0:26657\"", content)
content = re.sub(r"cors_allowed_origins\s*=\s*\[\]", "cors_allowed_origins = [\"*\"]", content)
content = re.sub(r"type\s*=\s*\"flood\"", "type = \"app\"", content)

with open(path, "w") as f:
    f.write(content)
' "${CONFIG_TOML}"
fi

echo "=========================================================="
echo "Sovereign L1 Node initialized successfully!"
echo "Home directory: ${CHAIN_HOME}"
echo "Starting block production..."
echo "=========================================================="

exec "$BIN" start --home "${CHAIN_HOME}" --chain-id "${CHAIN_ID}"
