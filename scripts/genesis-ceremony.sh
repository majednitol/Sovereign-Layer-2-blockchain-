#!/usr/bin/env bash

# genesis-ceremony.sh
# Coordinates the gathering of validator gentx files and compiling the final genesis.
# This script enforces safety checks BEFORE producing a genesis that could be used on mainnet.
#
# Usage: ./scripts/genesis-ceremony.sh [--skip-horcrux-check]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
GENESIS_BASE="${WORKSPACE_DIR}/chain/genesis.prod.json"
GENTXS_DIR="${WORKSPACE_DIR}/infra/mainnet/gentxs"
HOME_DIR="${WORKSPACE_DIR}/.genesis-ceremony"
MIN_VALIDATORS=3

echo "=== Genesis Ceremony Tooling ==="
echo "Started at: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo ""

# ──────────────────────────────────────────────────────────────
# PRE-FLIGHT SAFETY CHECKS
# These checks prevent the ceremony from producing a genesis
# that would fail on chain start or ship with placeholder values.
# ──────────────────────────────────────────────────────────────

PREFLIGHT_FAILURES=0

# Check 1: Base genesis exists
if [ ! -f "${GENESIS_BASE}" ]; then
  echo "FAIL: Base production genesis not found at ${GENESIS_BASE}."
  exit 1
fi
echo "PASS: Base genesis found at ${GENESIS_BASE}"

# Check 2: No OWNER_ACTION_REQUIRED placeholders remaining
PLACEHOLDER_HITS=$(grep -c "OWNER_ACTION_REQUIRED" "${GENESIS_BASE}" || true)
if [ "${PLACEHOLDER_HITS}" -gt 0 ]; then
  echo ""
  echo "FAIL: genesis.prod.json still contains ${PLACEHOLDER_HITS} OWNER_ACTION_REQUIRED placeholder(s):"
  grep -n "OWNER_ACTION_REQUIRED" "${GENESIS_BASE}" | while IFS= read -r line; do
    echo "  ${line}"
  done
  echo ""
  echo "  You MUST replace these with real addresses before running the ceremony."
  echo "  See: doc/mainnet/bsc-bridge-checklist.md for bridge address requirements."
  PREFLIGHT_FAILURES=$((PREFLIGHT_FAILURES + 1))
fi

# Check 3: No obvious test/placeholder bridge addresses
for pattern in "0x0000000000000000000000000000000000000000" "0x1111111111111111111111111111111111111111" "cosmos1gs_addr" "cosmos1cb_addr" "cosmos1dev"; do
  if grep -q "${pattern}" "${GENESIS_BASE}"; then
    echo "FAIL: genesis.prod.json contains suspicious placeholder pattern: '${pattern}'"
    PREFLIGHT_FAILURES=$((PREFLIGHT_FAILURES + 1))
  fi
done

# Check 4: uwsov (Bridge Minted Token) supply must be 0 at genesis
UWSOV_SUPPLY=$(grep -A1 '"uwsov"' "${GENESIS_BASE}" | grep '"amount"' | head -1 | sed 's/.*"amount"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || echo "0")
if [ -n "${UWSOV_SUPPLY}" ] && [ "${UWSOV_SUPPLY}" != "0" ]; then
  echo "FAIL: uwsov supply in genesis is '${UWSOV_SUPPLY}' — must be '0' (bridge-minted tokens cannot exist before any bridge activity)"
  PREFLIGHT_FAILURES=$((PREFLIGHT_FAILURES + 1))
fi

# Check 5: Minimum gentx count
GENTX_COUNT=0
for f in "${GENTXS_DIR}"/*.json; do
  [ -e "$f" ] || continue
  GENTX_COUNT=$((GENTX_COUNT + 1))
done

if [ "${GENTX_COUNT}" -lt "${MIN_VALIDATORS}" ]; then
  echo "FAIL: Found only ${GENTX_COUNT} gentx file(s), minimum required is ${MIN_VALIDATORS}."
  echo "  Validators must submit gentx files to: ${GENTXS_DIR}/"
  echo "  See: doc/mainnet/validator-onboarding.md for instructions."
  PREFLIGHT_FAILURES=$((PREFLIGHT_FAILURES + 1))
else
  echo "PASS: Found ${GENTX_COUNT} validator gentx files (minimum: ${MIN_VALIDATORS})"
fi

# Check 6: Horcrux ceremony config check (optional skip)
if [[ "${1:-}" != "--skip-horcrux-check" ]]; then
  HORCRUX_CHECK="${SCRIPT_DIR}/horcrux_ceremony_check.sh"
  if [ -f "${HORCRUX_CHECK}" ]; then
    echo ""
    echo "Running Horcrux configuration verification..."
    if ! bash "${HORCRUX_CHECK}"; then
      echo "FAIL: Horcrux ceremony check failed."
      PREFLIGHT_FAILURES=$((PREFLIGHT_FAILURES + 1))
    fi
  else
    echo "WARN: Horcrux ceremony check script not found at ${HORCRUX_CHECK}"
  fi
else
  echo "SKIP: Horcrux check skipped (--skip-horcrux-check flag)"
fi

# Check 7: chain-id must be sovereign-1 (mainnet)
CHAIN_ID=$(grep '"chain_id"' "${GENESIS_BASE}" | head -1 | sed 's/.*"chain_id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
if [ "${CHAIN_ID}" != "sovereign-1" ]; then
  echo "FAIL: chain_id in genesis is '${CHAIN_ID}', expected 'sovereign-1' for mainnet"
  PREFLIGHT_FAILURES=$((PREFLIGHT_FAILURES + 1))
else
  echo "PASS: chain_id is 'sovereign-1'"
fi

# ──────────────────────────────────────────────────────────────
# ABORT IF PRE-FLIGHT CHECKS FAILED
# ──────────────────────────────────────────────────────────────

echo ""
if [ "${PREFLIGHT_FAILURES}" -gt 0 ]; then
  echo "========================================================"
  echo "CEREMONY ABORTED: ${PREFLIGHT_FAILURES} pre-flight check(s) failed."
  echo "Fix all issues above before re-running the ceremony."
  echo "========================================================"
  exit 1
fi

echo "All pre-flight checks passed."
echo ""

# ──────────────────────────────────────────────────────────────
# CEREMONY EXECUTION
# ──────────────────────────────────────────────────────────────

# 1. Setup clean ceremony directory
rm -rf "${HOME_DIR}"
mkdir -p "${HOME_DIR}/config"

# 2. Copy production base genesis
cp "${GENESIS_BASE}" "${HOME_DIR}/config/genesis.json"
echo "Copied base genesis to ceremony directory."

# 3. Import gentxs
echo "Importing ${GENTX_COUNT} gentx files from ${GENTXS_DIR}..."
mkdir -p "${HOME_DIR}/config/gentx"

for f in "${GENTXS_DIR}"/*.json; do
  [ -e "$f" ] || continue
  cp "$f" "${HOME_DIR}/config/gentx/"
  echo "  Imported: $(basename "$f")"
done

# 4. Compile final genesis with collect-gentxs
echo ""
echo "Compiling final genesis with chaind collect-gentxs..."
chaind collect-gentxs --home "${HOME_DIR}"

# 5. Validate the compiled genesis
echo ""
echo "Validating compiled genesis..."
if chaind validate-genesis --home "${HOME_DIR}" 2>/dev/null; then
  echo "PASS: Genesis validation succeeded."
else
  echo "FAIL: Genesis validation failed! The compiled genesis is invalid."
  echo "  Check the chaind output above for details."
  exit 1
fi

# 6. Generate SHA-256 Checksum
GENESIS_FILE="${HOME_DIR}/config/genesis.json"
SHA=$(shasum -a 256 "${GENESIS_FILE}" | awk '{print $1}')
SIZE=$(wc -c < "${GENESIS_FILE}" | tr -d ' ')

echo ""
echo "========================================================"
echo "GENESIS CEREMONY COMPLETED SUCCESSFULLY"
echo "========================================================"
echo ""
echo "  Final Genesis Hash (SHA-256): ${SHA}"
echo "  File Size: ${SIZE} bytes"
echo "  Validators Included: ${GENTX_COUNT}"
echo "  Chain ID: ${CHAIN_ID}"
echo "  Timestamp: $(date -u '+%Y-%m-%dT%H:%M:%SZ')"
echo ""
echo "  Genesis file: ${GENESIS_FILE}"
echo ""
echo "========================================================"
echo "NEXT STEPS FOR ALL VALIDATOR OPERATORS:"
echo "========================================================"
echo ""
echo "  1. Copy the genesis file to your node:"
echo "     cp ${GENESIS_FILE} \$HOME/.chain/config/genesis.json"
echo ""
echo "  2. Verify the checksum matches EXACTLY:"
echo "     shasum -a 256 \$HOME/.chain/config/genesis.json"
echo "     Expected: ${SHA}"
echo ""
echo "  3. Start your node ONLY after confirming the hash matches."
echo ""
echo "  4. Cold multisig custodians: confirm you are reachable for"
echo "     the first 48 hours post-launch in case of emergency pause."
echo ""
