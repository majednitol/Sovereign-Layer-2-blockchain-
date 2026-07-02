-- Redefine block_time_1h continuous aggregate using native aggregates to prevent toolkit segfault on ARM64

-- Drop existing continuous aggregate view to redefine it
DROP MATERIALIZED VIEW IF EXISTS block_time_1h CASCADE;

-- Recreate block_time_1h
CREATE MATERIALIZED VIEW IF NOT EXISTS block_time_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         AVG(block_time_ms)                    AS avg_ms,
         MAX(block_time_ms)                    AS max_ms
  FROM block_stats
  GROUP BY period
  WITH NO DATA;

-- Re-apply policy to refresh the aggregate view periodically
SELECT add_continuous_aggregate_policy('block_time_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes', if_not_exists => TRUE);
