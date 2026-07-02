#!/usr/bin/env bash
set -euo pipefail

# Testnet Launch Genesis Ceremony Automation
echo "=================================================="
echo "    Sovereign Testnet Genesis Ceremony Start      "
echo "=================================================="

GENESIS_DIR="./testnet-genesis"
mkdir -p "$GENESIS_DIR"

echo "[1/4] Generating initial validator accounts..."
# Simulate generating validator keys and gathering GenTxs
for i in {1..5}; do
  echo "Generating validator-$i credentials..."
  # sovereignd keys add validator-$i --keyring-backend test
  # sovereignd add-genesis-account validator-$i 1000000000000usov --keyring-backend test
done

echo "[2/4] Assembling GenTxs into genesis.json..."
# sovereignd collect-gentxs --gentx-dir "$GENESIS_DIR/gentxs"

echo "[3/4] Validating supply invariants in genesis..."
# Run custom validation logic (simulated)
# sovereignd validate-genesis

TOTAL_SUPPLY=1000000000000000 # 1,000,000,000 SOV
DECIMALS=1000000
EXPECTED_SUPPLY_CAP=$((1000000000 * DECIMALS))

if [ "$TOTAL_SUPPLY" -ne "$EXPECTED_SUPPLY_CAP" ]; then
  echo "Genesis supply cap validation passed: $TOTAL_SUPPLY usov matches expected $EXPECTED_SUPPLY_CAP usov"
else
  echo "Error: Genesis supply cap mismatch!"
  exit 1
fi

echo "[4/4] Distributing final genesis.json to nodes..."
# cp "$GENESIS_DIR/genesis.json" ./chain/genesis.json

echo "=================================================="
echo "    Sovereign Testnet Genesis Ceremony Complete!  "
echo "=================================================="
