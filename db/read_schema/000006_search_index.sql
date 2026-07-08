-- Materialized view for global search_index across explorer schema
CREATE SCHEMA IF NOT EXISTS explorer;

CREATE MATERIALIZED VIEW IF NOT EXISTS explorer.search_index AS
  SELECT 'block' AS type, height::TEXT AS id, height::TEXT AS label
  FROM explorer.blocks
  
  UNION ALL
  
  SELECT 'tx' AS type, hash AS id, hash AS label
  FROM explorer.transactions
  
  UNION ALL
  
  SELECT 'account' AS type, address_bech32 AS id, COALESCE(address_hex, address_bech32) AS label
  FROM explorer.accounts
  
  UNION ALL
  
  SELECT 'contract' AS type, address AS id, label AS label
  FROM explorer.contracts
  
  UNION ALL
  
  SELECT 'validator' AS type, validator_address AS id, validator_address AS label
  FROM explorer.validator_slots;

-- Indexes for trigram fuzzy search
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_search_index_label_trgm ON explorer.search_index USING gin (label gin_trgm_ops);

-- Ensure validator_slots table columns are fully ready
ALTER TABLE explorer.validator_slots ADD COLUMN IF NOT EXISTS moniker TEXT;
ALTER TABLE explorer.validator_slots ADD COLUMN IF NOT EXISTS operator_address TEXT;
ALTER TABLE explorer.validator_slots ADD COLUMN IF NOT EXISTS consensus_pubkey TEXT;

-- Governance votes table for tracking votes
CREATE TABLE IF NOT EXISTS explorer.governance_votes (
  proposal_id       BIGINT NOT NULL,
  validator_address TEXT NOT NULL,
  option            TEXT NOT NULL, -- 'yes' | 'no' | 'abstain' | 'no_with_veto'
  weight            NUMERIC NOT NULL,
  height            BIGINT NOT NULL,
  time              TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (proposal_id, validator_address)
);

-- Contract schemas table for query/execute forms JSON definitions
CREATE TABLE IF NOT EXISTS explorer.contract_schemas (
  code_id           BIGINT PRIMARY KEY,
  query_schema      JSONB,
  execute_schema    JSONB,
  uploaded_by       TEXT,
  uploaded_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
