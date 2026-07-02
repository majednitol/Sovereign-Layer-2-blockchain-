package milestone

import (
	"context"
	"testing"

	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockMultiStore struct {
	storetypes.MultiStore
	store storetypes.KVStore
}

func (m mockMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return m.store
}

func (m mockMultiStore) GetStore(key storetypes.StoreKey) storetypes.Store {
	return m.store
}

type kvStoreV2Wrapper struct {
	legacytypes.KVStore
}

func (w kvStoreV2Wrapper) GetStoreType() storetypes.StoreType {
	return storetypes.StoreType(w.KVStore.GetStoreType())
}

func (w kvStoreV2Wrapper) Iterator(start, end []byte) storetypes.Iterator {
	return w.KVStore.Iterator(start, end)
}

func (w kvStoreV2Wrapper) ReverseIterator(start, end []byte) storetypes.Iterator {
	return w.KVStore.ReverseIterator(start, end)
}

func (w kvStoreV2Wrapper) CacheWrap() storetypes.CacheWrap {
	return nil
}

type mockOracleKeeper struct {
	prices map[string]uint64
	stale  map[string]bool
}

func (m mockOracleKeeper) IsFeedStale(ctx sdk.Context, feedID string) bool {
	return m.stale[feedID]
}

func (m mockOracleKeeper) GetLatestPrice(ctx sdk.Context, feedID string) (uint64, int64, error) {
	return m.prices[feedID], ctx.BlockHeight(), nil
}

type mockBankKeeper struct {
	payouts []struct {
		to  sdk.AccAddress
		amt sdk.Coins
	}
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	m.payouts = append(m.payouts, struct {
		to  sdk.AccAddress
		amt sdk.Coins
	}{toAddr, amt})
	return nil
}

func setupKeeper(t *testing.T, oracle OracleKeeper, bank BankKeeper) (Keeper, sdk.Context) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithEventManager(sdk.NewEventManager())

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	keeper := NewKeeper(storeKey, nil, oracle, bank)
	return keeper, ctx
}

func TestMilestoneLifecycle(t *testing.T) {
	oracle := mockOracleKeeper{
		prices: map[string]uint64{"BTC_USD": 50000},
		stale:  map[string]bool{"BTC_USD": false},
	}
	bank := &mockBankKeeper{}

	keeper, ctx := setupKeeper(t, oracle, bank)

	vestingPool := sdk.AccAddress([]byte("vesting_pool________")).String()
	m := Milestone{
		ID:                 "m1",
		FeedID:             "BTC_USD",
		TargetPrice:        60000,
		RemainingBlocks:    5,
		State:              StatePending,
		VestingPoolAddress: vestingPool,
	}

	keeper.SetMilestone(ctx, m)

	// Step 1: Normal processing, price is 50000 < 60000, not stale -> remains pending, decrements block count
	keeper.EndBlocker(ctx)
	m, _ = keeper.GetMilestone(ctx, "m1")
	if m.State != StatePending || m.RemainingBlocks != 4 {
		t.Errorf("Expected pending state and 4 remaining blocks, got state %s, remaining %d", m.State, m.RemainingBlocks)
	}

	// Step 2: Feed goes stale -> transitions to stale-blocked
	oracle.stale["BTC_USD"] = true
	keeper.oracleKeeper = oracle

	keeper.EndBlocker(ctx)
	m, _ = keeper.GetMilestone(ctx, "m1")
	if m.State != StateStaleBlocked || m.RemainingBlocks != 4 {
		t.Errorf("Expected stale-blocked state and remaining blocks paused at 4, got state %s, remaining %d", m.State, m.RemainingBlocks)
	}

	// Step 3: Block counter stays paused when stale-blocked
	keeper.EndBlocker(ctx)
	m, _ = keeper.GetMilestone(ctx, "m1")
	if m.RemainingBlocks != 4 {
		t.Errorf("Expected remaining blocks to stay paused at 4, got %d", m.RemainingBlocks)
	}

	// Step 4: Feed recovers, price is still 50000 < 60000 -> transitions back to pending
	oracle.stale["BTC_USD"] = false
	keeper.oracleKeeper = oracle

	keeper.EndBlocker(ctx)
	m, _ = keeper.GetMilestone(ctx, "m1")
	if m.State != StatePending || m.RemainingBlocks != 4 {
		t.Errorf("Expected pending state and 4 remaining blocks, got state %s, remaining %d", m.State, m.RemainingBlocks)
	}

	// Step 5: Feed goes stale again
	oracle.stale["BTC_USD"] = true
	keeper.oracleKeeper = oracle
	keeper.EndBlocker(ctx)

	// Step 6: Feed recovers and price jumps directly to target -> transitions directly to achieved
	oracle.stale["BTC_USD"] = false
	oracle.prices["BTC_USD"] = 65000
	keeper.oracleKeeper = oracle

	keeper.EndBlocker(ctx)
	m, _ = keeper.GetMilestone(ctx, "m1")
	if m.State != StateAchieved {
		t.Errorf("Expected direct transition to achieved, got state %s", m.State)
	}

	// Verify vesting payout was triggered
	if len(bank.payouts) != 1 {
		t.Fatalf("Expected 1 bank payout to be triggered, got %d", len(bank.payouts))
	}
	expectedTo := sdk.AccAddress([]byte("vesting_pool________"))
	if !bank.payouts[0].to.Equals(expectedTo) {
		t.Errorf("Expected payout to address %s, got %s", expectedTo, bank.payouts[0].to)
	}
}

func TestMilestoneExpiry(t *testing.T) {
	oracle := mockOracleKeeper{
		prices: map[string]uint64{"BTC_USD": 50000},
		stale:  map[string]bool{"BTC_USD": false},
	}
	bank := &mockBankKeeper{}

	keeper, ctx := setupKeeper(t, oracle, bank)

	m := Milestone{
		ID:                 "m2",
		FeedID:             "BTC_USD",
		TargetPrice:        60000,
		RemainingBlocks:    1,
		State:              StatePending,
		VestingPoolAddress: "vesting_pool",
	}
	keeper.SetMilestone(ctx, m)

	// Process 1 block -> reaches remaining blocks 0 -> transitions to expired
	keeper.EndBlocker(ctx)
	m, _ = keeper.GetMilestone(ctx, "m2")
	if m.State != StateExpired {
		t.Errorf("Expected state to be expired, got %s", m.State)
	}
}
