#!/usr/bin/env bash
set -euo pipefail

# Software Upgrade Drill Runbook Simulator
echo "=================================================="
echo "    Sovereign Testnet Upgrade Drill Simulator     "
echo "=================================================="

UPGRADE_HEIGHT=50
UPGRADE_NAME="v2-testnet"

echo "[1/5] Submitting governance upgrade proposal..."
# sovereignd tx gov submit-legacy-proposal software-upgrade "$UPGRADE_NAME" \
#   --upgrade-height "$UPGRADE_HEIGHT" \
#   --title "Testnet Upgrade" \
#   --description "Drill for testnet upgrade" \
#   --from admin \
#   --yes

echo "[2/5] Simulating voting period passage..."
# sovereignd tx gov vote 1 yes --from validator-1 --yes
# sovereignd tx gov vote 1 yes --from validator-2 --yes

echo "[3/5] Monitoring block height. Waiting for upgrade height $UPGRADE_HEIGHT..."
CURRENT_HEIGHT=40
while [ "$CURRENT_HEIGHT" -lt "$UPGRADE_HEIGHT" ]; do
  echo "Current Block Height: $CURRENT_HEIGHT"
  CURRENT_HEIGHT=$((CURRENT_HEIGHT + 2))
  sleep 0.1
done

echo "Upgrade height $UPGRADE_HEIGHT reached! Chain has successfully halted."

echo "[4/5] Executing validator binary swap..."
# mv ./build/sovereignd-new ./build/sovereignd
echo "Swapped sovereignd binary with upgraded v2 version successfully."

echo "[5/5] Restarting node and verifying consensus resume..."
# sovereignd start
echo "Chain successfully resumed block production. Upgraded store active."

echo "=================================================="
echo "    Upgrade Drill Simulation Complete & Success!   "
echo "=================================================="
