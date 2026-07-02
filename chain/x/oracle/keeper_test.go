package oracle

import (
	"bytes"
	"encoding/json"
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

func setupKeeper(t *testing.T) (Keeper, sdk.Context) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithBlockHeight(10)

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	keeper := NewKeeper(storeKey, nil, nil, nil)
	return keeper, ctx
}

func TestCommitReveal(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	operator := "cosmosvaloper1x..."
	feedID := "BTC_USD"
	roundID := uint64(1)
	value := uint64(50000)
	nonce := "secret123"

	// Check missing commit
	err := keeper.RevealReport(ctx, operator, feedID, roundID, value, nonce)
	if err == nil {
		t.Error("Expected error for missing commit")
	}

	// Commit
	hash := ComputeCommitHash(operator, feedID, roundID, value, nonce)
	keeper.CommitHash(ctx, operator, feedID, roundID, hash)

	// Verify commit is saved
	savedHash := keeper.GetCommit(ctx, operator, feedID, roundID)
	if !bytes.Equal(savedHash, hash) {
		t.Error("Expected saved hash to match committed hash")
	}

	// Reveal with incorrect value (mismatch)
	err = keeper.RevealReport(ctx, operator, feedID, roundID, 99999, nonce)
	if err == nil {
		t.Error("Expected hash mismatch error for incorrect value")
	}

	// Reveal with incorrect nonce (mismatch)
	err = keeper.RevealReport(ctx, operator, feedID, roundID, value, "wrong")
	if err == nil {
		t.Error("Expected hash mismatch error for incorrect nonce")
	}

	// Correct reveal
	err = keeper.RevealReport(ctx, operator, feedID, roundID, value, nonce)
	if err != nil {
		t.Errorf("Expected successful reveal, got error: %v", err)
	}

	// Check saved reveal value
	revealedVal, ok := keeper.GetReveal(ctx, operator, feedID, roundID)
	if !ok || revealedVal != value {
		t.Errorf("Expected revealed value to be %d, got %d (ok: %t)", value, revealedVal, ok)
	}
}

func TestMADAggregationAndOutliers(t *testing.T) {
	keeper, ctx := setupKeeper(t)

	feedID := "ETH_USD"
	roundID := uint64(1)

	// Set parameters: min 3 operators
	keeper.SetParams(ctx, Params{
		CommitWindow:             10,
		RevealWindow:             10,
		MinOperatorCommits:       3,
		StalenessThresholdBlocks: 100,
	})

	// Operators reporting prices: 2500, 2510, 2490, and one outlier at 8000
	reports := []struct {
		operator string
		value    uint64
		nonce    string
	}{
		{"op1", 2500, "n1"},
		{"op2", 2510, "n2"},
		{"op3", 2490, "n3"},
		{"op4", 8000, "n4"}, // Outlier!
	}

	for _, rep := range reports {
		hash := ComputeCommitHash(rep.operator, feedID, roundID, rep.value, rep.nonce)
		keeper.CommitHash(ctx, rep.operator, feedID, roundID, hash)
		_ = keeper.RevealReport(ctx, rep.operator, feedID, roundID, rep.value, rep.nonce)
	}

	// Aggregate
	price, err := keeper.AggregateRound(ctx, feedID, roundID)
	if err != nil {
		t.Fatalf("Expected successful aggregation, got error: %v", err)
	}

	// Without outlier 8000, median of {2490, 2500, 2510} should be 2500
	if price != 2500 {
		t.Errorf("Expected aggregated price to be 2500 (outlier 8000 removed), got %d", price)
	}

	// Test insufficient commits error
	_, err = keeper.AggregateRound(ctx, "UNKNOWN_FEED", 1)
	if err == nil {
		t.Error("Expected error for insufficient commits")
	}
}

func TestStalenessState(t *testing.T) {
	keeper, ctx := setupKeeper(t)
	feedID := "BTC_USD"

	keeper.SetParams(ctx, Params{
		CommitWindow:             10,
		RevealWindow:             10,
		MinOperatorCommits:       2,
		StalenessThresholdBlocks: 50,
	})

	// Setup aggregate price at block height 10
	store := ctx.KVStore(keeper.storeKey)
	agg := AggregatePrice{
		Price:       60000,
		BlockHeight: 10,
	}
	bz, _ := json.Marshal(agg)
	store.Set(append(AggregateKeyPrefix, []byte(feedID)...), bz)

	// At height 20 (delta 10 < threshold 50) -> fresh
	ctx = ctx.WithBlockHeight(20)
	price, _, err := keeper.GetLatestPrice(ctx, feedID)
	if err != nil {
		t.Errorf("Expected fresh price, got error: %v", err)
	}
	if price != 60000 {
		t.Errorf("Expected price 60000, got %d", price)
	}
	if keeper.IsFeedStale(ctx, feedID) {
		t.Error("Expected feed not to be stale at height 20")
	}

	// At height 70 (delta 60 > threshold 50) -> stale
	ctx = ctx.WithBlockHeight(70)
	_, _, err = keeper.GetLatestPrice(ctx, feedID)
	if err == nil {
		t.Error("Expected error for stale price")
	}
	if !keeper.IsFeedStale(ctx, feedID) {
		t.Error("Expected feed to be stale at height 70")
	}
}
