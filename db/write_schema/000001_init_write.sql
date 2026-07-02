-- Write-side schema: partitioned event store

CREATE TABLE IF NOT EXISTS events (
    block_height BIGINT NOT NULL,
    event_index INT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    nats_published BOOLEAN DEFAULT false,
    PRIMARY KEY (block_height, event_index)
) PARTITION BY RANGE (block_height);

-- Default partition to catch all inserts if specific partition is not defined
CREATE TABLE IF NOT EXISTS events_default PARTITION OF events DEFAULT;

-- Index for backfilling unpublished events on reconnect
CREATE INDEX IF NOT EXISTS idx_events_nats_published 
ON events (nats_published) 
WHERE nats_published = false;
