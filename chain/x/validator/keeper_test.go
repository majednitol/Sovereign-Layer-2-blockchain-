package validator

import (
	"context"
	"testing"

	"cosmossdk.io/log/v2"
	"cosmossdk.io/math"
	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
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
	validators []struct {
		addr  sdk.ValAddress
		power int64
	}
}

func (m mockStakingKeeper) GetLastValidatorPower(ctx context.Context, valAddr sdk.ValAddress) (int64, error) {
	for _, v := range m.validators {
		if v.addr.Equals(valAddr) {
			return v.power, nil
		}
	}
	return 0, nil
}

func (m mockStakingKeeper) GetLastTotalPower(ctx context.Context) (math.Int, error) {
	var total int64
	for _, v := range m.validators {
		total += v.power
	}
	return math.NewInt(total), nil
}

func (m mockStakingKeeper) IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) bool) error {
	for _, v := range m.validators {
		if handler(v.addr, v.power) {
			break
		}
	}
	return nil
}

func (m mockStakingKeeper) GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error) {
	pk := ed25519.GenPrivKey().PubKey()
	anyPk, _ := codectypes.NewAnyWithValue(pk)
	return stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		ConsensusPubkey: anyPk,
	}, nil
}

type mockSlashingKeeper struct {
	tombstoneCalls []sdk.ConsAddress
	initCalls      []sdk.ConsAddress
}

func (m *mockSlashingKeeper) Tombstone(ctx context.Context, valAddr sdk.ConsAddress) error {
	m.tombstoneCalls = append(m.tombstoneCalls, valAddr)
	return nil
}

func (m *mockSlashingKeeper) HasValidatorSigningInfo(ctx context.Context, consAddr sdk.ConsAddress) bool {
	for _, c := range m.initCalls {
		if c.Equals(consAddr) {
			return true
		}
	}
	return false
}

func (m *mockSlashingKeeper) SetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) error {
	m.initCalls = append(m.initCalls, address)
	return nil
}

func setupKeeper(t *testing.T, maxValidators uint32, validators []struct {
	addr  sdk.ValAddress
	power int64
}) (Keeper, sdk.Context, *mockSlashingKeeper) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithLogger(log.NewNopLogger()).
		WithEventManager(sdk.NewEventManager())

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	staking := mockStakingKeeper{validators: validators}
	slashing := &mockSlashingKeeper{}

	keeper := NewKeeper(storeKey, nil, staking, slashing, nil, nil, maxValidators)
	return keeper, ctx, slashing
}

func TestValidatorKeeperSlots(t *testing.T) {
	valAddr1 := sdk.ValAddress([]byte("val1________________"))
	valAddr2 := sdk.ValAddress([]byte("val2________________"))

	keeper, ctx, _ := setupKeeper(t, 2, nil)

	// Test IsValidatorActive initial
	if keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected val1 to be inactive initially")
	}

	// Set active
	keeper.SetValidatorActive(ctx, valAddr1)
	if !keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected val1 to be active")
	}

	// Remove active
	keeper.RemoveValidatorActive(ctx, valAddr1)
	if keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected val1 to be inactive after removal")
	}

	// Queue Ejection
	if keeper.IsEjectionQueued(ctx, valAddr2) {
		t.Error("Expected val2 not to be in ejection queue initially")
	}

	keeper.QueueEjection(ctx, valAddr2)
	if !keeper.IsEjectionQueued(ctx, valAddr2) {
		t.Error("Expected val2 to be in ejection queue")
	}
}

func TestValidatorKeeperEndBlocker(t *testing.T) {
	valAddr1 := sdk.ValAddress([]byte("val_slot_1__________"))
	valAddr2 := sdk.ValAddress([]byte("val_slot_2__________"))
	valAddr3 := sdk.ValAddress([]byte("val_slot_3__________"))

	validators := []struct {
		addr  sdk.ValAddress
		power int64
	}{
		{addr: valAddr1, power: 100},
		{addr: valAddr2, power: 80},
		{addr: valAddr3, power: 50},
	}

	// MaxValidators = 2. So slot 1 and slot 2 qualify. Validator 3 gets ejected.
	keeper, ctx, slashing := setupKeeper(t, 2, validators)

	// Pre-fill active set for validator 3 to trigger ejection logic
	keeper.SetValidatorActive(ctx, valAddr3)

	updates := keeper.EndBlocker(ctx)

	// Assertions
	if !keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected val1 to be active")
	}
	if !keeper.IsValidatorActive(ctx, valAddr2) {
		t.Error("Expected val2 to be active")
	}
	if keeper.IsValidatorActive(ctx, valAddr3) {
		t.Error("Expected val3 to be inactive (ejected)")
	}
	if !keeper.IsEjectionQueued(ctx, valAddr3) {
		t.Error("Expected val3 ejection to be queued")
	}

	// updates check
	if len(updates) != 3 {
		t.Errorf("Expected 3 validator updates, got %d", len(updates))
	}

	// Check power overrides
	if updates[0].Power != 1000000 {
		t.Errorf("Expected val1 equalized power to be 1000000, got %d", updates[0].Power)
	}
	if updates[1].Power != 1000000 {
		t.Errorf("Expected val2 equalized power to be 1000000, got %d", updates[1].Power)
	}
	if updates[2].Power != 0 {
		t.Errorf("Expected val3 power to be 0, got %d", updates[2].Power)
	}

	// Slashing assertions
	if len(slashing.initCalls) != 2 {
		t.Errorf("Expected 2 InitializeValidatorSigningInfo calls, got %d", len(slashing.initCalls))
	}
	if len(slashing.tombstoneCalls) != 1 {
		t.Errorf("Expected 1 Tombstone call, got %d", len(slashing.tombstoneCalls))
	}
}

func TestValidatorMsgServer(t *testing.T) {
	valAddr1 := sdk.ValAddress([]byte("val_slot_1__________"))
	keeper, ctx, _ := setupKeeper(t, 2, nil)

	govAuthority := sdk.AccAddress([]byte("gov_________________")).String()
	server := NewMsgServerImpl(keeper, govAuthority)

	// Test FillValidatorSlot
	msgFill := MsgFillValidatorSlot{
		ValidatorAddress: valAddr1.String(),
	}
	_, err := server.FillValidatorSlot(sdk.WrapSDKContext(ctx), &msgFill)
	if err != nil {
		t.Fatalf("Expected successful FillValidatorSlot, got: %v", err)
	}
	if !keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected validator to be active after FillValidatorSlot")
	}

	// Test EjectValidator
	msgEject := MsgEjectValidator{
		ValidatorAddress: valAddr1.String(),
	}
	_, err = server.EjectValidator(sdk.WrapSDKContext(ctx), &msgEject)
	if err != nil {
		t.Fatalf("Expected successful EjectValidator, got: %v", err)
	}
	if keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected validator to be inactive after EjectValidator")
	}

	// Test UpdatePartitionScheme
	msgUpdate := MsgUpdatePartitionScheme{
		Authority: govAuthority,
		NewScheme: "equal-slots-50",
	}
	_, err = server.UpdatePartitionScheme(sdk.WrapSDKContext(ctx), &msgUpdate)
	if err != nil {
		t.Fatalf("Expected successful UpdatePartitionScheme, got: %v", err)
	}
	if keeper.GetPartitionScheme(ctx) != "equal-slots-50" {
		t.Errorf("Expected partition scheme equal-slots-50, got %s", keeper.GetPartitionScheme(ctx))
	}

	// Unauthorized update partition scheme
	msgUpdateBad := MsgUpdatePartitionScheme{
		Authority: sdk.AccAddress([]byte("bad_________________")).String(),
		NewScheme: "equal-slots-10",
	}
	_, err = server.UpdatePartitionScheme(sdk.WrapSDKContext(ctx), &msgUpdateBad)
	if err == nil {
		t.Error("Expected unauthorized error for non-gov authority")
	}
}

func TestValidatorGenesis(t *testing.T) {
	valAddr1 := sdk.ValAddress([]byte("val1________________"))
	valAddr2 := sdk.ValAddress([]byte("val2________________"))

	keeper, ctx, _ := setupKeeper(t, 2, nil)

	keeper.SetMaxValidators(ctx, 40)
	keeper.SetPartitionScheme(ctx, "equal-slots-40")
	keeper.SetValidatorActive(ctx, valAddr1)
	keeper.QueueEjection(ctx, valAddr2)

	am := NewAppModule(keeper)
	jsonBz := am.ExportGenesis(ctx, nil)

	// Clean state
	keeper.RemoveValidatorActive(ctx, valAddr1)

	am.InitGenesis(ctx, nil, jsonBz)

	if keeper.GetMaxValidators(ctx) != 40 {
		t.Errorf("Expected max validators 40, got %d", keeper.GetMaxValidators(ctx))
	}
	if keeper.GetPartitionScheme(ctx) != "equal-slots-40" {
		t.Errorf("Expected partition scheme equal-slots-40, got %s", keeper.GetPartitionScheme(ctx))
	}
	if !keeper.IsValidatorActive(ctx, valAddr1) {
		t.Error("Expected valAddr1 to be active after InitGenesis")
	}
	if !keeper.IsEjectionQueued(ctx, valAddr2) {
		t.Error("Expected valAddr2 ejection to be queued after InitGenesis")
	}
}

