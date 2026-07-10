-- Alter explorer.contracts table to add token info
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS token_name TEXT;
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS token_symbol TEXT;
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS decimals INT;
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS total_supply NUMERIC(78,0);
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS minter_address TEXT;
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS owner_address TEXT;
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE explorer.contracts ADD COLUMN IF NOT EXISTS metadata_status TEXT NOT NULL DEFAULT 'pending';

-- Create table explorer.evm_token_transfers
CREATE TABLE IF NOT EXISTS explorer.evm_token_transfers (
  tx_hash           TEXT NOT NULL,
  log_index         INT NOT NULL,
  block_height      BIGINT NOT NULL,
  block_time        TIMESTAMPTZ NOT NULL,
  token_address     TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  from_address      TEXT NOT NULL,
  to_address        TEXT NOT NULL,
  value             NUMERIC(78, 0) NOT NULL,
  token_standard    TEXT NOT NULL, -- 'ERC20' | 'ERC721' | 'ERC1155' | 'ERC4626'
  token_id          NUMERIC(78, 0),
  PRIMARY KEY (tx_hash, log_index)
);
CREATE INDEX IF NOT EXISTS idx_evm_transfers_token ON explorer.evm_token_transfers(token_address);
CREATE INDEX IF NOT EXISTS idx_evm_transfers_from ON explorer.evm_token_transfers(from_address);
CREATE INDEX IF NOT EXISTS idx_evm_transfers_to ON explorer.evm_token_transfers(to_address);

-- Create table explorer.evm_vault_events
CREATE TABLE IF NOT EXISTS explorer.evm_vault_events (
  tx_hash                 TEXT NOT NULL,
  log_index               INT NOT NULL,
  vault_address           TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  underlying_asset_address TEXT NOT NULL,
  sender                  TEXT NOT NULL,
  owner                   TEXT NOT NULL,
  assets                  NUMERIC(78, 0) NOT NULL,
  shares                  NUMERIC(78, 0) NOT NULL,
  event_type              TEXT NOT NULL, -- 'deposit' | 'withdraw'
  PRIMARY KEY (tx_hash, log_index)
);

-- Create table explorer.evm_token_holders
CREATE TABLE IF NOT EXISTS explorer.evm_token_holders (
  token_address     TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  holder_address    TEXT NOT NULL,
  balance           NUMERIC(78, 0) NOT NULL DEFAULT 0,
  PRIMARY KEY (token_address, holder_address)
);
CREATE INDEX IF NOT EXISTS idx_evm_holders_bal ON explorer.evm_token_holders(token_address, balance DESC);

-- Create table explorer.evm_nft_ownership
CREATE TABLE IF NOT EXISTS explorer.evm_nft_ownership (
  token_address     TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  token_id          NUMERIC(78, 0) NOT NULL,
  owner_address     TEXT NOT NULL,
  token_uri         TEXT,
  metadata_json     JSONB,
  PRIMARY KEY (token_address, token_id)
);

-- Create table explorer.cw_token_transfers
CREATE TABLE IF NOT EXISTS explorer.cw_token_transfers (
  id                BIGSERIAL PRIMARY KEY,
  tx_hash           TEXT NOT NULL,
  block_height      BIGINT NOT NULL,
  block_time        TIMESTAMPTZ NOT NULL,
  contract_address  TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  from_address      TEXT NOT NULL,
  to_address        TEXT NOT NULL,
  amount            NUMERIC(78, 0) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cw_transfers_contract ON explorer.cw_token_transfers(contract_address);
CREATE INDEX IF NOT EXISTS idx_cw_transfers_from ON explorer.cw_token_transfers(from_address);
CREATE INDEX IF NOT EXISTS idx_cw_transfers_to ON explorer.cw_token_transfers(to_address);

-- Create table explorer.cw_token_holders
CREATE TABLE IF NOT EXISTS explorer.cw_token_holders (
  contract_address  TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  holder_address    TEXT NOT NULL,
  balance           NUMERIC(78, 0) NOT NULL DEFAULT 0,
  PRIMARY KEY (contract_address, holder_address)
);
CREATE INDEX IF NOT EXISTS idx_cw_holders_bal ON explorer.cw_token_holders(contract_address, balance DESC);

-- Create table explorer.cw_nft_ownership
CREATE TABLE IF NOT EXISTS explorer.cw_nft_ownership (
  contract_address  TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  token_id          TEXT NOT NULL,
  owner_address     TEXT NOT NULL,
  token_uri         TEXT,
  metadata_json     JSONB,
  PRIMARY KEY (contract_address, token_id)
);

-- Create table explorer.cw_nft_transfers
CREATE TABLE IF NOT EXISTS explorer.cw_nft_transfers (
  id                BIGSERIAL PRIMARY KEY,
  tx_hash           TEXT NOT NULL,
  block_height      BIGINT NOT NULL,
  block_time        TIMESTAMPTZ NOT NULL,
  contract_address  TEXT NOT NULL REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  from_address      TEXT NOT NULL,
  to_address        TEXT NOT NULL,
  token_id          TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_cw_nft_transfers_contract ON explorer.cw_nft_transfers(contract_address);

-- Create table explorer.contract_deployments
CREATE TABLE IF NOT EXISTS explorer.contract_deployments (
  address           TEXT PRIMARY KEY REFERENCES explorer.contracts(address) ON DELETE CASCADE,
  standard          TEXT NOT NULL, -- 'ERC20' | 'ERC721' | 'ERC1155' | 'ERC4626' | 'CW20' | 'CW721' | 'CW1155' | 'unknown'
  deployer          TEXT NOT NULL,
  tx_hash           TEXT NOT NULL,
  block_height      BIGINT NOT NULL,
  block_time        TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_deployments_height ON explorer.contract_deployments(block_height DESC);
