package milestone

import (
	"fmt"
	"testing"

	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func BenchmarkEndBlocker(b *testing.B) {
	oracle := mockOracleKeeper{
		prices: make(map[string]uint64),
		stale:  make(map[string]bool),
	}
	bank := &mockBankKeeper{}

	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithEventManager(sdk.NewEventManager())

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	keeper := NewKeeper(storeKey, nil, oracle, bank)

	numFeeds := 10
	numMilestones := 500

	for i := 0; i < numFeeds; i++ {
		feedID := fmt.Sprintf("feed_%d", i)
		oracle.prices[feedID] = 50000
		oracle.stale[feedID] = false
	}
	keeper.oracleKeeper = oracle

	for i := 0; i < numMilestones; i++ {
		feedID := fmt.Sprintf("feed_%d", i%numFeeds)
		m := Milestone{
			ID:                 fmt.Sprintf("m_%d", i),
			FeedID:             feedID,
			TargetPrice:        60000,
			RemainingBlocks:    1000,
			State:              StatePending,
			VestingPoolAddress: "vesting_pool",
		}
		keeper.SetMilestone(ctx, m)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		keeper.EndBlocker(ctx)
	}
}
