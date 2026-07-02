-- Read-side schema: denormalized metrics and query tables

CREATE TABLE IF NOT EXISTS bridge_volume (
    token_address VARCHAR(42) NOT NULL,
    chain_id VARCHAR(64) NOT NULL,
    timeframe VARCHAR(10) NOT NULL, -- e.g. "hourly", "daily", "all"
    bucket_time TIMESTAMP NOT NULL,
    total_minted NUMERIC(78, 0) DEFAULT 0,
    total_burned NUMERIC(78, 0) DEFAULT 0,
    volume_usd NUMERIC(18, 4) DEFAULT 0.0,
    transaction_count BIGINT DEFAULT 0,
    PRIMARY KEY (token_address, chain_id, timeframe, bucket_time)
);

CREATE TABLE IF NOT EXISTS validator_uptime (
    validator_address VARCHAR(42) PRIMARY KEY,
    total_blocks BIGINT DEFAULT 0,
    missed_blocks BIGINT DEFAULT 0,
    uptime_percentage DOUBLE PRECISION DEFAULT 100.0
);

CREATE TABLE IF NOT EXISTS oracle_participation (
    oracle_address VARCHAR(42) PRIMARY KEY,
    total_requests BIGINT DEFAULT 0,
    successful_reveals BIGINT DEFAULT 0,
    participation_rate DOUBLE PRECISION DEFAULT 100.0
);

CREATE TABLE IF NOT EXISTS settlements (
    settlement_id VARCHAR(66) PRIMARY KEY,
    proof BYTEA NOT NULL,
    status VARCHAR(20) NOT NULL,
    block_height BIGINT NOT NULL,
    signatures TEXT[] DEFAULT '{}'::TEXT[]
);

CREATE TABLE IF NOT EXISTS milestone_status (
    milestone_id VARCHAR(66) PRIMARY KEY,
    status VARCHAR(20) NOT NULL,
    block_height BIGINT NOT NULL
);

CREATE TABLE IF NOT EXISTS bridge_pending (
    nonce BIGINT PRIMARY KEY,
    token_address VARCHAR(42) NOT NULL,
    amount NUMERIC(78, 0) NOT NULL,
    recipient VARCHAR(42) NOT NULL,
    status VARCHAR(20) NOT NULL
);
