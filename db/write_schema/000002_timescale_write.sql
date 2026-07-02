-- Apply TimescaleDB to Write DB — migration 002

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Drop declarative partition tables if they exist to allow hypertable initialization
DROP TABLE IF EXISTS events_default CASCADE;
DROP TABLE IF EXISTS events CASCADE;

-- Recreate events table with created_at time dimension
CREATE TABLE IF NOT EXISTS events (
    block_height    BIGINT                   NOT NULL,
    event_index     INT                      NOT NULL,
    event_type      TEXT                     NOT NULL,
    payload         JSONB                    NOT NULL,
    nats_published  BOOLEAN                  DEFAULT false,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (block_height, event_index, created_at)
);

-- Initialize events table as a hypertable using created_at with a 7-day interval
SELECT create_hypertable('events', 'created_at', chunk_time_interval => INTERVAL '7 days', if_not_exists => TRUE);

-- Recreate index for backfilling unpublished events on reconnect
CREATE INDEX IF NOT EXISTS idx_events_nats_published 
ON events (nats_published) 
WHERE nats_published = false;

-- Enable TimescaleDB compression on events table
ALTER TABLE events SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'event_type'
);

-- Add compression policy for chunks older than 60 days
SELECT add_compression_policy('events', INTERVAL '60 days', if_not_exists => TRUE);
