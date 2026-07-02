-- Apply TimescaleDB to Read DB — migration 002

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Block-level statistics written by module/projection on every block
CREATE TABLE IF NOT EXISTS block_stats (
  block_height   BIGINT    NOT NULL,
  block_time_ms  INT       NOT NULL,
  tx_count       INT       NOT NULL,
  avg_fee_uatom  NUMERIC   NOT NULL,
  PRIMARY KEY (block_height)
);
SELECT create_hypertable('block_stats', 'block_height', chunk_time_interval => 432000, if_not_exists => TRUE);

-- Oracle price submissions written by module/projection on every oracle event
CREATE TABLE IF NOT EXISTS oracle_submissions (
  block_height  BIGINT   NOT NULL,
  asset_id      TEXT     NOT NULL,
  price         NUMERIC  NOT NULL,
  validator     TEXT     NOT NULL,
  PRIMARY KEY (block_height, asset_id, validator)
);
SELECT create_hypertable('oracle_submissions', 'block_height', chunk_time_interval => 432000, if_not_exists => TRUE);

-- Validator signature records written by module/projection on every block
CREATE TABLE IF NOT EXISTS validator_signatures (
  block_height       BIGINT   NOT NULL,
  validator_address  TEXT     NOT NULL,
  signed             BOOLEAN  NOT NULL,
  PRIMARY KEY (block_height, validator_address)
);
SELECT create_hypertable('validator_signatures', 'block_height', chunk_time_interval => 432000, if_not_exists => TRUE);

-- Bridge events written by module/projection on every bridge event
CREATE TABLE IF NOT EXISTS bridge_events (
  block_height  BIGINT   NOT NULL,
  event_index   INT      NOT NULL,
  direction     TEXT     NOT NULL,   -- 'lock' | 'release'
  asset         TEXT     NOT NULL,
  amount        NUMERIC  NOT NULL,
  PRIMARY KEY (block_height, event_index)
);
SELECT create_hypertable('bridge_events', 'block_height', chunk_time_interval => 432000, if_not_exists => TRUE);

-- Define integer now function for block-height based hypertables (required for continuous aggregates)
CREATE OR REPLACE FUNCTION public.current_block_height() RETURNS bigint AS $$
  SELECT COALESCE(MAX(block_height), 0)::bigint FROM public.block_stats;
$$ LANGUAGE SQL STABLE;

SELECT set_integer_now_func('block_stats', 'public.current_block_height', replace_if_exists => TRUE);
SELECT set_integer_now_func('oracle_submissions', 'public.current_block_height', replace_if_exists => TRUE);
SELECT set_integer_now_func('validator_signatures', 'public.current_block_height', replace_if_exists => TRUE);
SELECT set_integer_now_func('bridge_events', 'public.current_block_height', replace_if_exists => TRUE);

-- TPS: average and peak transactions per second per hour
CREATE MATERIALIZED VIEW IF NOT EXISTS tps_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         COUNT(*)::FLOAT / 600.0 / 6.0         AS tps_avg,
         MAX(tx_count)::FLOAT / 6.0            AS tps_peak,
         SUM(tx_count)                         AS total_txs
  FROM block_stats
  GROUP BY period
WITH NO DATA;

SELECT add_continuous_aggregate_policy('tps_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes', if_not_exists => TRUE);

-- Block time statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS block_time_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         AVG(block_time_ms)                    AS avg_ms,
         MAX(block_time_ms)                    AS max_ms
  FROM block_stats
  GROUP BY period
WITH NO DATA;

SELECT add_continuous_aggregate_policy('block_time_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes', if_not_exists => TRUE);

-- Oracle OHLC candles per asset per hour
CREATE MATERIALIZED VIEW IF NOT EXISTS oracle_price_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         asset_id,
         first(price, block_height)            AS open,
         max(price)                            AS high,
         min(price)                            AS low,
         last(price, block_height)             AS close,
         count(*)                              AS submission_count
  FROM oracle_submissions
  GROUP BY period, asset_id
WITH NO DATA;

SELECT add_continuous_aggregate_policy('oracle_price_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes', if_not_exists => TRUE);

-- Validator uptime percentage per day
CREATE MATERIALIZED VIEW IF NOT EXISTS validator_uptime_1d
WITH (timescaledb.continuous) AS
  SELECT time_bucket(14400, block_height)      AS period,
         validator_address,
         AVG(signed::INT)::FLOAT * 100         AS uptime_pct,
         COUNT(*)                              AS blocks_in_window
  FROM validator_signatures
  GROUP BY period, validator_address
WITH NO DATA;

SELECT add_continuous_aggregate_policy('validator_uptime_1d',
  start_offset => 43200, end_offset => 14400, schedule_interval => INTERVAL '1 hour', if_not_exists => TRUE);

-- Bridge volume per direction per asset per hour
CREATE MATERIALIZED VIEW IF NOT EXISTS bridge_volume_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         direction,
         asset,
         SUM(amount)                           AS volume,
         COUNT(*)                              AS tx_count
  FROM bridge_events
  GROUP BY period, direction, asset
WITH NO DATA;

SELECT add_continuous_aggregate_policy('bridge_volume_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes', if_not_exists => TRUE);
