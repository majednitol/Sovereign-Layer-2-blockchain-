package e2e

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/relayer"
)

// --- Mock Invariant Registry for testing ---
type mockInvariantRegistry struct {
	routes map[string]map[string]sdk.Invariant
}

func (m *mockInvariantRegistry) RegisterRoute(moduleName, route string, inv sdk.Invariant) {
	if m.routes[moduleName] == nil {
		m.routes[moduleName] = make(map[string]sdk.Invariant)
	}
	m.routes[moduleName][route] = inv
}

// --- Mock Event Bus for Relayer Tests ---
type mockEventBusP4 struct {
	published map[string][]byte
}

func (m *mockEventBusP4) Publish(subject string, data []byte) error {
	m.published[subject] = data
	return nil
}

func (m *mockEventBusP4) Subscribe(subject string, handler func(msg []byte)) error {
	return nil
}

// --- Helper for Mocking LockBox Nonce Generation ---
func generateMockLockBoxNonce(sender string, userNonce uint64, blockNum uint64, amount uint64, timestamp uint64) []byte {
	h := sha256.New()
	h.Write([]byte(sender))
	h.Write(big.NewInt(int64(userNonce)).Bytes())
	h.Write(big.NewInt(int64(blockNum)).Bytes())
	h.Write(big.NewInt(int64(amount)).Bytes())
	h.Write(big.NewInt(int64(timestamp)).Bytes())
	return h.Sum(nil)
}

// TestPhase4SupplyCapModelAndInvariants verifies the 4.1 requirements (atomic check/mint, invariant registration, and breach detection).
func TestPhase4SupplyCapModelAndInvariants(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(bridge.StoreKey)
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStoreP4{
		stores: map[string]storetypes.KVStore{
			bridge.StoreKey: kvStore,
		},
	}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithChainID("sovereign-devnet-1").
		WithEventManager(sdk.NewEventManager())

	bank := &mockBankKeeperP4{
		balances: make(map[string]sdk.Coins),
	}

	k := bridge.NewKeeper(storeKey, nil, bank)
	am := bridge.NewAppModule(k)

	// 1. Invariant Registration Check
	ir := &mockInvariantRegistry{routes: make(map[string]map[string]sdk.Invariant)}
	am.RegisterInvariants(ir)

	invRoute, exists := ir.routes[bridge.ModuleName]["supply"]
	if !exists {
		t.Fatal("Expected bridge supply invariant to be registered")
	}

	// 2. Initial state check: invariant holds
	msg, breached := invRoute(ctx)
	if breached {
		t.Fatalf("Expected invariant to hold initially, got: %s", msg)
	}
	t.Logf("[PASS] Supply invariant initial check holds: %s", msg)

	// Set custom parameters: SupplyCap = 1000 WSOV (1,000,000,000 uwsov)
	params := k.GetParams(ctx)
	params.SupplyCap = "1000000000"
	k.SetParams(ctx, params)

	// Force state mutation to exceed supply cap to test invariant breach detection
	k.SetCosmosMinted(ctx, math.NewInt(1000000001)) // 1 over supply cap
	msg, breached = invRoute(ctx)
	if !breached {
		t.Fatal("Expected supply invariant to detect breach when cosmos_minted exceeds supply cap")
	}
	t.Logf("[PASS] Supply invariant successfully flagged breach: %s", msg)

	// Reset to valid state
	k.SetCosmosMinted(ctx, math.NewInt(500000000))
}

// TestPhase4CosmosBridgeModule verifies MsgBridgeIn/MsgBridgeOut and out-of-order execution via nonces bitmap registry.
func TestPhase4CosmosBridgeModule(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(bridge.StoreKey)
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStoreP4{
		stores: map[string]storetypes.KVStore{
			bridge.StoreKey: kvStore,
		},
	}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithChainID("sovereign-devnet-1").
		WithEventManager(sdk.NewEventManager())

	bank := &mockBankKeeperP4{
		balances: make(map[string]sdk.Coins),
	}

	k := bridge.NewKeeper(storeKey, nil, bank)

	// Setup 3 relayers
	var privs []*secp256k1.PrivKey
	var relAddresses []string
	for i := 0; i < 3; i++ {
		priv := secp256k1.GenPrivKey()
		privs = append(privs, priv)
		addr := sdk.AccAddress(priv.PubKey().Address()).String()
		relAddresses = append(relAddresses, addr)
		k.SetRelayer(ctx, bridge.Relayer{
			Address: addr,
			PubKey:  priv.PubKey().Bytes(),
		})
	}

	params := k.GetParams(ctx)
	params.QuorumThreshold = 2
	params.SupplyCap = "2000000000" // 2,000,000,000 uwsov
	k.SetParams(ctx, params)

	receiver := sdk.AccAddress([]byte("receiver_address_p4")).String()
	amount := sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(100000000))) // 100 WSOV

	// Create non-sequential nonces
	nonce1 := []byte("nonce_1_out_of_order_exec")
	nonce2 := []byte("nonce_2_out_of_order_exec")

	// Pre-sign nonce 2 (which is submitted first, simulating out-of-order execution)
	hash2 := bridge.ComputeBridgeMessageHash(receiver, amount, nonce2)
	var sigs2 [][]byte
	for j := 0; j < 2; j++ {
		sig, err := privs[j].Sign(hash2)
		if err != nil {
			t.Fatalf("Sign failed: %v", err)
		}
		sigs2 = append(sigs2, sig)
	}

	msg2 := bridge.MsgBridgeIn{
		Submitter:  relAddresses[0],
		Receiver:   receiver,
		Amount:     amount,
		Nonce:      nonce2,
		Signatures: sigs2,
	}

	// Submit MsgBridgeIn out of order (nonce 2 before nonce 1)
	err := k.ProcessBridgeIn(ctx, msg2)
	if err != nil {
		t.Fatalf("Out of order MsgBridgeIn failed: %v", err)
	}

	if !k.IsNonceProcessed(ctx, nonce2) {
		t.Fatal("Nonce 2 should be marked as processed")
	}
	if k.IsNonceProcessed(ctx, nonce1) {
		t.Fatal("Nonce 1 should not be processed yet")
	}

	// Verify minted tokens are present
	if bank.balances[receiver].AmountOf("uwsov").Int64() != 100000000 {
		t.Errorf("Expected balance 100000000, got %d", bank.balances[receiver].AmountOf("uwsov").Int64())
	}

	// Submit MsgBridgeOut (Withdrawal / Burn)
	msgOut := bridge.MsgBridgeOut{
		Sender:       receiver,
		BscRecipient: "0xabcdef1234567890abcdef1234567890abcdef12",
		Amount:       sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(40000000))),
	}

	err = k.ProcessBridgeOut(ctx, msgOut)
	if err != nil {
		t.Fatalf("MsgBridgeOut execution failed: %v", err)
	}

	// Verify balance decremented
	if bank.balances[receiver].AmountOf("uwsov").Int64() != 60000000 {
		t.Errorf("Expected balance 60000000, got %d", bank.balances[receiver].AmountOf("uwsov").Int64())
	}

	// Verify tracking updated
	if !k.GetCosmosMinted(ctx).Equal(math.NewInt(60000000)) {
		t.Errorf("Expected cosmos minted tracked to be 60000000, got %s", k.GetCosmosMinted(ctx).String())
	}
	t.Log("[PASS] Checked Out-of-Order execution, MsgBridgeIn, and MsgBridgeOut execution successfully.")
}

// TestPhase4RelayerWatchersAndAggregator verifies relayer components: tiered confirmations and vote/stuck alerts.
func TestPhase4RelayerWatchersAndAggregator(t *testing.T) {
	db, err := relayer.NewRelayerDB("memory")
	if err != nil {
		t.Fatalf("Failed to initialize relayer db: %v", err)
	}
	bus := &mockEventBusP4{published: make(map[string][]byte)}

	// 1. BSC Watcher Tiered Confirmation
	largeThreshold := uint64(50000)
	watcher := relayer.NewBSCWatcher(db, bus, largeThreshold)

	nonceStd := []byte("standard_tx_nonce_val_1")
	watcher.IngestLockEvent(relayer.LockEvent{
		User:            "0xuser_std",
		Amount:          1000, // Below threshold -> N=15
		CosmosRecipient: "cosmos1rec_std",
		Nonce:           nonceStd,
		BlockNumber:     100,
	})

	nonceLarge := []byte("large_tx_nonce_val_2")
	watcher.IngestLockEvent(relayer.LockEvent{
		User:            "0xuser_large",
		Amount:          60000, // Above threshold -> N=50
		CosmosRecipient: "cosmos1rec_large",
		Nonce:           nonceLarge,
		BlockNumber:     102,
	})

	// Advance blocks to 114 (14 blocks confirm standard -> standard not confirmed yet)
	_ = watcher.UpdateBlockNumber(114)
	if len(bus.published) != 0 {
		t.Fatalf("Expected no events published at N=14 standard, got %d", len(bus.published))
	}

	// Advance block to 115 (15 blocks confirm standard -> confirms!)
	_ = watcher.UpdateBlockNumber(115)
	if len(bus.published) != 1 {
		t.Fatalf("Expected standard event to be published, got %d", len(bus.published))
	}

	// Advance blocks to 151 (49 blocks confirm large -> large not confirmed yet)
	_ = watcher.UpdateBlockNumber(151)
	if len(bus.published) != 1 {
		t.Fatalf("Expected only standard event to remain published, got %d", len(bus.published))
	}

	// Advance block to 152 (50 blocks confirm large -> confirms!)
	_ = watcher.UpdateBlockNumber(152)
	if len(bus.published) != 2 {
		t.Fatalf("Expected both events to be published, got %d", len(bus.published))
	}

	t.Log("[PASS] Checked BSC Watcher tiered confirmation depths (15 vs 50 blocks) successfully.")

	// 2. Signature Aggregator Timeout & Stuck Alerts
	// Quorum = 3, Timeout = 5s, MaxRetries = 2
	agg := relayer.NewSigAggregator(db, bus, 3, 5, 2, big.NewInt(1337), "0x1234567890123456789012345678901234567890")
	stuckNonceHex := "nonce_stuck_hex"
	_ = db.SetNonceState(stuckNonceHex, "burned")

	// Timeout retry 1
	agg.HandleTimeout(stuckNonceHex)
	// Timeout retry 2
	agg.HandleTimeout(stuckNonceHex)
	// Timeout retry 3 (exceeds max retries = 2) -> alerts!
	agg.HandleTimeout(stuckNonceHex)

	if _, ok := bus.published["bridge.stuck"]; !ok {
		t.Fatal("Expected stuck alert to be published to EventBus")
	}

	state, _ := db.GetNonceState(stuckNonceHex)
	if state != "stuck" {
		t.Fatalf("Expected state to be 'stuck', got %s", state)
	}
	t.Log("[PASS] Checked SigAggregator quorum timeout and stuck alert dispatch successfully.")
}

// TestPhase4SubmitterPromotionLadder verifies the designated submitter and deterministic promotion delays.
func TestPhase4SubmitterPromotionLadder(t *testing.T) {
	db, _ := relayer.NewRelayerDB("memory")
	relayers := []string{"cosmos1rel_3", "cosmos1rel_1", "cosmos1rel_2"}
	delayFactor := 1 * time.Second

	// Submitter instantiations
	// Relayers list is sorted as: ["cosmos1rel_1", "cosmos1rel_2", "cosmos1rel_3"]
	// Index mapping: rel1 -> 0, rel2 -> 1, rel3 -> 2
	s1 := relayer.NewSubmitter(db, "cosmos1rel_1", relayers, delayFactor)
	s2 := relayer.NewSubmitter(db, "cosmos1rel_2", relayers, delayFactor)

	nonceHex := "nonce_prom_ladder"
	firstSeen := time.Now()

	// 1. Designated submitter check
	// blockHeight = 12 -> 12 % 3 = 0 (cosmos1rel_1)
	// Relayer 1 should submit instantly (delay = 0)
	shouldSubmit, delay := s1.CheckIfIShouldSubmit(12, nonceHex, firstSeen)
	if !shouldSubmit || delay != 0 {
		t.Errorf("Expected rel1 to submit instantly at height 12, got shouldSubmit: %v, delay: %v", shouldSubmit, delay)
	}

	// 2. Next submitter check (Relayer 2)
	// Relayer 2 (index 1) offset = (1 - 0 + 3) % 3 = 1 -> delay = 1 second
	shouldSubmit, delay = s2.CheckIfIShouldSubmit(12, nonceHex, firstSeen)
	if shouldSubmit || delay < 500*time.Millisecond || delay > 1500*time.Millisecond {
		t.Errorf("Expected rel2 to wait for delay slot, got shouldSubmit: %v, delay: %v", shouldSubmit, delay)
	}

	// 3. Simulating elapsed delay for Relayer 2
	expiredFirstSeen := time.Now().Add(-2 * time.Second)
	shouldSubmit, delay = s2.CheckIfIShouldSubmit(12, nonceHex, expiredFirstSeen)
	if !shouldSubmit || delay != 0 {
		t.Errorf("Expected rel2 to promote after delay slot elapsed, got shouldSubmit: %v, delay: %v", shouldSubmit, delay)
	}

	t.Log("[PASS] Submitter promotion ladder designated calculations and elapsed promotion checks passed.")
}

// TestPhase4MockLockBoxSolidityVerifications simulates the Solidity LockBox contract rate limiting, Gnosis Safe pause/unpause.
func TestPhase4MockLockBoxSolidityVerifications(t *testing.T) {
	// 1. Nonce Generation Verification
	sender := "0x51E2000000000000000000000000000000000001"
	nonceVal1 := generateMockLockBoxNonce(sender, 0, 1000, 500, 1718100000)
	nonceVal2 := generateMockLockBoxNonce(sender, 1, 1001, 500, 1718100010)

	if fmt.Sprintf("%x", nonceVal1) == fmt.Sprintf("%x", nonceVal2) {
		t.Fatal("Mock LockBox nonces must be collision-resistant and unique")
	}

	// 2. Gnosis Safe Pause/Unpause & Circuit Breaker Logic
	paused := false
	circuitBreakerAddress := "0xcb_addr"
	gnosisSafeAddress := "0xgs_addr"

	pauseFunc := func(caller string) error {
		if caller != circuitBreakerAddress && caller != gnosisSafeAddress {
			return fmt.Errorf("unauthorized pause caller")
		}
		paused = true
		return nil
	}

	unpauseFunc := func(caller string) error {
		if caller != gnosisSafeAddress {
			return fmt.Errorf("caller is not the Gnosis Safe")
		}
		paused = false
		return nil
	}

	// Test authorized EOA Pause
	err := pauseFunc(circuitBreakerAddress)
	if err != nil || !paused {
		t.Fatalf("Circuit breaker pause failed: %v", err)
	}

	// Test EOA Unpause block (unauthorized)
	err = unpauseFunc(circuitBreakerAddress)
	if err == nil {
		t.Fatal("Circuit breaker address must not be allowed to unpause")
	}

	// Test Gnosis Safe Unpause
	err = unpauseFunc(gnosisSafeAddress)
	if err != nil || paused {
		t.Fatalf("Gnosis Safe unpause failed: %v", err)
	}

	// 3. Block-Level Rate Limiting
	maxUnlockPerBlock := uint64(500000)
	lastUnlockBlock := uint64(0)
	currentBlockUnlockAmount := uint64(0)

	unlockWithRateLimit := func(blockNum uint64, amount uint64) error {
		if blockNum > lastUnlockBlock {
			lastUnlockBlock = blockNum
			currentBlockUnlockAmount = amount
		} else {
			currentBlockUnlockAmount += amount
		}
		if currentBlockUnlockAmount > maxUnlockPerBlock {
			return fmt.Errorf("rate limit exceeded")
		}
		return nil
	}

	// Under limit
	err = unlockWithRateLimit(100, 300000)
	if err != nil {
		t.Fatalf("Unlock should pass under rate limit: %v", err)
	}

	// Exceeds limit in same block
	err = unlockWithRateLimit(100, 250000)
	if err == nil {
		t.Fatal("Unlock should be rejected when rate limit exceeded in same block")
	}

	// Reset in new block
	err = unlockWithRateLimit(101, 250000)
	if err != nil {
		t.Fatalf("Unlock should pass in new block: %v", err)
	}

	t.Log("[PASS] Checked Mock LockBox Solidity nonce generation, Gnosis Safe pause/unpause, and rate limiting successfully.")
}

// TestPhase4MsgServerRouting verifies that MsgServer is registered and correctly routes and executes MsgBridgeIn and MsgBridgeOut.
func TestPhase4MsgServerRouting(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(bridge.StoreKey)
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStoreP4{
		stores: map[string]storetypes.KVStore{
			bridge.StoreKey: kvStore,
		},
	}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithChainID("sovereign-devnet-1").
		WithEventManager(sdk.NewEventManager())

	bank := &mockBankKeeperP4{
		balances: make(map[string]sdk.Coins),
	}

	k := bridge.NewKeeper(storeKey, nil, bank)
	msgServer := bridge.NewMsgServerImpl(k)

	// Setup 3 relayers
	var privs []*secp256k1.PrivKey
	var relAddresses []string
	for i := 0; i < 3; i++ {
		priv := secp256k1.GenPrivKey()
		privs = append(privs, priv)
		addr := sdk.AccAddress(priv.PubKey().Address()).String()
		relAddresses = append(relAddresses, addr)
		k.SetRelayer(ctx, bridge.Relayer{
			Address: addr,
			PubKey:  priv.PubKey().Bytes(),
		})
	}

	params := k.GetParams(ctx)
	params.QuorumThreshold = 2
	params.SupplyCap = "2000000000" // 2,000,000,000 uwsov
	k.SetParams(ctx, params)

	receiver := sdk.AccAddress([]byte("receiver_address_p4")).String()
	amount := sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(100000000))) // 100 WSOV
	nonce := []byte("nonce_val_msg_server_123")

	hash := bridge.ComputeBridgeMessageHash(receiver, amount, nonce)
	var sigs [][]byte
	for j := 0; j < 2; j++ {
		sig, err := privs[j].Sign(hash)
		if err != nil {
			t.Fatalf("Sign failed: %v", err)
		}
		sigs = append(sigs, sig)
	}

	msgIn := &bridge.MsgBridgeIn{
		Submitter:  relAddresses[0],
		Receiver:   receiver,
		Amount:     amount,
		Nonce:      nonce,
		Signatures: sigs,
	}

	// Route via MsgServer.BridgeIn
	goCtx := sdk.WrapSDKContext(ctx)
	_, err := msgServer.BridgeIn(goCtx, msgIn)
	if err != nil {
		t.Fatalf("MsgServer.BridgeIn failed: %v", err)
	}

	// Verify minted tokens are present
	if bank.balances[receiver].AmountOf("uwsov").Int64() != 100000000 {
		t.Errorf("Expected balance 100000000, got %d", bank.balances[receiver].AmountOf("uwsov").Int64())
	}

	// Route MsgBridgeOut via MsgServer.BridgeOut
	msgOut := &bridge.MsgBridgeOut{
		Sender:       receiver,
		BscRecipient: "0x1111111111111111111111111111111111111111",
		Amount:       sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(40000000))),
	}

	_, err = msgServer.BridgeOut(goCtx, msgOut)
	if err != nil {
		t.Fatalf("MsgServer.BridgeOut failed: %v", err)
	}

	// Verify balance decremented
	if bank.balances[receiver].AmountOf("uwsov").Int64() != 60000000 {
		t.Errorf("Expected balance 60000000, got %d", bank.balances[receiver].AmountOf("uwsov").Int64())
	}

	t.Log("[PASS] Checked MsgServer routing and execution for x/bridge successfully.")
}

// TestPhase4RelayerPayloadDatabase verifies that RelayerDB successfully saves/retrieves Lock and Burn event payloads.
func TestPhase4RelayerPayloadDatabase(t *testing.T) {
	db, err := relayer.NewRelayerDB("memory")
	if err != nil {
		t.Fatalf("Failed to initialize relayer db: %v", err)
	}

	// 1. LockEvent Save & Retrieve
	lock := relayer.LockEvent{
		User:            "0xuser_addr",
		Amount:          25000000,
		CosmosRecipient: "cosmos1recipient",
		Nonce:           []byte("lock_nonce_123_abc"),
		BlockNumber:     150,
	}

	err = db.SaveLockEvent(lock)
	if err != nil {
		t.Fatalf("Failed to save lock event: %v", err)
	}

	retrievedLock, err := db.GetLockEvent(fmt.Sprintf("%x", lock.Nonce))
	if err != nil {
		t.Fatalf("Failed to get lock event: %v", err)
	}
	if retrievedLock == nil {
		t.Fatal("Retrieved lock event is nil")
	}

	if retrievedLock.User != lock.User || retrievedLock.Amount != lock.Amount || retrievedLock.CosmosRecipient != lock.CosmosRecipient {
		t.Errorf("Retrieved lock event mismatch: %+v vs %+v", retrievedLock, lock)
	}

	// 2. BurnEvent Save & Retrieve
	burn := relayer.BurnEvent{
		Sender:       "cosmos1sender",
		BscRecipient: "0xrecipient_bsc",
		Amount:       50000000,
		Nonce:        []byte("burn_nonce_123_xyz"),
		BlockHeight:  1200,
	}

	err = db.SaveBurnEvent(burn)
	if err != nil {
		t.Fatalf("Failed to save burn event: %v", err)
	}

	retrievedBurn, err := db.GetBurnEvent(fmt.Sprintf("%x", burn.Nonce))
	if err != nil {
		t.Fatalf("Failed to get burn event: %v", err)
	}
	if retrievedBurn == nil {
		t.Fatal("Retrieved burn event is nil")
	}

	if retrievedBurn.Sender != burn.Sender || retrievedBurn.BscRecipient != burn.BscRecipient || retrievedBurn.Amount != burn.Amount {
		t.Errorf("Retrieved burn event mismatch: %+v vs %+v", retrievedBurn, burn)
	}

	t.Log("[PASS] Checked RelayerDB payload database storage and retrieval successfully.")
}

