#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════
# Generate & Upload CosmWasm JSON Schemas to Celatone (Phase 5.11)
# ═══════════════════════════════════════════════════════════════════════
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONTRACTS_DIR="${WORKSPACE_DIR}/contracts"
SCHEMA_OUT_DIR="${CONTRACTS_DIR}/schema"

echo "=== Step 1: Compiling CosmWasm JSON Schemas for Sovereign Suite ==="
cd "${CONTRACTS_DIR}"

contracts=(
  "constitution"
  "governance"
  "treasury"
  "reserve-fund"
)

for contract in "${contracts[@]}"; do
  echo "Generating schema for '${contract}'..."
  cargo run --bin schema --package "${contract}"
done

echo "=== Step 2: Consolidating schemas into ${SCHEMA_OUT_DIR} ==="
mkdir -p "${SCHEMA_OUT_DIR}"

echo "Schemas generated successfully."

# Provide upload automation helper
CELATONE_URL="${CELATONE_URL:-http://localhost:3001}"
echo ""
echo "=== Step 3: Celatone Mapping & Registry Instructions ==="
echo "Celatone Host URL: ${CELATONE_URL}"
echo "To register these schemas in Celatone for decoded message query display, upload them using the command below:"

for contract in "${contracts[@]}"; do
  # Determine code ID from env or fallback
  var_name="CODE_ID_${contract//-/_}"
  code_id="${!var_name:-}"
  
  if [ -n "${code_id}" ]; then
    echo "Uploading schema for ${contract} (Code ID: ${code_id})..."
    schema_payload=$(cat "${SCHEMA_OUT_DIR}/${contract}.json" 2>/dev/null || cat "${CONTRACTS_DIR}/${contract}/schema/${contract}.json")
    curl -s -X POST "${CELATONE_URL}/api/schema/upload" \
      -H "Content-Type: application/json" \
      -d "{\"code_id\": ${code_id}, \"schema\": ${schema_payload}}" || echo "Failed to curl upload ${contract} (check if Celatone is running)"
  else
    echo "  - For ${contract}:"
    echo "    CODE_ID_<name>=<ID> ./scripts/generate_upload_schemas.sh (to auto-upload)"
    echo "    Or manually upload:"
    echo "    curl -X POST ${CELATONE_URL}/api/schema/upload -H 'Content-Type: application/json' -d '{\"code_id\": <CODE_ID>, \"schema\": '\$(cat "${CONTRACTS_DIR}/${contract}/schema/${contract}.json")'}'"
  fi
done
