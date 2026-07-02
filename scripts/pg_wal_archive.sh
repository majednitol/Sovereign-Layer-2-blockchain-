#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════
# PostgreSQL WAL Archive Script for PITR (Point-in-Time Recovery)
# ═══════════════════════════════════════════════════════════════════════
set -euo pipefail

# This script is called by PostgreSQL's archive_command parameter:
# archive_command = 'scripts/pg_wal_archive.sh %p %f'

WAL_PATH="${1}"
WAL_FILE="${2}"
S3_BUCKET="${S3_BUCKET:-sovereign-mainnet-backups}"
LOG_FILE="${LOG_FILE:-/var/log/postgresql/wal_archive.log}"

# Ensure log directory exists
mkdir -p "$(dirname "${LOG_FILE}")"

echo "[$(date +'%Y-%m-%d %H:%M:%S')] Archiving WAL segment ${WAL_FILE}..." >> "${LOG_FILE}"

# In production, we run the aws s3 cp command:
# aws s3 cp "${WAL_PATH}" "s3://${S3_BUCKET}/wal/${WAL_FILE}"
# For testing/fallback, we copy to a local directory representing S3 mount:
LOCAL_S3_MOCK="./db_backups/s3_mock/wal"
mkdir -p "${LOCAL_S3_MOCK}"

if cp "${WAL_PATH}" "${LOCAL_S3_MOCK}/${WAL_FILE}"; then
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: Archived ${WAL_FILE}" >> "${LOG_FILE}"
  exit 0
else
  echo "[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: Failed to archive ${WAL_FILE}" >> "${LOG_FILE}"
  exit 1
fi
