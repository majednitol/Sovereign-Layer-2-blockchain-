#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# Sovereign L1 — CosmWasm Counter Interact Script
# ═══════════════════════════════════════════════════════════════════════════════
#
# This script allows you to query and execute transactions on the CosmWasm Counter.
#
# Usage:
#   ./query_counter.sh <action> [arguments]
#
# Actions:
#   query-count               Query the current counter value
#   query-summary             Query the contract summary (owner, label, paused status)
#   increment                 Increment the counter by 1
#   decrement                 Decrement the counter by 1
#   set-label <new_label>     Set a new label for the contract (owner only)
#   pause                     Pause the contract (owner only)
#   unpause                   Unpause the contract (owner only)
#
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────
CONTRACT_ADDR="${CONTRACT_ADDR:-cosmos1pvrwmjuusn9wh34j7y520g8gumuy9xtl3gvprlljfdpwju3x7ucsn7cktv}"
CHAIN_NODE="chain-node"
CHAIN_ID="sovereign-1"
KEY_NAME="validator"
GAS_PRICES="0aesov"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${BLUE}ℹ️  ${1}${NC}"; }
log_ok()   { echo -e "${GREEN}✅ ${1}${NC}"; }
log_err()  { echo -e "${RED}❌ ${1}${NC}"; }

show_usage() {
    echo "Usage: $0 <action> [arguments]"
    echo ""
    echo "Actions:"
    echo "  query-count               Query the current counter value"
    echo "  query-summary             Query the contract summary"
    echo "  increment                 Increment the counter"
    echo "  decrement                 Decrement the counter"
    echo "  set-label <new_label>     Set a new label (owner only)"
    echo "  pause                     Pause the contract (owner only)"
    echo "  unpause                   Unpause the contract (owner only)"
    echo ""
    echo "Environment Variables:"
    echo "  CONTRACT_ADDR  Current: ${CONTRACT_ADDR}"
}

if [ $# -lt 1 ]; then
    show_usage
    exit 1
fi

ACTION="$1"

case "$ACTION" in
    query-count)
        log_info "Querying count from contract ${CONTRACT_ADDR}..."
        docker exec "$CHAIN_NODE" chaind q wasm contract-state smart "$CONTRACT_ADDR" '{"get_count":{}}' --output json
        ;;

    query-summary)
        log_info "Querying summary from contract ${CONTRACT_ADDR}..."
        docker exec "$CHAIN_NODE" chaind q wasm contract-state smart "$CONTRACT_ADDR" '{"get_summary":{}}' --output json
        ;;

    increment)
        log_info "Executing Increment transaction..."
        docker exec -it "$CHAIN_NODE" chaind tx wasm execute "$CONTRACT_ADDR" '{"increment":{}}' \
            --from "$KEY_NAME" \
            --chain-id "$CHAIN_ID" \
            --gas-prices "$GAS_PRICES" \
            --gas auto \
            --gas-adjustment 1.3 \
            --keyring-backend test \
            --home /root/.sovereign \
            -y --output json
        log_ok "Increment transaction submitted!"
        ;;

    decrement)
        log_info "Executing Decrement transaction..."
        docker exec -it "$CHAIN_NODE" chaind tx wasm execute "$CONTRACT_ADDR" '{"decrement":{}}' \
            --from "$KEY_NAME" \
            --chain-id "$CHAIN_ID" \
            --gas-prices "$GAS_PRICES" \
            --gas auto \
            --gas-adjustment 1.3 \
            --keyring-backend test \
            --home /root/.sovereign \
            -y --output json
        log_ok "Decrement transaction submitted!"
        ;;

    set-label)
        if [ $# -lt 2 ]; then
            log_err "Missing argument: new label string"
            echo "Usage: $0 set-label <new_label>"
            exit 1
        fi
        NEW_LABEL="$2"
        log_info "Executing SetLabel transaction to '${NEW_LABEL}'..."
        docker exec -it "$CHAIN_NODE" chaind tx wasm execute "$CONTRACT_ADDR" "{\"set_label\":{\"label\":\"${NEW_LABEL}\"}}" \
            --from "$KEY_NAME" \
            --chain-id "$CHAIN_ID" \
            --gas-prices "$GAS_PRICES" \
            --gas auto \
            --gas-adjustment 1.3 \
            --keyring-backend test \
            --home /root/.sovereign \
            -y --output json
        log_ok "SetLabel transaction submitted!"
        ;;

    pause)
        log_info "Executing Pause transaction..."
        docker exec -it "$CHAIN_NODE" chaind tx wasm execute "$CONTRACT_ADDR" '{"pause":{}}' \
            --from "$KEY_NAME" \
            --chain-id "$CHAIN_ID" \
            --gas-prices "$GAS_PRICES" \
            --gas auto \
            --gas-adjustment 1.3 \
            --keyring-backend test \
            --home /root/.sovereign \
            -y --output json
        log_ok "Pause transaction submitted!"
        ;;

    unpause)
        log_info "Executing Unpause transaction..."
        docker exec -it "$CHAIN_NODE" chaind tx wasm execute "$CONTRACT_ADDR" '{"unpause":{}}' \
            --from "$KEY_NAME" \
            --chain-id "$CHAIN_ID" \
            --gas-prices "$GAS_PRICES" \
            --gas auto \
            --gas-adjustment 1.3 \
            --keyring-backend test \
            --home /root/.sovereign \
            -y --output json
        log_ok "Unpause transaction submitted!"
        ;;

    *)
        log_err "Unknown action: ${ACTION}"
        show_usage
        exit 1
        ;;
esac
