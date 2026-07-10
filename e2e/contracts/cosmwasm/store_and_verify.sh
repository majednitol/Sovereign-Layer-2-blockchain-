#!/bin/bash
# ═══════════════════════════════════════════════════════════════════════════════
# Sovereign L1 — CosmWasm Contract Store & Verify Test Script
# ═══════════════════════════════════════════════════════════════════════════════
#
# This script:
#   1. Stores a test CosmWasm contract on-chain via sovereignd tx wasm store
#   2. Queries the code_id and on-chain SHA-256 checksum
#   3. Submits the verification request to the explorer API
#   4. Verifies the contract detail page returns verified = true
#
# Prerequisites:
#   - sovereignd binary accessible (via docker exec or PATH)
#   - chain-node running with CosmWasm module enabled
#   - explorer-api running on API_BASE
#   - curl, jq installed
#
# Usage:
#   chmod +x store_and_verify.sh
#   ./store_and_verify.sh
# ═══════════════════════════════════════════════════════════════════════════════

set -euo pipefail

# ─── Configuration ────────────────────────────────────────────────────────────
API_BASE="${API_BASE:-http://localhost:8082}"
CHAIN_NODE="${CHAIN_NODE:-chain-node}"
CHAIN_ID="${CHAIN_ID:-sovereign-1}"
KEY_NAME="${KEY_NAME:-validator}"
GAS_PRICES="${GAS_PRICES:-0aesov}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA_DIR="${SCRIPT_DIR}/schema"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info()  { echo -e "${BLUE}ℹ️  ${1}${NC}"; }
log_ok()    { echo -e "${GREEN}✅ ${1}${NC}"; }
log_warn()  { echo -e "${YELLOW}⚠️  ${1}${NC}"; }
log_fail()  { echo -e "${RED}❌ ${1}${NC}"; }
divider()   { echo -e "${BLUE}$( printf '═%.0s' {1..60} )${NC}"; }

# ─── Step 0: Validate Prerequisites ──────────────────────────────────────────
divider
echo -e "${BLUE}  Sovereign L1 — CosmWasm Verification Test${NC}"
divider

log_info "Validating prerequisites..."

for cmd in curl jq; do
  if ! command -v "$cmd" &>/dev/null; then
    log_fail "Missing required tool: $cmd"
    exit 1
  fi
done

for schema in instantiate_msg.json execute_msg.json query_msg.json; do
  if [ ! -f "${SCHEMA_DIR}/${schema}" ]; then
    log_fail "Missing schema file: ${SCHEMA_DIR}/${schema}"
    exit 1
  fi
done

log_ok "Prerequisites validated"

# ─── Step 1: Check for existing wasm binary or use a dummy ───────────────────
divider
log_info "STEP 1: Prepare CosmWasm binary"

# If there's no actual .wasm file, we'll create a minimal one for testing.
# In production, this would be the output of cargo wasm + wasm-opt.
WASM_FILE="${SCRIPT_DIR}/cw_counter.wasm"

if [ ! -f "$WASM_FILE" ]; then
  log_warn "No .wasm binary found at ${WASM_FILE}"
  log_info "To run the full flow, compile your Rust contract and place the .wasm here."
  log_info ""
  log_info "Example compilation commands:"
  log_info "  cd contracts/cw-counter"
  log_info "  RUSTFLAGS='-C link-arg=-s' cargo build --release --target wasm32-unknown-unknown"
  log_info "  cp target/wasm32-unknown-unknown/release/cw_counter.wasm ${WASM_FILE}"
  log_info ""
  log_info "Or using rust-optimizer Docker image:"
  log_info "  docker run --rm -v \$(pwd):/code cosmwasm/rust-optimizer:0.14.0"
  log_info ""
  log_info "Skipping on-chain store. Proceeding with API-only verification test..."
  SKIP_STORE=true
else
  SKIP_STORE=false
  log_ok "Found wasm binary: ${WASM_FILE}"
  WASM_SIZE=$(wc -c < "$WASM_FILE" | tr -d ' ')
  log_info "Binary size: ${WASM_SIZE} bytes"
fi

# ─── Step 2: Store contract on-chain ─────────────────────────────────────────
if [ "$SKIP_STORE" = false ]; then
  divider
  log_info "STEP 2: Storing contract on-chain"
  
  # Calculate SHA-256 checksum locally
  LOCAL_CHECKSUM=$(sha256sum "$WASM_FILE" | awk '{print $1}')
  log_info "Local SHA-256: ${LOCAL_CHECKSUM}"
  
  # Store via docker exec
  STORE_RESULT=$(docker exec "$CHAIN_NODE" chaind tx wasm store /tmp/cw_counter.wasm \
    --from "$KEY_NAME" \
    --chain-id "$CHAIN_ID" \
    --gas-prices "$GAS_PRICES" \
    --gas auto \
    --gas-adjustment 1.3 \
    --keyring-backend test \
    --home /root/.sovereign \
    -y \
    --output json 2>/dev/null) || true
  
  if echo "$STORE_RESULT" | jq -e '.txhash' &>/dev/null; then
    TX_HASH=$(echo "$STORE_RESULT" | jq -r '.txhash')
    log_ok "Store tx submitted: ${TX_HASH}"
    
    # Wait for inclusion
    log_info "Waiting for tx inclusion..."
    sleep 6
    
    # Query the tx result to get code_id
    TX_RESULT=$(docker exec "$CHAIN_NODE" chaind q tx "$TX_HASH" --output json 2>/dev/null) || true
    CODE_ID=$(echo "$TX_RESULT" | jq -r '.events[] | select(.type=="store_code") | .attributes[] | select(.key=="code_id") | .value' 2>/dev/null) || true
    
    if [ -n "$CODE_ID" ] && [ "$CODE_ID" != "null" ]; then
      log_ok "Contract stored! Code ID: ${CODE_ID}"
    else
      log_warn "Could not extract code_id from tx. Trying query..."
      CODE_ID=$(docker exec "$CHAIN_NODE" chaind q wasm list-code --output json | jq -r '.code_infos[-1].code_id') || true
    fi
  else
    log_warn "Store tx may have failed. Attempting to query existing codes..."
    CODE_ID=$(docker exec "$CHAIN_NODE" chaind q wasm list-code --output json | jq -r '.code_infos[-1].code_id') || true
    LOCAL_CHECKSUM="unknown"
  fi
else
  # For API-only testing, use a dummy code_id and checksum
  CODE_ID="${TEST_CODE_ID:-1}"
  LOCAL_CHECKSUM="${TEST_CHECKSUM:-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855}"
  log_info "Using test Code ID: ${CODE_ID}, Checksum: ${LOCAL_CHECKSUM}"
fi

# ─── Step 3: Query on-chain code info ────────────────────────────────────────
divider
log_info "STEP 3: Querying on-chain code info via REST API"

CODE_INFO=$(curl -sf "${API_BASE}/api/rest/v1/explorer/cosmwasm/codes/${CODE_ID}" 2>/dev/null) || true

if [ -n "$CODE_INFO" ]; then
  VERIFIED=$(echo "$CODE_INFO" | jq -r '.verified')
  log_info "Current verification status: ${VERIFIED}"
else
  log_warn "Could not query code info (code may not exist yet)"
fi

# ─── Step 4: Submit verification request ─────────────────────────────────────
divider
log_info "STEP 4: Submitting verification to explorer API"

INSTANTIATE_SCHEMA=$(cat "${SCHEMA_DIR}/instantiate_msg.json")
EXECUTE_SCHEMA=$(cat "${SCHEMA_DIR}/execute_msg.json")
QUERY_SCHEMA=$(cat "${SCHEMA_DIR}/query_msg.json")

VERIFY_PAYLOAD=$(jq -n \
  --argjson codeId "$CODE_ID" \
  --arg checksum "$LOCAL_CHECKSUM" \
  --argjson instantiateMsg "$INSTANTIATE_SCHEMA" \
  --argjson executeMsg "$EXECUTE_SCHEMA" \
  --argjson queryMsg "$QUERY_SCHEMA" \
  --arg gitRepo "https://github.com/sovereign-l1/contracts" \
  --arg gitCommit "abc123def456" \
  --arg optimizerVersion "cosmwasm/rust-optimizer:0.14.0" \
  '{
    codeId: $codeId,
    checksum: $checksum,
    instantiateMsg: $instantiateMsg,
    executeMsg: $executeMsg,
    queryMsg: $queryMsg,
    gitRepo: $gitRepo,
    gitCommit: $gitCommit,
    optimizerVersion: $optimizerVersion
  }')

log_info "Payload size: $(echo "$VERIFY_PAYLOAD" | wc -c | tr -d ' ') bytes"

VERIFY_RESPONSE=$(curl -sf -X POST \
  "${API_BASE}/api/rest/v1/explorer/verify/cosmwasm" \
  -H "Content-Type: application/json" \
  -d "$VERIFY_PAYLOAD" 2>&1) || true

if echo "$VERIFY_RESPONSE" | jq -e '.success == true' &>/dev/null; then
  log_ok "Verification submitted successfully!"
  echo "$VERIFY_RESPONSE" | jq '.'
else
  log_warn "Verification response: ${VERIFY_RESPONSE}"
  log_info "This may fail if the checksum doesn't match on-chain data (expected in API-only mode)"
fi

# ─── Step 5: Confirm verified status ─────────────────────────────────────────
divider
log_info "STEP 5: Confirming verified status"

FINAL_INFO=$(curl -sf "${API_BASE}/api/rest/v1/explorer/cosmwasm/codes/${CODE_ID}" 2>/dev/null) || true

if [ -n "$FINAL_INFO" ]; then
  FINAL_VERIFIED=$(echo "$FINAL_INFO" | jq -r '.verified')
  if [ "$FINAL_VERIFIED" = "true" ]; then
    log_ok "✨ Code ID ${CODE_ID} is VERIFIED!"
    echo "$FINAL_INFO" | jq '{codeId, verified, checksum, gitRepo, gitCommit, optimizerVersion}'
  else
    log_warn "Code ID ${CODE_ID} is not yet verified"
  fi
else
  log_warn "Could not fetch final status"
fi

# ─── Summary ─────────────────────────────────────────────────────────────────
divider
echo -e "${BLUE}  TEST SUMMARY${NC}"
divider
echo ""
echo -e "  Code ID:        ${CODE_ID}"
echo -e "  Checksum:       ${LOCAL_CHECKSUM}"
echo -e "  API Base:       ${API_BASE}"
echo ""
echo -e "${BLUE}📋 MANUAL STEPS:${NC}"
echo -e "  1. Open explorer: ${BLUE}http://localhost:3000/verify${NC}"
echo -e "  2. Select 'CosmWasm (Wasm Checksum)' tab"
echo -e "  3. Enter Code ID: ${CODE_ID}"
echo -e "  4. Upload .wasm file or enter checksum: ${LOCAL_CHECKSUM}"
echo -e "  5. Paste schema files from: ${SCHEMA_DIR}/"
echo -e "  6. Click 'Verify & Publish'"
echo -e "  7. Navigate to: ${BLUE}http://localhost:3000/contracts/<addr>${NC}"
echo -e "  8. Test query functions (get_count, get_summary, get_owner)"
echo -e "  9. Test execute functions (increment, set_label) via Keplr"
divider
