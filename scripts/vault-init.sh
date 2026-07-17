#!/usr/bin/env bash

# ═══════════════════════════════════════════════════════════════════════
# HashiCorp Vault Production Bootstrapping, Seeding & Unsealing Script
# ═══════════════════════════════════════════════════════════════════════

set -euo pipefail

export VAULT_ADDR="http://127.0.0.1:8200"
KEYS_FILE=".vault-keys.json"

echo "Waiting for Vault to respond..."
# Note: vault status returns exit code 2 when sealed, 0 when unsealed, but fails/hangs if vault is down.
until docker exec vault vault status >/dev/null 2>&1 || [ $? -eq 2 ]; do
    sleep 1
    echo -n "."
done
echo " Vault is online."

# 1. Initialize Vault if not already initialized
INIT_STATUS=$(docker exec vault vault status -format=json | jq -r '.initialized')

if [ "$INIT_STATUS" != "true" ]; then
    echo "Initializing Vault..."
    INIT_OUT=$(docker exec vault vault operator init -format=json)
    
    # Save unseal keys and root token to local gitignored file
    echo "$INIT_OUT" > "$KEYS_FILE"
    chmod 600 "$KEYS_FILE"
    echo "Vault initialized. Keys stored in $KEYS_FILE"
else
    echo "Vault is already initialized."
    if [ ! -f "$KEYS_FILE" ]; then
        echo "ERROR: Vault is initialized but $KEYS_FILE is missing!"
        exit 1
    fi
fi

# Load root token and unseal keys
ROOT_TOKEN=$(jq -r '.root_token' "$KEYS_FILE")
export VAULT_TOKEN="$ROOT_TOKEN"

# 2. Unseal Vault if sealed
SEALED_STATUS=$(docker exec vault vault status -format=json | jq -r '.sealed')
if [ "$SEALED_STATUS" == "true" ]; then
    echo "Vault is sealed. Unsealing..."
    # Read first 3 keys to reach default threshold
    KEYS=$(jq -r '.unseal_keys_b64[0,1,2]' "$KEYS_FILE")
    for key in $KEYS; do
        docker exec vault vault operator unseal "$key" > /dev/null
    done
    echo "Vault unsealed."
else
    echo "Vault is already unsealed."
fi

echo "Enabling kv-v2 engine at secret/..."
docker exec -e VAULT_TOKEN="$ROOT_TOKEN" vault vault secrets enable -path=secret kv-v2 || true

echo "Seeding database credentials..."
docker exec -e VAULT_TOKEN="$ROOT_TOKEN" vault vault kv put secret/sovereign/database \
    write_db_pass="sovereign_write_pwd" \
    read_db_pass="sovereign_read_pwd" \
    relayer_db_pass="sovereign_relayer_pwd"

echo "Seeding NATS credentials (real NKeys)..."
docker exec -e VAULT_TOKEN="$ROOT_TOKEN" vault vault kv put secret/sovereign/nats \
    ingestion_nkey="SUAFFNTD6H6ST7VGTZDXYQDC5BPNGYRTEFY4TZM32TJEMBTFN5TJO4WNXU" \
    projection_nkey="SUAINVHHXAR4PZTQC4VEME4P3HB2CQ3QNQY4WK3YNULE2IJZLNOLNDGBUE" \
    stream_nkey="SUAO6IIZLMQHQYVKKHJIEXIC5T6XNKM2PUVF4EGZW23UALD7WTFFE7R2LQ" \
    bridge_nkey="U-bridge-nkey-mock-56789"

echo "Seeding cryptographic keys..."
docker exec -e VAULT_TOKEN="$ROOT_TOKEN" vault vault kv put secret/sovereign/keys \
    validator_key="mock-validator-private-key-hex-000000001" \
    witness_key="mock-witness-private-key-hex-000000002"

echo "Retrieving verification read..."
docker exec -e VAULT_TOKEN="$ROOT_TOKEN" vault vault kv get secret/sovereign/database

echo "═══════════════════════════════════════════════════════════════════════"
echo "Vault successfully initialized, unsealed, and seeded!"
echo "ROOT TOKEN: $ROOT_TOKEN"
echo "═══════════════════════════════════════════════════════════════════════"
