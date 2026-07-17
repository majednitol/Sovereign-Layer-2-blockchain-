package e2e

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/sovereign-l1/chain/x/milestone"
	"github.com/sovereign-l1/chain/x/oracle"
	"github.com/sovereign-l1/chain/x/validator"
)

// Chaos-specific Mock NATS
type chaosMockNATS struct {
	connected   bool
	droppedMsgs []interface{}
	published   []interface{}
}

func (n *chaosMockNATS) Publish(msg interface{}) {
	if !n.connected {
		n.droppedMsgs = append(n.droppedMsgs, msg)
		return
	}
	n.published = append(n.published, msg)
}

func (n *chaosMockNATS) Reconnect() {
	n.connected = true
	for _, msg := range n.droppedMsgs {
		n.published = append(n.published, msg)
	}
	n.droppedMsgs = nil
}

func TestChaosNatsDropAndBackfill(t *testing.T) {
	nats := &chaosMockNATS{connected: true}

	// 1. Publish normally
	nats.Publish("event1")
	if len(nats.published) != 1 || len(nats.droppedMsgs) != 0 {
		t.Errorf("Expected 1 published, got %d, dropped: %d", len(nats.published), len(nats.droppedMsgs))
	}

	// 2. Drop connection
	nats.connected = false
	nats.Publish("event2")
	nats.Publish("event3")

	if len(nats.published) != 1 || len(nats.droppedMsgs) != 2 {
		t.Errorf("Expected 1 published, 2 dropped, got published: %d, dropped: %d", len(nats.published), len(nats.droppedMsgs))
	}

	// 3. Reconnect & Backfill
	nats.Reconnect()
	if len(nats.published) != 3 || len(nats.droppedMsgs) != 0 {
		t.Errorf("Expected 3 published after backfill, got %d, dropped: %d", len(nats.published), len(nats.droppedMsgs))
	}
}

// Staking/Slashing mocks for validator E2E test
type chaosStakingKeeper struct {
	validators []stakingtypes.Validator
}

func (m chaosStakingKeeper) GetLastValidatorPower(ctx context.Context, valAddr sdk.ValAddress) (int64, error) {
	for _, v := range m.validators {
		if v.OperatorAddress == valAddr.String() {
			return v.GetConsensusPower(sdk.DefaultPowerReduction), nil
		}
	}
	return 0, nil
}

func (m chaosStakingKeeper) GetLastTotalPower(ctx context.Context) (math.Int, error) {
	var total int64
	for _, v := range m.validators {
		total += v.GetConsensusPower(sdk.DefaultPowerReduction)
	}
	return math.NewInt(total), nil
}

func (m chaosStakingKeeper) IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) (stop bool)) error {
	for _, v := range m.validators {
		addr, _ := sdk.ValAddressFromBech32(v.OperatorAddress)
		if handler(addr, v.GetConsensusPower(sdk.DefaultPowerReduction)) {
			break
		}
	}
	return nil
}

func (m chaosStakingKeeper) GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error) {
	for _, v := range m.validators {
		if v.OperatorAddress == valAddr.String() {
			return v, nil
		}
	}
	return stakingtypes.Validator{}, errors.New("validator not found")
}

type chaosSlashingKeeper struct {
	tombstones []sdk.ConsAddress
	jails      []sdk.ConsAddress
	inits      []sdk.ConsAddress
}

func (m *chaosSlashingKeeper) Tombstone(ctx context.Context, consAddr sdk.ConsAddress) error {
	m.tombstones = append(m.tombstones, consAddr)
	return nil
}

func (m *chaosSlashingKeeper) Jail(ctx context.Context, consAddr sdk.ConsAddress) error {
	m.jails = append(m.jails, consAddr)
	return nil
}

func (m *chaosSlashingKeeper) HasValidatorSigningInfo(ctx context.Context, consAddr sdk.ConsAddress) bool {
	for _, c := range m.inits {
		if c.Equals(consAddr) {
			return true
		}
	}
	return false
}

func (m *chaosSlashingKeeper) SetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) error {
	m.inits = append(m.inits, address)
	return nil
}

func TestChaosValidatorEjectionSlashing(t *testing.T) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	dbMap := make(map[string]storetypes.KVStore)
	dbMap[validator.StoreKey] = kvStore
	ms := phase2MockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(storetypes.NewInfiniteGasMeter())

	valAddr1 := sdk.ValAddress([]byte("val1________________"))
	valAddr2 := sdk.ValAddress([]byte("val2________________"))
	valAddr3 := sdk.ValAddress([]byte("val3________________"))

	pk1 := sdked25519.GenPrivKey().PubKey()
	pk2 := sdked25519.GenPrivKey().PubKey()
	pk3 := sdked25519.GenPrivKey().PubKey()

	anyPk1, _ := codectypes.NewAnyWithValue(pk1)
	anyPk2, _ := codectypes.NewAnyWithValue(pk2)
	anyPk3, _ := codectypes.NewAnyWithValue(pk3)

	val1 := stakingtypes.Validator{OperatorAddress: valAddr1.String(), ConsensusPubkey: anyPk1}
	val2 := stakingtypes.Validator{OperatorAddress: valAddr2.String(), ConsensusPubkey: anyPk2}
	val3 := stakingtypes.Validator{OperatorAddress: valAddr3.String(), ConsensusPubkey: anyPk3}

	staking := chaosStakingKeeper{
		validators: []stakingtypes.Validator{val1, val2, val3},
	}
	slashing := &chaosSlashingKeeper{}

	// MaxValidators = 2. Validator 3 should be ejected.
	storeKey := storetypes.NewKVStoreKey(validator.StoreKey)
	k := validator.NewKeeper(storeKey, nil, staking, slashing, nil, nil, 2)

	// Pre-fill active set for validator 3 to trigger ejection
	k.SetValidatorActive(ctx, valAddr3)

	k.EndBlocker(ctx)

	// Check active validator statuses
	if !k.IsValidatorActive(ctx, valAddr1) || !k.IsValidatorActive(ctx, valAddr2) {
		t.Fatal("Expected validator 1 and 2 to be active")
	}
	if k.IsValidatorActive(ctx, valAddr3) {
		t.Fatal("Expected validator 3 to be ejected")
	}

	// Slashing assertions
	if len(slashing.jails) != 1 {
		t.Errorf("Expected 1 Jail call, got %d", len(slashing.jails))
	}
	cons3, _ := val3.GetConsAddr()
	if !slashing.jails[0].Equals(sdk.ConsAddress(cons3)) {
		t.Error("Expected ejected validator 3 to be jailed")
	}

	if len(slashing.inits) != 2 {
		t.Errorf("Expected 2 signing info initializations, got %d", len(slashing.inits))
	}
}

// Oracle/Milestone Chaos
type chaosOracleKeeper struct {
	stale  bool
	prices map[string]uint64
}

func (m chaosOracleKeeper) IsFeedStale(ctx sdk.Context, feedID string) bool {
	return m.stale
}

func (m chaosOracleKeeper) GetLatestPrice(ctx sdk.Context, feedID string) (uint64, int64, error) {
	return m.prices[feedID], ctx.BlockHeight(), nil
}

func TestChaosOracleStalenessMilestoneClock(t *testing.T) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	dbMap := make(map[string]storetypes.KVStore)
	dbMap[milestone.StoreKey] = kvStore
	dbMap[oracle.StoreKey] = kvStoreV2Wrapper{dbadapter.Store{DB: dbm.NewMemDB()}}
	ms := phase2MockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithEventManager(sdk.NewEventManager())

	oKeeper := &chaosOracleKeeper{stale: false, prices: map[string]uint64{"BTC_USD": 50000}}
	bKeeper := &phase2MockBankKeeper{}

	storeKey := storetypes.NewKVStoreKey(milestone.StoreKey)
	k := milestone.NewKeeper(storeKey, nil, oKeeper, bKeeper)
	k.SetParams(ctx, milestone.Params{MaxActiveMilestones: 500})

	mID := "m1"
	m := milestone.Milestone{
		ID:                 mID,
		FeedID:             "BTC_USD",
		TargetPrice:        60000,
		RemainingBlocks:    10,
		State:              milestone.StatePending,
		VestingPoolAddress: sdk.AccAddress([]byte("pool")).String(),
		PayoutAmount:       10000000,
	}
	k.SetMilestone(ctx, m)

	// 1. Stale feed -> transitions to stale-blocked, remaining blocks stays 10
	oKeeper.stale = true
	k.EndBlocker(ctx)

	ret, _ := k.GetMilestone(ctx, mID)
	if ret.State != milestone.StateStaleBlocked {
		t.Errorf("Expected state stale-blocked, got %s", ret.State)
	}
	if ret.RemainingBlocks != 10 {
		t.Errorf("Expected RemainingBlocks to stay 10 (paused), got %d", ret.RemainingBlocks)
	}

	// 2. Feed recovers, price is below target -> resumes clock
	oKeeper.stale = false
	k.EndBlocker(ctx) // transitions stale-blocked to pending
	k.EndBlocker(ctx) // ticks clock (pending -> remaining=9)

	ret, _ = k.GetMilestone(ctx, mID)
	if ret.State != milestone.StatePending {
		t.Errorf("Expected state pending, got %s", ret.State)
	}
	if ret.RemainingBlocks != 9 {
		t.Errorf("Expected clock to tick (RemainingBlocks = 9), got %d", ret.RemainingBlocks)
	}

	// 3. Stale again -> clock pauses at 9
	oKeeper.stale = true
	k.EndBlocker(ctx)

	ret, _ = k.GetMilestone(ctx, mID)
	if ret.State != milestone.StateStaleBlocked {
		t.Errorf("Expected state stale-blocked again, got %s", ret.State)
	}
	if ret.RemainingBlocks != 9 {
		t.Errorf("Expected clock to freeze at 9, got %d", ret.RemainingBlocks)
	}

	// 4. Recover, price jumps above target -> achieved directly
	oKeeper.stale = false
	oKeeper.prices["BTC_USD"] = 65000
	k.EndBlocker(ctx)

	ret, _ = k.GetMilestone(ctx, mID)
	if ret.State != milestone.StateAchieved {
		t.Errorf("Expected milestone to be achieved, got %s", ret.State)
	}
	if len(bKeeper.Transfers) != 1 {
		t.Errorf("Expected vesting payout to be triggered once, got %d", len(bKeeper.Transfers))
	}
}
