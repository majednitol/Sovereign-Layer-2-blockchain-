-- Explorer DB (new schema on shared Read DB)
CREATE SCHEMA IF NOT EXISTS explorer;

-- PHASE 1: Core Tables
CREATE TABLE IF NOT EXISTS explorer.blocks (
  height        BIGINT PRIMARY KEY,
  time          TIMESTAMPTZ NOT NULL,
  proposer      TEXT NOT NULL,         -- bech32
  tx_count      INT NOT NULL,
  gas_used      BIGINT,
  gas_limit     BIGINT,
  app_hash      TEXT
);

CREATE TABLE IF NOT EXISTS explorer.transactions (
  hash          TEXT PRIMARY KEY,
  height        BIGINT NOT NULL REFERENCES explorer.blocks(height) ON DELETE CASCADE,
  time          TIMESTAMPTZ NOT NULL,
  type          TEXT NOT NULL,         -- 'cosmos' | 'evm' | 'cosmwasm' | 'bridge' | 'oracle'
  msg_types     TEXT[] NOT NULL,       -- e.g. ['/cosmos.staking.v1beta1.MsgDelegate']
  decoded       JSONB,                 -- all msg fields
  fee           BIGINT,
  gas_used      BIGINT,
  status        SMALLINT NOT NULL      -- 0=success, 1=failed
);

CREATE TABLE IF NOT EXISTS explorer.accounts (
  address_bech32  TEXT PRIMARY KEY,
  address_hex     TEXT,
  first_seen      BIGINT,              -- block height
  last_active     BIGINT               -- block height
);

CREATE INDEX IF NOT EXISTS idx_transactions_height ON explorer.transactions(height);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON explorer.transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_msg_types ON explorer.transactions USING GIN (msg_types);

-- PHASE 2: Custom Modules, Governance, IBC, CosmWasm
CREATE TABLE IF NOT EXISTS explorer.validator_slots (
  slot_index        INT PRIMARY KEY,
  validator_address TEXT NOT NULL,
  power             BIGINT NOT NULL,
  status            TEXT NOT NULL,
  missed_blocks     BIGINT NOT NULL,
  certification_score INT NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.slot_events (
  id                SERIAL PRIMARY KEY,
  event_type        TEXT NOT NULL, -- 'filled' | 'ejected' | 'slashed'
  slot_index        INT NOT NULL,
  validator         TEXT NOT NULL,
  height            BIGINT NOT NULL,
  reason            TEXT,
  time              TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.certification_scores (
  address           TEXT PRIMARY KEY,
  attestation_score INT NOT NULL,
  window_size       INT NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.oracle_rounds (
  round_id          BIGINT NOT NULL,
  feed_id           TEXT NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL,
  aggregated_median NUMERIC,
  status            TEXT NOT NULL, -- 'commit' | 'reveal' | 'done'
  PRIMARY KEY (round_id, feed_id)
);

CREATE TABLE IF NOT EXISTS explorer.oracle_commits (
  round_id          BIGINT NOT NULL,
  feed_id           TEXT NOT NULL,
  validator         TEXT NOT NULL,
  hash              TEXT NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (round_id, feed_id, validator)
);

CREATE TABLE IF NOT EXISTS explorer.oracle_reveals (
  round_id          BIGINT NOT NULL,
  feed_id           TEXT NOT NULL,
  validator         TEXT NOT NULL,
  value             NUMERIC NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (round_id, feed_id, validator)
);

CREATE TABLE IF NOT EXISTS explorer.milestones (
  id                BIGINT PRIMARY KEY,
  creator           TEXT NOT NULL,
  status            TEXT NOT NULL, -- 'pending' | 'stale-blocked' | 'achieved' | 'expired'
  title             TEXT NOT NULL,
  target_price      NUMERIC NOT NULL,
  feed_id           TEXT NOT NULL,
  achieved_height   BIGINT,
  expired_height    BIGINT,
  total_paused_duration BIGINT -- in seconds
);

CREATE TABLE IF NOT EXISTS explorer.milestone_events (
  id                SERIAL PRIMARY KEY,
  milestone_id      BIGINT NOT NULL REFERENCES explorer.milestones(id) ON DELETE CASCADE,
  height            BIGINT NOT NULL,
  event_type        TEXT NOT NULL, -- 'created' | 'transitioned' | 'paused' | 'resumed'
  value             TEXT,
  time              TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.settlements (
  id                BIGINT PRIMARY KEY,
  witness           TEXT NOT NULL,
  status            TEXT NOT NULL, -- 'pending' | 'settled' | 'failed'
  chain_id          TEXT NOT NULL,
  tx_hash           TEXT NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL,
  witness_signatures JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.contracts (
  address           TEXT PRIMARY KEY,
  code_id           BIGINT NOT NULL,
  label             TEXT NOT NULL,
  creator           TEXT NOT NULL,
  admin             TEXT,
  type_badge        TEXT,
  execute_history   JSONB
);


CREATE INDEX IF NOT EXISTS idx_slot_events_slot_index ON explorer.slot_events(slot_index);
CREATE INDEX IF NOT EXISTS idx_oracle_rounds_feed_id ON explorer.oracle_rounds(feed_id);
CREATE INDEX IF NOT EXISTS idx_milestone_events_milestone_id ON explorer.milestone_events(milestone_id);

-- PHASE 3: Bridge Tracking & BSC watcher
CREATE TABLE IF NOT EXISTS explorer.bridge_txs (
  id                BIGSERIAL PRIMARY KEY,
  direction         TEXT NOT NULL, -- 'deposit' (BSC -> Cosmos) | 'withdraw' (Cosmos -> BSC)
  nonce             BIGINT NOT NULL,
  status            TEXT NOT NULL, -- 'locked' | 'confirming' | 'confirmed' | 'minted' | 'released'
  source_hash       TEXT NOT NULL,
  dest_hash         TEXT,
  amount            NUMERIC NOT NULL,
  sender            TEXT NOT NULL,
  receiver          TEXT NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.relayers (
  address           TEXT PRIMARY KEY,
  status            TEXT NOT NULL, -- 'Primary' | 'Secondary' | 'Candidate'
  last_active       BIGINT NOT NULL,
  miss_count        INT NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.circuit_breaker_events (
  height            BIGINT PRIMARY KEY,
  event_type        TEXT NOT NULL, -- 'pause' | 'unpause'
  trigger_address   TEXT NOT NULL,
  time              TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS explorer.bsc_lock_events (
  tx_hash           TEXT PRIMARY KEY,
  sender            TEXT NOT NULL,
  amount            NUMERIC NOT NULL,
  nonce             BIGINT NOT NULL,
  status            TEXT NOT NULL, -- 'locked' | 'confirming' | 'confirmed' | 'minted'
  time              TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_bridge_txs_nonce ON explorer.bridge_txs(nonce);
CREATE INDEX IF NOT EXISTS idx_bridge_txs_status ON explorer.bridge_txs(status);
CREATE INDEX IF NOT EXISTS idx_bsc_lock_events_nonce ON explorer.bsc_lock_events(nonce);

-- PHASE 4: Hardening, Webhooks, Trigram Search
CREATE TABLE IF NOT EXISTS explorer.webhooks (
  id                BIGSERIAL PRIMARY KEY,
  url               TEXT NOT NULL,
  address           TEXT NOT NULL,
  secret            TEXT NOT NULL,
  events            TEXT[] NOT NULL,
  created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_accounts_bech32_trgm ON explorer.accounts USING gin (address_bech32 gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_accounts_hex_trgm ON explorer.accounts USING gin (address_hex gin_trgm_ops);
