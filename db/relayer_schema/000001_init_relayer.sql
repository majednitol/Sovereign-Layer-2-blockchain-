-- Relayer database schema: tracks signatures, nonce states, and watch checkpoints

CREATE TABLE IF NOT EXISTS nonces (
    nonce TEXT PRIMARY KEY,
    state TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS votes (
    nonce TEXT,
    relayer TEXT,
    signature BYTEA,
    PRIMARY KEY (nonce, relayer)
);

CREATE TABLE IF NOT EXISTS checkpoints (
    key TEXT PRIMARY KEY,
    block_num BIGINT NOT NULL
);
