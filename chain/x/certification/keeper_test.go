package certification

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/math"
	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
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

type mockStakingKeeper struct {
	validators map[string]stakingtypes.Validator
	powers     map[string]int64
}

func (m mockStakingKeeper) IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) bool) error {
	for addrStr, power := range m.powers {
		if handler(sdk.ValAddress(addrStr), power) {
			break
		}
	}
	return nil
}

func (m mockStakingKeeper) GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error) {
	val, ok := m.validators[string(valAddr)]
	if !ok {
		return stakingtypes.Validator{}, errors.New("validator not found")
	}
	return val, nil
}

type mockSlashingKeeper struct {
	slashed map[string]int
	jailed  map[string]bool
}

func (m *mockSlashingKeeper) Slash(ctx context.Context, consAddr sdk.ConsAddress, fraction math.LegacyDec, power, distributionHeight int64) error {
	m.slashed[string(consAddr)] = m.slashed[string(consAddr)] + 1
	return nil
}

func (m *mockSlashingKeeper) Jail(ctx context.Context, consAddr sdk.ConsAddress) error {
	m.jailed[string(consAddr)] = true
	return nil
}

func setupKeeper(t *testing.T, staking mockStakingKeeper, slashing *mockSlashingKeeper) (Keeper, sdk.Context) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithEventManager(sdk.NewEventManager())

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	keeper := NewKeeper(storeKey, nil, staking, slashing)
	return keeper, ctx
}

func TestCertificationKeeperEndBlocker(t *testing.T) {
	slashing := &mockSlashingKeeper{
		slashed: make(map[string]int),
		jailed:  make(map[string]bool),
	}
	keeper, ctx := setupKeeper(t, mockStakingKeeper{}, slashing)

	// Normal initially
	if keeper.IsDegradedMode(ctx) {
		t.Error("Expected not to be in degraded mode initially")
	}

	// 1st rejection
	keeper.EndBlocker(ctx, true)
	if keeper.GetConsecutiveRejectionCount(ctx) != 1 {
		t.Errorf("Expected rejection count to be 1, got %d", keeper.GetConsecutiveRejectionCount(ctx))
	}
	if keeper.IsDegradedMode(ctx) {
		t.Error("Expected not to be in degraded mode after 1 rejection")
	}

	// 4 more rejections (total 5)
	for i := 0; i < 4; i++ {
		keeper.EndBlocker(ctx, true)
	}

	if keeper.GetConsecutiveRejectionCount(ctx) != 5 {
		t.Errorf("Expected rejection count to be 5, got %d", keeper.GetConsecutiveRejectionCount(ctx))
	}
	if !keeper.IsDegradedMode(ctx) {
		t.Error("Expected to enter degraded mode after 5 rejections")
	}

	// Test threshold in degraded mode
	ratio := keeper.CheckProcessProposalThreshold(ctx)
	if ratio != 0.51 {
		t.Errorf("Expected relaxed threshold 0.51, got %f", ratio)
	}

	// Success block
	keeper.EndBlocker(ctx, false)
	if keeper.GetConsecutiveRejectionCount(ctx) != 0 {
		t.Errorf("Expected rejection count to be reset to 0, got %d", keeper.GetConsecutiveRejectionCount(ctx))
	}
}

func TestMissedExtensions(t *testing.T) {
	slashing := &mockSlashingKeeper{
		slashed: make(map[string]int),
		jailed:  make(map[string]bool),
	}
	keeper, ctx := setupKeeper(t, mockStakingKeeper{}, slashing)
	valAddr := sdk.ValAddress([]byte("val_address_test____"))

	if keeper.GetMissedExtensions(ctx, valAddr) != 0 {
		t.Error("Expected 0 missed extensions initially")
	}

	missed := keeper.IncrementMissedExtensions(ctx, valAddr)
	if missed != 1 {
		t.Errorf("Expected missed extensions to be 1, got %d", missed)
	}

	if keeper.GetMissedExtensions(ctx, valAddr) != 1 {
		t.Errorf("Expected missed extensions to retrieve 1, got %d", keeper.GetMissedExtensions(ctx, valAddr))
	}

	keeper.ResetMissedExtensions(ctx, valAddr)
	if keeper.GetMissedExtensions(ctx, valAddr) != 0 {
		t.Errorf("Expected missed extensions to be reset, got %d", keeper.GetMissedExtensions(ctx, valAddr))
	}
}

func TestParamsSerialization(t *testing.T) {
	slashing := &mockSlashingKeeper{
		slashed: make(map[string]int),
		jailed:  make(map[string]bool),
	}
	keeper, ctx := setupKeeper(t, mockStakingKeeper{}, slashing)

	params := Params{
		MaxConsecutiveRejections: 8,
		MissedExtensionLimit:     15,
	}

	keeper.SetParams(ctx, params)
	retrieved := keeper.GetParams(ctx)

	if retrieved.MaxConsecutiveRejections != 8 {
		t.Errorf("Expected MaxConsecutiveRejections to be 8, got %d", retrieved.MaxConsecutiveRejections)
	}
	if retrieved.MissedExtensionLimit != 15 {
		t.Errorf("Expected MissedExtensionLimit to be 15, got %d", retrieved.MissedExtensionLimit)
	}
}

func TestLivenessWindowAndJailing(t *testing.T) {
	valAddr := sdk.ValAddress([]byte("val_liveness_test___"))
	pk := ed25519.GenPrivKey().PubKey()
	anyPk, _ := codectypes.NewAnyWithValue(pk)
	val := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		ConsensusPubkey: anyPk,
	}
	staking := mockStakingKeeper{
		validators: map[string]stakingtypes.Validator{
			string(valAddr): val,
		},
		powers: map[string]int64{
			string(valAddr): 100,
		},
	}
	slashing := &mockSlashingKeeper{
		slashed: make(map[string]int),
		jailed:  make(map[string]bool),
	}
	keeper, ctx := setupKeeper(t, staking, slashing)

	// Set initial params
	params := Params{
		MaxConsecutiveRejections: 5,
		MissedExtensionLimit:     3,
	}
	keeper.SetParams(ctx, params)

	// Mark validator attested initially
	keeper.SetValidatorAttested(ctx, valAddr, true)

	// Set block height to 50 (below 100 threshold bootstrapping, no jailing should happen even with 0 signed blocks)
	ctx = ctx.WithBlockHeight(50)
	keeper.EndBlocker(ctx, false)

	consAddr, _ := val.GetConsAddr()
	if slashing.jailed[string(consAddr)] {
		t.Error("Expected validator not to be jailed at height < 100")
	}

	// Set block height to 120 (above 100)
	// Required threshold at height 120 is (120 * 5000) / 10000 = 60 signed blocks.
	// Since signed count is 0, validator should get jailed.
	ctx = ctx.WithBlockHeight(120)
	keeper.EndBlocker(ctx, false)

	if !slashing.jailed[string(consAddr)] {
		t.Error("Expected validator to be jailed due to low liveness at height 120")
	}

	// Reset jailed status for testing attestation jailing
	slashing.jailed = make(map[string]bool)

	// Make sure liveness is satisfied (signed count >= threshold)
	// Let's set signed count to 70
	keeper.SetValidatorSignedCount(ctx, valAddr, 70)

	// Mark validator as NOT attested
	keeper.SetValidatorAttested(ctx, valAddr, false)

	keeper.EndBlocker(ctx, false)
	if !slashing.jailed[string(consAddr)] {
		t.Error("Expected validator to be jailed because they are not attested")
	}
}

func TestHandleMissedExtension(t *testing.T) {
	valAddr := sdk.ValAddress([]byte("val_missed_test_____"))
	pk := ed25519.GenPrivKey().PubKey()
	anyPk, _ := codectypes.NewAnyWithValue(pk)
	val := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		ConsensusPubkey: anyPk,
	}
	staking := mockStakingKeeper{
		validators: map[string]stakingtypes.Validator{
			string(valAddr): val,
		},
		powers: map[string]int64{
			string(valAddr): 100,
		},
	}
	slashing := &mockSlashingKeeper{
		slashed: make(map[string]int),
		jailed:  make(map[string]bool),
	}
	keeper, ctx := setupKeeper(t, staking, slashing)

	params := Params{
		MaxConsecutiveRejections: 5,
		MissedExtensionLimit:     3,
	}
	keeper.SetParams(ctx, params)

	// 1st missed
	keeper.HandleMissedExtension(ctx, valAddr)
	if keeper.GetMissedExtensions(ctx, valAddr) != 1 {
		t.Errorf("Expected 1 missed extension, got %d", keeper.GetMissedExtensions(ctx, valAddr))
	}

	// 2nd missed
	keeper.HandleMissedExtension(ctx, valAddr)

	consAddr, _ := val.GetConsAddr()
	if slashing.jailed[string(consAddr)] || slashing.slashed[string(consAddr)] > 0 {
		t.Error("Expected validator not to be jailed or slashed yet")
	}

	// 3rd missed (reaches MissedExtensionLimit)
	keeper.HandleMissedExtension(ctx, valAddr)

	if !slashing.jailed[string(consAddr)] {
		t.Error("Expected validator to be jailed after exceeding missed extension limit")
	}
	if slashing.slashed[string(consAddr)] != 1 {
		t.Errorf("Expected validator to be slashed once, got %d", slashing.slashed[string(consAddr)])
	}
	if keeper.GetMissedExtensions(ctx, valAddr) != 0 {
		t.Errorf("Expected missed extensions count to reset, got %d", keeper.GetMissedExtensions(ctx, valAddr))
	}
}
