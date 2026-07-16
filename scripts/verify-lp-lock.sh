#!/usr/bin/env bash

# verify-lp-lock.sh
# Verifies if PancakeSwap LP tokens are locked or burned.
# Uses public BSC RPC endpoint or a custom RPC endpoint provided by environment.

set -euo pipefail

RPC_URL="${BSC_RPC_URL:-https://bsc-dataseed.binance.org/}"
TOKEN_ADDRESS=""
LOCKER_ADDRESS=""
EXPECTED_MIN_LOCKED=""

usage() {
  echo "Usage: $0 -t <lp_token_address> -l <locker_or_burn_address> -m <min_expected_locked_amount>"
  echo "  -t: PancakeSwap V2 LP token contract address (hex)"
  echo "  -l: UNCX/TeamFinance lock contract address or dead address (0x000000000000000000000000000000000000dEaD)"
  echo "  -m: Minimum expected LP tokens locked (wei format or raw integer)"
  exit 1
}

while getopts "t:l:m:" opt; do
  case "$opt" in
    t) TOKEN_ADDRESS="$OPTARG" ;;
    l) LOCKER_ADDRESS="$OPTARG" ;;
    m) EXPECTED_MIN_LOCKED="$OPTARG" ;;
    *) usage ;;
  esac
done

if [ -z "$TOKEN_ADDRESS" ] || [ -z "$LOCKER_ADDRESS" ] || [ -z "$EXPECTED_MIN_LOCKED" ]; then
  usage
fi

# Clean addresses
clean_addr() {
  echo "${1#0x}" | tr '[:upper:]' '[:lower:]'
}

CLEAN_TOKEN=$(clean_addr "$TOKEN_ADDRESS")
CLEAN_LOCKER=$(clean_addr "$LOCKER_ADDRESS")

# ERC-20 balanceOf(address) selector is 70a08231
# Address must be padded to 32 bytes (64 hex characters)
PADDED_LOCKER=$(printf "%064s" "$CLEAN_LOCKER" | tr ' ' '0')
DATA="0x70a08231${PADDED_LOCKER}"

echo "Querying BSC RPC: $RPC_URL"
echo "Token: 0x$CLEAN_TOKEN"
echo "Holder/Locker: 0x$CLEAN_LOCKER"

# Build JSON-RPC request
REQUEST_JSON=$(cat <<EOF
{
  "jsonrpc": "2.0",
  "method": "eth_call",
  "params": [
    {
      "to": "0x$CLEAN_TOKEN",
      "data": "$DATA"
    },
    "latest"
  ],
  "id": 1
}
EOF
)

RESPONSE=$(curl -s -X POST -H "Content-Type: application/json" --data "$REQUEST_JSON" "$RPC_URL")

if echo "$RESPONSE" | grep -q "error"; then
  echo "Error from RPC response:"
  echo "$RESPONSE" | jq .error
  exit 1
fi

HEX_RESULT=$(echo "$RESPONSE" | jq -r .result)

if [ "$HEX_RESULT" = "null" ] || [ -z "$HEX_RESULT" ]; then
  echo "Failed to retrieve balance from contract (result is null or empty)."
  exit 1
fi

# Parse hex to decimal (removing 0x prefix if present)
HEX_CLEAN="${HEX_RESULT#0x}"
# Convert hex to decimal using python or bc (bc might not be installed, python is standard on macOS/Linux)
DEC_BALANCE=$(python3 -c "print(int('$HEX_CLEAN', 16))")

echo "Current balance: $DEC_BALANCE units"
echo "Expected minimum: $EXPECTED_MIN_LOCKED units"

if [ "$DEC_BALANCE" -ge "$EXPECTED_MIN_LOCKED" ]; then
  echo "SUCCESS: LP tokens are locked/burned. Current balance exceeds or meets expected minimum."
  
  # Output JSON report
  cat <<EOF
{
  "status": "verified",
  "lp_token": "0x$CLEAN_TOKEN",
  "locker_address": "0x$CLEAN_LOCKER",
  "locked_balance": "$DEC_BALANCE",
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
  exit 0
else
  echo "FAILURE: LP tokens are not sufficiently locked/burned."
  exit 2
fi
