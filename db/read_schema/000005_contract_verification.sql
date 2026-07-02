-- EVM Verified Contracts Table
CREATE TABLE IF NOT EXISTS explorer.verified_evm_contracts (
  address            TEXT PRIMARY KEY CHECK (address ~ '^0x[a-fA-F0-9]{40}$'),
  verified           BOOLEAN NOT NULL DEFAULT TRUE,
  compiler_version   TEXT NOT NULL,
  source_code        TEXT NOT NULL,
  abi                JSONB NOT NULL,
  optimizer_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
  optimizer_runs     INT NOT NULL DEFAULT 200,
  constructor_args   TEXT, -- hex-encoded constructor args
  match_type         TEXT NOT NULL CHECK (match_type IN ('perfect', 'partial')),
  created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- CosmWasm Verified Codes & Build Metadata
CREATE TABLE IF NOT EXISTS explorer.verified_codes (
  code_id            BIGINT PRIMARY KEY,
  verified           BOOLEAN NOT NULL DEFAULT TRUE,
  checksum           TEXT NOT NULL CHECK (checksum ~ '^[a-f0-9]{64}$'),
  instantiate_msg    JSONB NOT NULL,
  execute_msg        JSONB NOT NULL,
  query_msg          JSONB NOT NULL,
  git_repo           TEXT, -- Git URL
  git_commit         TEXT, -- Commit hash
  optimizer_version  TEXT, -- cosmwasm/rust-optimizer docker tag version
  created_at         TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_verified_evm_contracts_addr ON explorer.verified_evm_contracts(address);
CREATE INDEX IF NOT EXISTS idx_verified_codes_id ON explorer.verified_codes(code_id);
