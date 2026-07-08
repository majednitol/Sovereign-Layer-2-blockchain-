-- Wave 2 Database Migrations: Analytics aggregates, account fields, and EVM properties

-- Alter verified_evm_contracts to support proxy and vault details
ALTER TABLE explorer.verified_evm_contracts ADD COLUMN IF NOT EXISTS is_proxy BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE explorer.verified_evm_contracts ADD COLUMN IF NOT EXISTS implementation_address TEXT CHECK (implementation_address IS NULL OR implementation_address ~ '^0x[a-fA-F0-9]{40}$');
ALTER TABLE explorer.verified_evm_contracts ADD COLUMN IF NOT EXISTS is_vault BOOLEAN NOT NULL DEFAULT FALSE;

-- Alter accounts table to support balance and tx_count
ALTER TABLE explorer.accounts ADD COLUMN IF NOT EXISTS balance NUMERIC NOT NULL DEFAULT 0;
ALTER TABLE explorer.accounts ADD COLUMN IF NOT EXISTS tx_count BIGINT NOT NULL DEFAULT 0;

-- Daily Network Statistics Aggregation Table
CREATE TABLE IF NOT EXISTS explorer.daily_network_stats (
  date              DATE PRIMARY KEY,
  tx_count          BIGINT NOT NULL DEFAULT 0,
  gas_used          BIGINT NOT NULL DEFAULT 0,
  active_accounts   BIGINT NOT NULL DEFAULT 0,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Daily Bridge Volume Aggregation Table
CREATE TABLE IF NOT EXISTS explorer.daily_bridge_volume (
  date              DATE PRIMARY KEY,
  deposit_volume    NUMERIC NOT NULL DEFAULT 0,
  withdraw_volume   NUMERIC NOT NULL DEFAULT 0,
  deposit_count     INT NOT NULL DEFAULT 0,
  withdraw_count    INT NOT NULL DEFAULT 0,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Daily IBC Volume Aggregation Table
CREATE TABLE IF NOT EXISTS explorer.daily_ibc_volume (
  date              DATE PRIMARY KEY,
  inbound_volume    NUMERIC NOT NULL DEFAULT 0,
  outbound_volume   NUMERIC NOT NULL DEFAULT 0,
  inbound_count     INT NOT NULL DEFAULT 0,
  outbound_count    INT NOT NULL DEFAULT 0,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexing for quick analytics sorting
CREATE INDEX IF NOT EXISTS idx_daily_network_stats_date ON explorer.daily_network_stats(date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_bridge_volume_date ON explorer.daily_bridge_volume(date DESC);
CREATE INDEX IF NOT EXISTS idx_daily_ibc_volume_date ON explorer.daily_ibc_volume(date DESC);

-- Unique index required for REFRESH MATERIALIZED VIEW CONCURRENTLY
CREATE UNIQUE INDEX IF NOT EXISTS idx_search_index_type_id ON explorer.search_index (type, id);
