package gov_ext

import (
	"errors"
	"testing"

	"cosmossdk.io/log/v2"
	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sovereign-l1/chain/simutil"
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

type mockWasmKeeper struct {
	failExecute bool
	executed    bool
}

func (m *mockWasmKeeper) Execute(ctx sdk.Context, contractAddr sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	m.executed = true
	if m.failExecute {
		return nil, errors.New("wasm execution failed")
	}
	return nil, nil
}

type dummyMsg struct {
	sdk.Msg
}

func setupKeeper(t *testing.T, wasm WasmKeeper) (Keeper, sdk.Context) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithEventManager(sdk.NewEventManager()).
		WithLogger(log.NewNopLogger())

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	constAddr := sdk.AccAddress([]byte("constitution________"))

	keeper := NewKeeper(storeKey, nil, wasm, constAddr, nil, nil, nil, nil, nil)
	return keeper, ctx
}

func TestMsgMigrateContractsBypass(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: true} // if constitution check executes, it will fail
	keeper, ctx := setupKeeper(t, wasm)
	simGov := simutil.NewSimGov(keeper)

	msg := &MsgMigrateContracts{
		Authority:          "authority",
		ContractAddress:    "contract",
		NewCodeID:          1,
		ExecutionDelaySecs: 604800, // 7 days (valid)
	}

	// Should succeed (bypasses Constitution) and not trigger Wasm execution
	err := simGov.ProposeAndExecute(ctx, msg)
	if err != nil {
		t.Errorf("Expected successful execution of MsgMigrateContracts, got: %v", err)
	}
	if wasm.executed {
		t.Error("Expected Wasm keeper not to be called (bypass failed)")
	}

	// Should fail due to delay < 7 days
	invalidMsg := msg
	invalidMsg.ExecutionDelaySecs = 1000
	err = simGov.ProposeAndExecute(ctx, invalidMsg)
	if err == nil {
		t.Error("Expected error for delay < 7 days")
	}
}

func TestMsgUpdateGasLimitBypass(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: true} // if constitution check executes, it will fail
	keeper, ctx := setupKeeper(t, wasm)
	simGov := simutil.NewSimGov(keeper)

	msg := &MsgUpdateGasLimit{
		Authority: "authority",
		GasLimit:  500000, // within default bounds [100,000 - 2,000,000]
	}

	// Should succeed (bypasses Constitution) and not trigger Wasm execution
	err := simGov.ProposeAndExecute(ctx, msg)
	if err != nil {
		t.Errorf("Expected successful execution of MsgUpdateGasLimit, got: %v", err)
	}
	if wasm.executed {
		t.Error("Expected Wasm keeper not to be called (bypass failed)")
	}

	// Should fail due to out-of-bounds gas limit
	invalidMsg := msg
	invalidMsg.GasLimit = 50000
	err = simGov.ProposeAndExecute(ctx, invalidMsg)
	if err == nil {
		t.Error("Expected error for gas limit below minimum bounds")
	}
}

func TestConstitutionCheckFallbacks(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: false}
	keeper, ctx := setupKeeper(t, wasm)
	simGov := simutil.NewSimGov(keeper)

	// Dummy message that does not bypass
	msg := dummyMsg{}

	// Case 1: Wasm executes successfully -> proposal succeeds
	err := simGov.ProposeAndExecute(ctx, msg)
	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}
	if !wasm.executed {
		t.Error("Expected Wasm keeper to be called")
	}

	// Case 2: Wasm fails -> proposal fails
	wasm.failExecute = true
	err = simGov.ProposeAndExecute(ctx, msg)
	if err == nil {
		t.Error("Expected error when Wasm execution fails")
	}
}

func TestCustomProposalsConstitutionCheck(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: false}
	keeper, ctx := setupKeeper(t, wasm)
	simGov := simutil.NewSimGov(keeper)

	msgs := []sdk.Msg{
		&MsgUpdateValidatorSlot{Authority: "authority", MaxValidators: 10},
		&MsgUpdateMilestone{Authority: "authority", MilestoneID: "m1", TargetPrice: 100},
		&MsgUpdateOracleOperator{Authority: "authority", OperatorAddress: "oracle", Active: true},
		&MsgUpdateWitnessRegistry{Authority: "authority", WitnessAddress: "witness", Active: true},
		&MsgUpdateBridgeRelayerSet{Authority: "authority", RelayerAddress: "relayer", Active: true},
	}

	for _, msg := range msgs {
		wasm.executed = false
		err := simGov.ProposeAndExecute(ctx, msg)
		if err != nil {
			t.Errorf("Expected successful execution for %T, got: %v", msg, err)
		}
		if !wasm.executed {
			t.Errorf("Expected constitution check to be executed for %T", msg)
		}

		// Now make constitution check fail
		wasm.failExecute = true
		err = simGov.ProposeAndExecute(ctx, msg)
		if err == nil {
			t.Errorf("Expected failure for %T when Wasm check fails", msg)
		}
		wasm.failExecute = false
	}
}

func TestMsgServerGasLimitBypass(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: false}
	keeper, ctx := setupKeeper(t, wasm)

	govAuthority := sdk.AccAddress([]byte("gov_________________")).String()
	server := NewMsgServerImpl(keeper, govAuthority)

	// Valid gas limit update (bypasses constitution)
	msg := &MsgUpdateGasLimit{
		Authority: govAuthority,
		GasLimit:  500000,
	}
	_, err := server.UpdateGasLimit(sdk.WrapSDKContext(ctx), msg)
	if err != nil {
		t.Fatalf("Expected successful UpdateGasLimit, got: %v", err)
	}

	// Out-of-bounds gas limit should fail
	msgBad := &MsgUpdateGasLimit{
		Authority: govAuthority,
		GasLimit:  50000, // below min_gas_limit
	}
	_, err = server.UpdateGasLimit(sdk.WrapSDKContext(ctx), msgBad)
	if err == nil {
		t.Error("Expected error for gas limit below minimum bounds")
	}
}

func TestMsgServerUnauthorized(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: false}
	keeper, ctx := setupKeeper(t, wasm)

	govAuthority := sdk.AccAddress([]byte("gov_________________")).String()
	server := NewMsgServerImpl(keeper, govAuthority)

	// Unauthorized validator slot update
	msg := &MsgUpdateValidatorSlot{
		Authority:     sdk.AccAddress([]byte("bad_________________")).String(),
		MaxValidators: 50,
	}
	_, err := server.UpdateValidatorSlot(sdk.WrapSDKContext(ctx), msg)
	if err == nil {
		t.Error("Expected unauthorized error for non-gov authority")
	}
}

func TestGenesisRoundTrip(t *testing.T) {
	wasm := &mockWasmKeeper{failExecute: false}
	keeper, ctx := setupKeeper(t, wasm)

	// Set custom params
	keeper.SetParams(ctx, Params{
		MinGasLimit: 200000,
		MaxGasLimit: 5000000,
	})

	am := NewAppModule(keeper)
	jsonBz := am.ExportGenesis(ctx, nil)

	// Clear state
	keeper.SetParams(ctx, Params{})

	// Re-import
	am.InitGenesis(ctx, nil, jsonBz)

	p := keeper.GetParams(ctx)
	if p.MinGasLimit != 200000 {
		t.Errorf("Expected MinGasLimit 200000, got %d", p.MinGasLimit)
	}
	if p.MaxGasLimit != 5000000 {
		t.Errorf("Expected MaxGasLimit 5000000, got %d", p.MaxGasLimit)
	}
}

func TestValidateGenesisInvalid(t *testing.T) {
	basic := AppModuleBasic{}

	// min > max should fail
	invalidJSON := []byte(`{"params":{"min_gas_limit":5000000,"max_gas_limit":100000}}`)
	err := basic.ValidateGenesis(nil, nil, invalidJSON)
	if err == nil {
		t.Error("Expected error for min_gas_limit > max_gas_limit")
	}

	// zero min should fail
	zeroJSON := []byte(`{"params":{"min_gas_limit":0,"max_gas_limit":100000}}`)
	err = basic.ValidateGenesis(nil, nil, zeroJSON)
	if err == nil {
		t.Error("Expected error for min_gas_limit = 0")
	}
}
