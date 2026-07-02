#!/usr/bin/env bash

# ═══════════════════════════════════════════════════════════════════════
# HashiCorp Vault Bootstrapping & Seeding Script
# ═══════════════════════════════════════════════════════════════════════

set -euo pipefail

export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"

echo "Waiting for Vault to respond and be ready..."
until docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root vault vault status > /dev/null 2>&1; do
    sleep 1
    echo -n "."
done
echo " Vault is ready."

echo "Enabling kv-v2 engine at secret/..."
# In dev mode, kv-v2 might be enabled at secret/ by default. We check and ignore error if already enabled.
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root vault vault secrets enable -path=secret kv-v2 || true

echo "Seeding database credentials..."
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root vault vault kv put secret/sovereign/database \
    write_db_pass="sovereign_write_pwd" \
    read_db_pass="sovereign_read_pwd" \
    relayer_db_pass="sovereign_relayer_pwd"

echo "Seeding NATS credentials (real NKeys)..."
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root vault vault kv put secret/sovereign/nats \
    ingestion_nkey="SUAFFNTD6H6ST7VGTZDXYQDC5BPNGYRTEFY4TZM32TJEMBTFN5TJO4WNXU" \
    projection_nkey="SUAINVHHXAR4PZTQC4VEME4P3HB2CQ3QNQY4WK3YNULE2IJZLNOLNDGBUE" \
    stream_nkey="SUAO6IIZLMQHQYVKKHJIEXIC5T6XNKM2PUVF4EGZW23UALD7WTFFE7R2LQ" \
    bridge_nkey="U-bridge-nkey-mock-56789"

echo "Seeding cryptographic keys..."
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root vault vault kv put secret/sovereign/keys \
    validator_key="mock-validator-private-key-hex-000000001" \
    witness_key="mock-witness-private-key-hex-000000002"

echo "Retrieving verification read..."
docker exec -e VAULT_ADDR=http://127.0.0.1:8200 -e VAULT_TOKEN=root vault vault kv get secret/sovereign/database

echo "═══════════════════════════════════════════════════════════════════════"
echo "Vault successfully initialized and seeded with Phase 0 secrets!"
echo "═══════════════════════════════════════════════════════════════════════"
