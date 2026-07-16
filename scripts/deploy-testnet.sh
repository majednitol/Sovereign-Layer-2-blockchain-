#!/usr/bin/env bash

# deploy-testnet.sh
# Automates the generation and configuration of the public testnet genesis.
# Usage: ./scripts/deploy-testnet.sh

set -euo pipefail

# 1. Load configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIG_FILE="${WORKSPACE_DIR}/infra/testnet/testnet-config.env"

echo "=== Loading Testnet Configuration ==="
if [ -f "${CONFIG_FILE}" ]; then
  # shellcheck disable=SC1090
  source "${CONFIG_FILE}"
else
  echo "Error: testnet-config.env not found at ${CONFIG_FILE}"
  exit 1
fi

echo "Chain ID: ${CHAIN_ID}"
echo "EVM Chain ID: ${EVM_CHAIN_ID}"
echo "BSC Target Chain ID: ${BSC_TARGET_CHAIN_ID}"

# 2. Export environment variables for the genesis script
export BRIDGE_GNOSIS_SAFE_ADDRESS="${TESTNET_BRIDGE_GNOSIS_SAFE_ADDRESS}"
export BRIDGE_LOCKBOX_ADDRESS="${TESTNET_BRIDGE_LOCKBOX_ADDRESS}"
export BRIDGE_CIRCUIT_BREAKER_ADDRESS="${TESTNET_BRIDGE_CIRCUIT_BREAKER_ADDRESS}"

# 3. Generate testnet genesis file
echo "=== Generating Genesis File for Testnet ==="
cd "${WORKSPACE_DIR}"
go run scripts/generate_genesis.go \
  --env=dev \
  --chain-id="${CHAIN_ID}" \
  --out="chain/genesis.testnet.json"

echo "=== Genesis File Generated at chain/genesis.testnet.json ==="

# 4. Ceremony collection simulation placeholder
echo "=== Ready for Genesis Ceremony ==="
echo "Moniker gentxs should be placed in: infra/testnet/gentxs/"
echo "Once gentxs are collected, run the following to compile the final genesis:"
echo "  chaind collect-gentxs --home \$HOME/.chain-testnet"
