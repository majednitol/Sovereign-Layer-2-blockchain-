#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════
# Phase 1 to 5 Testnet Integration E2E Test Suite Orchestrator
# ═══════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "======================================================================"
# Export environment variables for the test
export BSC_TESTNET_PRIVATE_KEY="${BSC_TESTNET_PRIVATE_KEY:?Must set BSC_TESTNET_PRIVATE_KEY}"
export BSC_ERC20_ADDRESS="${BSC_ERC20_ADDRESS:-0xE26314197A03034962DfEe0AE688E3Dc57F493CA}"
export BSC_TESTNET_RPC_URL="${BSC_TESTNET_RPC_URL:-https://bsc-testnet-rpc.publicnode.com}"
export DOCKER_BUILDKIT=1
export COMPOSE_DOCKER_CLI_BUILD=1

echo "Deployer Address:"
cast wallet address --private-key "${BSC_TESTNET_PRIVATE_KEY}"

echo "E2E Token Address: ${BSC_ERC20_ADDRESS}"
echo "RPC URL: ${BSC_TESTNET_RPC_URL}"
echo "======================================================================"

# Compile Go code first to ensure all modules are green
echo "Compiling workspace modules..."
make build

# Run the integration test
echo "Running Real Testnet integration test..."
cd "${WORKSPACE_DIR}/e2e"
go test -v -run TestRealTestnetIntegration

echo "======================================================================"
echo "    Integration Test execution complete!"
echo "======================================================================"
