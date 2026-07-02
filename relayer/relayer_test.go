package relayer

import (
	"fmt"
	"testing"
	"time"
)

type mockEventBus struct {
	published map[string][]byte
}

func (m *mockEventBus) Publish(subject string, data []byte) error {
	m.published[subject] = data
	return nil
}

func (m *mockEventBus) Subscribe(subject string, handler func(msg []byte)) error {
	return nil
}

func TestBSCWatcherTieredConfirmations(t *testing.T) {
	db, _ := NewRelayerDB("memory")
	bus := &mockEventBus{published: make(map[string][]byte)}
	largeThreshold := uint64(50000)

	watcher := NewBSCWatcher(db, bus, largeThreshold)

	// Ingest standard locked event (100 usov) at block 10
	nonce1 := []byte("nonce1_standard_locked_event")
	watcher.IngestLockEvent(LockEvent{
		User:            "0xuser1",
		Amount:          100,
		CosmosRecipient: "cosmos1rec1",
		Nonce:           nonce1,
		BlockNumber:     10,
	})

	// Ingest large locked event (60000 usov) at block 12
	nonce2 := []byte("nonce2_large_locked_event___")
	watcher.IngestLockEvent(LockEvent{
		User:            "0xuser2",
		Amount:          60000,
		CosmosRecipient: "cosmos1rec2",
		Nonce:           nonce2,
		BlockNumber:     12,
	})

	// Process blocks progressing from 10 to 24 (14 blocks difference for standard -> standard not confirmed yet)
	_ = watcher.UpdateBlockNumber(24)
	if len(bus.published) != 0 {
		t.Fatalf("Expected 0 published events, got %d", len(bus.published))
	}

	// Update block number to 25 (15 confirmations for standard -> confirms!)
	_ = watcher.UpdateBlockNumber(25)
	if len(bus.published) != 1 {
		t.Fatalf("Expected 1 published event, got %d", len(bus.published))
	}
	nonce1Hex := fmt.Sprintf("%x", nonce1)
	if _, ok := bus.published[fmt.Sprintf("bridge.bsc.locked.%s", nonce1Hex)]; !ok {
		t.Fatal("Expected standard lock event to be published")
	}

	// Update block number to 61 (49 confirmations for large transfer -> not confirmed yet)
	_ = watcher.UpdateBlockNumber(61)
	if len(bus.published) != 1 {
		t.Fatalf("Expected only standard event to remain published, got %d", len(bus.published))
	}

	// Update block number to 62 (50 confirmations for large transfer -> confirms!)
	_ = watcher.UpdateBlockNumber(62)
	if len(bus.published) != 2 {
		t.Fatalf("Expected both standard and large events to be published, got %d", len(bus.published))
	}
	nonce2Hex := fmt.Sprintf("%x", nonce2)
	if _, ok := bus.published[fmt.Sprintf("bridge.bsc.locked.%s", nonce2Hex)]; !ok {
		t.Fatal("Expected large lock event to be published")
	}
}

func TestCosmosWatcherBurnEvents(t *testing.T) {
	db, _ := NewRelayerDB("memory")
	bus := &mockEventBus{published: make(map[string][]byte)}

	watcher := NewCosmosWatcher(db, bus)

	nonce := []byte("nonce_cosmos_burn_event")
	err := watcher.IngestBurnEvent(BurnEvent{
		Sender:       "cosmos1sender",
		BscRecipient: "0xuser1",
		Amount:       500,
		Nonce:        nonce,
		BlockHeight:  100,
	})
	if err != nil {
		t.Fatalf("Failed to ingest burn event: %v", err)
	}

	nonceHex := fmt.Sprintf("%x", nonce)
	if _, ok := bus.published[fmt.Sprintf("bridge.cosmos.burnout.%s", nonceHex)]; !ok {
		t.Fatal("Expected cosmos burn event to be published to NATS")
	}

	state, _ := db.GetNonceState(nonceHex)
	if state != "burned" {
		t.Errorf("Expected nonce state 'burned', got %s", state)
	}
}

func TestSignatureAggregatorQuorumAndAlerts(t *testing.T) {
	db, _ := NewRelayerDB("memory")
	bus := &mockEventBus{published: make(map[string][]byte)}

	// Quorum = 3, MaxRetries = 2
	agg := NewSigAggregator(db, bus, 3, 10, 2)
	nonceHex := "abc123noncehex"

	// Add first vote
	q, _ := agg.IngestVote(VoteMsg{NonceHex: nonceHex, RelayerAddress: "rel1", Signature: []byte("sig1")})
	if q {
		t.Fatal("Quorum should not be met with 1 vote")
	}

	// Add duplicate vote (cheating)
	q, _ = agg.IngestVote(VoteMsg{NonceHex: nonceHex, RelayerAddress: "rel1", Signature: []byte("sig1")})
	if q {
		t.Fatal("Quorum should not be met with duplicate vote")
	}

	// Add second unique vote
	q, _ = agg.IngestVote(VoteMsg{NonceHex: nonceHex, RelayerAddress: "rel2", Signature: []byte("sig2")})
	if q {
		t.Fatal("Quorum should not be met with 2 votes")
	}

	// Add third unique vote -> quorum met!
	q, _ = agg.IngestVote(VoteMsg{NonceHex: nonceHex, RelayerAddress: "rel3", Signature: []byte("sig3")})
	if !q {
		t.Fatal("Quorum should be met with 3 unique votes")
	}

	// Verify DB state is updated
	state, _ := db.GetNonceState(nonceHex)
	if state != "ready" {
		t.Errorf("Expected nonce state 'ready', got %s", state)
	}

	// Stuck alert test
	stuckNonce := "stucknoncehex"
	_, _ = agg.IngestVote(VoteMsg{NonceHex: stuckNonce, RelayerAddress: "rel1", Signature: []byte("sig1")})

	// Trigger timeout 1
	agg.HandleTimeout(stuckNonce)
	if agg.stuckAlerts[stuckNonce] {
		t.Fatal("Should not mark stuck on first retry")
	}

	// Trigger timeout 2
	agg.HandleTimeout(stuckNonce)
	if agg.stuckAlerts[stuckNonce] {
		t.Fatal("Should not mark stuck on second retry")
	}

	// Trigger timeout 3 (exceeds max retries = 2)
	agg.HandleTimeout(stuckNonce)
	if !agg.stuckAlerts[stuckNonce] {
		t.Fatal("Should mark stuck after exceeding max retries")
	}

	// Verify NATS alert was published
	if _, ok := bus.published["bridge.stuck"]; !ok {
		t.Fatal("Expected stuck alert to be published to NATS")
	}
}

func TestSubmitterPromotionLadder(t *testing.T) {
	db, _ := NewRelayerDB("memory")
	relayers := []string{"rel3_addr", "rel1_addr", "rel2_addr"}
	delayFactor := 2 * time.Second

	// Relayer 1
	s1 := NewSubmitter(db, "rel1_addr", relayers, delayFactor)
	// Relayer 2
	s2 := NewSubmitter(db, "rel2_addr", relayers, delayFactor)

	// Relayer sorted list will be ["rel1_addr", "rel2_addr", "rel3_addr"]
	// Index mappings: rel1 -> 0, rel2 -> 1, rel3 -> 2

	// At blockHeight = 10:
	// designatedIndex = 10 % 3 = 1 (rel2_addr)
	// Slot delay offset for rel1_addr (index 0):
	// slotDiff = (0 - 1 + 3) % 3 = 2 slots -> 4 seconds delay
	// Slot delay offset for rel2_addr (index 1):
	// slotDiff = (1 - 1 + 3) % 3 = 0 slots -> 0 seconds delay (submits instantly)

	firstSeen := time.Now()
	nonceHex := "nonceabc"

	// 1. Check designated submitter (rel2) instantly
	shouldSubmit, delay := s2.CheckIfIShouldSubmit(10, nonceHex, firstSeen)
	if !shouldSubmit || delay != 0 {
		t.Errorf("Expected rel2 to submit instantly at block 10, got shouldSubmit %v, delay %v", shouldSubmit, delay)
	}

	// 2. Check next in ladder (rel1) instantly -> should not submit yet, should return slot delay (~4s)
	shouldSubmit, delay = s1.CheckIfIShouldSubmit(10, nonceHex, firstSeen)
	if shouldSubmit || delay < 3*time.Second || delay > 5*time.Second {
		t.Errorf("Expected rel1 to wait slot delay, got shouldSubmit %v, delay %v", shouldSubmit, delay)
	}

	// 3. Simulating elapsed time (4s) for next in ladder
	expiredFirstSeen := time.Now().Add(-5 * time.Second)
	shouldSubmit, delay = s1.CheckIfIShouldSubmit(10, nonceHex, expiredFirstSeen)
	if !shouldSubmit || delay != 0 {
		t.Errorf("Expected rel1 to promote after slot delay elapsed, got shouldSubmit %v, delay %v", shouldSubmit, delay)
	}

	// 4. Verify no submission if already submitted
	_ = s1.MarkSubmitted(nonceHex)
	shouldSubmit, _ = s1.CheckIfIShouldSubmit(10, nonceHex, expiredFirstSeen)
	if shouldSubmit {
		t.Fatal("Should not submit if already marked submitted")
	}
}
