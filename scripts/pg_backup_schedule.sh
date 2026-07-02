#!/usr/bin/env bash

# ═══════════════════════════════════════════════════════════════════════
# PostgreSQL Daily Base Backup & Retention Script
# ═══════════════════════════════════════════════════════════════════════

set -euo pipefail

# Configuration
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5434}"  # Read DB local port
DB_USER="${DB_USER:-api_reader}"
export PGPASSWORD="${DB_PASSWORD:-sovereign_read_pwd}"

BACKUP_DIR="${BACKUP_DIR:-./db_backups}"
RETENTION_DAYS=7

DATE_SUFFIX=$(date +%Y-%m-%d_%H-%M-%S)
CURRENT_BACKUP_PATH="${BACKUP_DIR}/backup_${DATE_SUFFIX}"

echo "Starting PostgreSQL base backup..."
echo "Target: ${DB_USER}@${DB_HOST}:${DB_PORT}"
echo "Destination: ${CURRENT_BACKUP_PATH}"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

# Run pg_basebackup (tar format, compressed)
if pg_basebackup -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" \
  -D "${CURRENT_BACKUP_PATH}" -Ft -z -P -X stream; then
  echo "Backup successfully completed at ${CURRENT_BACKUP_PATH}."
else
  echo "ERROR: Backup failed!" >&2
  exit 1
fi

# Apply retention policy
echo "Enforcing backup retention policy (deleting backups older than ${RETENTION_DAYS} days)..."
find "${BACKUP_DIR}" -type d -name "backup_*" -mtime +${RETENTION_DAYS} -exec rm -rf {} \; -print

echo "Backup execution finished successfully."
