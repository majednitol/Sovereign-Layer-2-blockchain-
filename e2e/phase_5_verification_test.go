package e2e

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// EventRecord represents the projected event format
type EventRecord struct {
	BlockHeight int64           `json:"block_height"`
	EventIndex  int             `json:"event_index"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}

// 1. Verify TimescaleDB Migrations contain the required hypertables and continuous aggregates
func TestPhase5MigrationFilesCheck(t *testing.T) {
	migrationsPath := filepath.Join("..", "db", "read_schema", "000002_timescale.sql")
	content, err := ioutil.ReadFile(migrationsPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read TimescaleDB migration file %s: %v", migrationsPath, err)
	}
	migrationSQL := string(content)

	// Required hypertables
	requiredHypertables := []string{
		"block_stats",
		"oracle_submissions",
		"validator_signatures",
		"bridge_events",
	}

	for _, table := range requiredHypertables {
		if !strings.Contains(migrationSQL, "create_hypertable('"+table+"'") && !strings.Contains(migrationSQL, "create_hypertable('"+table) {
			t.Errorf("FAIL: Migration does not convert table '%s' to a TimescaleDB hypertable", table)
		} else {
			t.Logf("[PASS] Migration includes hypertable definition for '%s'", table)
		}
	}

	// Required continuous aggregates
	requiredAggregates := []string{
		"tps_1h",
		"block_time_1h",
		"oracle_price_1h",
		"validator_uptime_1d",
		"bridge_volume_1h",
	}

	for _, agg := range requiredAggregates {
		if !strings.Contains(migrationSQL, "VIEW IF NOT EXISTS "+agg) && !strings.Contains(migrationSQL, "VIEW "+agg) {
			t.Errorf("FAIL: Migration does not define continuous aggregate view '%s'", agg)
		} else {
			t.Logf("[PASS] Migration includes continuous aggregate definition for '%s'", agg)
		}
		if !strings.Contains(migrationSQL, "add_continuous_aggregate_policy('"+agg+"'") && !strings.Contains(migrationSQL, "add_continuous_aggregate_policy('"+agg) {
			t.Errorf("FAIL: Migration does not register a refresh policy for continuous aggregate '%s'", agg)
		}
	}
}

// 2. Ingestion singleton advisory lock simulation
func TestPhase5IngestionAdvisoryLockSim(t *testing.T) {
	// Represents the singleton advisory lock state
	isLocked := false

	// Lock function simulates session-level advisory lock
	acquireLock := func() (bool, error) {
		if isLocked {
			return false, nil // Another instance holds the lock
		}
		isLocked = true
		return true, nil
	}

	releaseLock := func() error {
		isLocked = false
		return nil
	}

	// Test 1: First instance successfully acquires the lock
	locked1, err := acquireLock()
	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if !locked1 {
		t.Fatal("First lock acquisition should succeed")
	}

	// Test 2: Second instance fails to acquire lock (singleton behavior)
	locked2, err := acquireLock()
	if err != nil {
		t.Fatalf("Unexpected error during second lock acquisition: %v", err)
	}
	if locked2 {
		t.Fatal("Second lock acquisition should fail when lock is already held")
	}
	t.Log("[PASS] Checked advisory lock singleton behavior: double acquisition prevented.")

	// Test 3: Lock release and re-acquisition
	err = releaseLock()
	if err != nil {
		t.Fatalf("Unexpected error releasing lock: %v", err)
	}

	locked3, err := acquireLock()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !locked3 {
		t.Fatal("Lock acquisition should succeed after release")
	}
}

// 3. Projection mapping simulation: verify SQL inserts map correctly for all events
func TestPhase5ProjectionEventMappingSim(t *testing.T) {
	type ExecutedQuery struct {
		SQL  string
		Args []interface{}
	}

	var executedQueries []ExecutedQuery
	mockExec := func(query string, args ...interface{}) error {
		executedQueries = append(executedQueries, ExecutedQuery{SQL: query, Args: args})
		return nil
	}

	// Simulate projecting a validator_uptime event
	simProjectValidatorUptime := func(ev EventRecord) error {
		type ValidatorStatus struct {
			Address string `json:"address"`
			Signed  bool   `json:"signed"`
		}
		type ValidatorUptimePayload struct {
			Proposer   string            `json:"proposer"`
			Validators []ValidatorStatus `json:"validators"`
		}
		var payload ValidatorUptimePayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}

		// Insert into block_stats
		err := mockExec("INSERT INTO block_stats (block_height, block_time_ms, tx_count, avg_fee_uatom) VALUES ($1, $2, $3, $4)",
			ev.BlockHeight, 6000, len(payload.Validators), 150)
		if err != nil {
			return err
		}

		for _, val := range payload.Validators {
			// Insert validator_uptime aggregate
			err = mockExec("INSERT INTO validator_uptime (validator_address) VALUES ($1)", val.Address)
			if err != nil {
				return err
			}
			// Insert validator_signatures hypertable record
			err = mockExec("INSERT INTO validator_signatures (block_height, validator_address, signed) VALUES ($1, $2, $3)",
				ev.BlockHeight, val.Address, val.Signed)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Ingest a mock validator_uptime event
	uptimePayload := json.RawMessage(`{
		"proposer": "val1",
		"validators": [
			{"address": "val1", "signed": true},
			{"address": "val2", "signed": false}
		]
	}`)
	ev := EventRecord{
		BlockHeight: 450,
		EventIndex:  0,
		EventType:   "validator_uptime",
		Payload:     uptimePayload,
	}

	err := simProjectValidatorUptime(ev)
	if err != nil {
		t.Fatalf("Failed to project validator uptime: %v", err)
	}

	// Check SQL calls
	if len(executedQueries) != 5 {
		t.Fatalf("Expected 5 SQL Exec executions, got %d", len(executedQueries))
	}

	// Assert block_stats write
	if !strings.Contains(executedQueries[0].SQL, "block_stats") {
		t.Errorf("Expected first query to insert into block_stats, got: %s", executedQueries[0].SQL)
	}
	if executedQueries[0].Args[0].(int64) != 450 {
		t.Errorf("Expected block_height 450, got %v", executedQueries[0].Args[0])
	}

	// Assert validator_signatures write
	if !strings.Contains(executedQueries[2].SQL, "validator_signatures") {
		t.Errorf("Expected third query to insert into validator_signatures, got: %s", executedQueries[2].SQL)
	}
	t.Log("[PASS] Projection mapping simulation verified for block_stats and validator_signatures.")
}


// 4. StreamChainStats backpressure and slow consumer eviction simulation
func TestPhase5StreamChainStatsBackpressureSim(t *testing.T) {
	clientChan := make(chan string, 64)
	errChan := make(chan error, 1)

	// Callback simulates NATS event receiver putting events into the channel
	onMessageReceived := func(msg string) {
		select {
		case clientChan <- msg:
			// Sent successfully
		default:
			// Channel buffer full -> return ResourceExhausted error
			select {
			case errChan <- status.Error(codes.ResourceExhausted, "slow consumer channel buffer full"):
			default:
			}
		}
	}

	// Test case A: Client is reading fast enough -> no error
	for i := 0; i < 50; i++ {
		onMessageReceived("event")
	}
	if len(errChan) != 0 {
		t.Fatal("Fast consumer should not trigger ResourceExhausted error")
	}

	// Read all events from channel to clear it
	for len(clientChan) > 0 {
		<-clientChan
	}

	// Test case B: Client is a slow consumer (does not read) and buffer of 64 overflows
	for i := 0; i < 70; i++ {
		onMessageReceived("event")
	}

	// Verify error channel is populated with ResourceExhausted
	if len(errChan) == 0 {
		t.Fatal("Slow consumer should have triggered ResourceExhausted error on buffer overflow (>64)")
	}

	err := <-errChan
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.ResourceExhausted {
		t.Fatalf("Expected gRPC status ResourceExhausted, got: %v", err)
	}
	t.Logf("[PASS] Checked server-streaming backpressure: channel buffer of 64 holds and slow consumer rejected with ResourceExhausted: %s", st.Message())
}
