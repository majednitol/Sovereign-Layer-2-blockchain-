#!/bin/bash
set -euo pipefail

# Horcrux Ceremony Configuration Check
# Checks all 3 horcrux config files for 2-of-3 threshold compliance and double-signing protection.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="${SCRIPT_DIR}/../infra/horcrux"
FILES=("horcrux-0.toml" "horcrux-1.toml" "horcrux-2.toml")
FAILURES=0

echo "=== Horcrux Ceremony Configuration Verification ==="
echo

for file in "${FILES[@]}"; do
  path="${CONFIG_DIR}/${file}"
  if [ ! -f "$path" ]; then
    echo "FAIL: Config file missing at $path"
    FAILURES=$((FAILURES + 1))
    continue
  fi

  echo "Checking $path..."

  # Extract threshold
  threshold=$(grep -E '^threshold\s*=' "$path" | awk -F '=' '{print $2}' | tr -d '[:space:]')
  if [ "$threshold" != "2" ]; then
    echo "  FAIL: Threshold is '$threshold', expected '2'"
    FAILURES=$((FAILURES + 1))
  else
    echo "  PASS: Threshold is 2"
  fi

  # Extract double sign protection
  double_sign=$(grep -E 'double-sign-protection\s*=' "$path" | awk -F '=' '{print $2}' | tr -d '[:space:]')
  if [ "$double_sign" != "true" ]; then
    echo "  FAIL: double-sign-protection is '$double_sign', expected 'true'"
    FAILURES=$((FAILURES + 1))
  else
    echo "  PASS: Double-sign protection enabled"
  fi

  # Count cosigner peers
  peer_count=$(grep -c 'address\s*=\s*' "$path" || true)
  if [ "$peer_count" -ne 3 ]; then
    echo "  FAIL: Found $peer_count cosigner peers, expected exactly 3 for 2-of-3 setup"
    FAILURES=$((FAILURES + 1))
  else
    echo "  PASS: Found 3 cosigner peers"
  fi
done

echo
if [ "$FAILURES" -gt 0 ]; then
  echo "=== VERIFICATION FAILED: $FAILURES issues found ==="
  exit 1
else
  echo "=== VERIFICATION SUCCESSFUL: 2-of-3 Horcrux ceremony configs are valid ==="
  exit 0
fi
