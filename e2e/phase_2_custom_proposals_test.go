package e2e

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/chain/x/milestone"
	gov_ext "github.com/sovereign-l1/chain/x/governance-ext"
)

// Define mock keepers for testing proposal mutations
type mockValidatorKeeper struct {
	maxValidators uint32
}

func (m *mockValidatorKeeper) SetMaxValidators(ctx sdk.Context, max uint32) {
	m.maxValidators = max
}

type mockMilestoneKeeper struct {
	milestones map[string]milestone.Milestone
}

func (m *mockMilestoneKeeper) GetMilestone(ctx sdk.Context, id string) (milestone.Milestone, bool) {
	ms, ok := m.milestones[id]
	return ms, ok
}

func (m *mockMilestoneKeeper) SetMilestone(ctx sdk.Context, ms milestone.Milestone) {
	if m.milestones == nil {
		m.milestones = make(map[string]milestone.Milestone)
	}
	m.milestones[ms.ID] = ms
}

type mockOracleKeeper struct {
	operators map[string]bool
}

func (m *mockOracleKeeper) SetOperatorActive(ctx sdk.Context, operator string, active bool) {
	if m.operators == nil {
		m.operators = make(map[string]bool)
	}
	m.operators[operator] = active
}

type mockSettlementKeeper struct {
	witnesses map[string][]byte
}

func (m *mockSettlementKeeper) SetWitnessPubKey(ctx sdk.Context, witnessID string, pubKey []byte) {
	if m.witnesses == nil {
		m.witnesses = make(map[string][]byte)
	}
	m.witnesses[witnessID] = pubKey
}

func (m *mockSettlementKeeper) DeleteWitnessPubKey(ctx sdk.Context, witnessID string) {
	delete(m.witnesses, witnessID)
}

type mockBridgeKeeper struct {
	relayers map[string]bridge.Relayer
}

func (m *mockBridgeKeeper) SetRelayer(ctx sdk.Context, r bridge.Relayer) {
	if m.relayers == nil {
		m.relayers = make(map[string]bridge.Relayer)
	}
	m.relayers[r.Address] = r
}

func (m *mockBridgeKeeper) DeleteRelayer(ctx sdk.Context, address string) {
	delete(m.relayers, address)
}

func TestPhase2CustomProposals(t *testing.T) {
	ctx, _ := phase2SetupTestContext()
	storeKey := storetypes.NewKVStoreKey(gov_ext.StoreKey)

	valKeeper := &mockValidatorKeeper{}
	milesKeeper := &mockMilestoneKeeper{
		milestones: map[string]milestone.Milestone{
			"m1": {
				ID:          "m1",
				TargetPrice: 1000,
			},
		},
	}
	oracleKeeper := &mockOracleKeeper{}
	settKeeper := &mockSettlementKeeper{
		witnesses: make(map[string][]byte),
	}
	bridgeKeeper := &mockBridgeKeeper{
		relayers: make(map[string]bridge.Relayer),
	}

	wasm := &phase2MockWasmKeeper{ShouldFail: false}
	constitutionAddr := sdk.AccAddress([]byte("constitution_address"))

	k := gov_ext.NewKeeper(
		storeKey,
		nil,
		wasm,
		constitutionAddr,
		valKeeper,
		milesKeeper,
		oracleKeeper,
		settKeeper,
		bridgeKeeper,
	)

	// Set governance params
	k.SetParams(ctx, gov_ext.Params{
		MinGasLimit: 100000,
		MaxGasLimit: 2000000,
	})

	authority := sdk.AccAddress([]byte("authority")).String()

	// 1. MsgUpdateValidatorSlot
	msgVal := &gov_ext.MsgUpdateValidatorSlot{
		Authority:     authority,
		MaxValidators: 45,
	}
	err := k.ExecuteProposal(ctx, msgVal)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateValidatorSlot: %v", err)
	}
	if valKeeper.maxValidators != 45 {
		t.Errorf("Expected max validators to be 45, got %d", valKeeper.maxValidators)
	}

	// 2. MsgUpdateMilestone
	msgMiles := &gov_ext.MsgUpdateMilestone{
		Authority:   authority,
		MilestoneID: "m1",
		TargetPrice: 2500,
	}
	err = k.ExecuteProposal(ctx, msgMiles)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateMilestone: %v", err)
	}
	m, ok := milesKeeper.milestones["m1"]
	if !ok || m.TargetPrice != 2500 {
		t.Errorf("Expected target price to be 2500, got %d", m.TargetPrice)
	}

	// 3. MsgUpdateOracleOperator
	msgOracle := &gov_ext.MsgUpdateOracleOperator{
		Authority:       authority,
		OperatorAddress: "operator_addr",
		Active:          true,
	}
	err = k.ExecuteProposal(ctx, msgOracle)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateOracleOperator: %v", err)
	}
	if active := oracleKeeper.operators["operator_addr"]; !active {
		t.Error("Expected oracle operator to be active")
	}

	// 4. MsgUpdateWitnessRegistry (Add)
	msgWitnessAdd := &gov_ext.MsgUpdateWitnessRegistry{
		Authority:      authority,
		WitnessAddress: "witness_addr",
		PubKey:         []byte("pubkey_bytes"),
		Active:         true,
	}
	err = k.ExecuteProposal(ctx, msgWitnessAdd)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateWitnessRegistry Add: %v", err)
	}
	pk, ok := settKeeper.witnesses["witness_addr"]
	if !ok || string(pk) != "pubkey_bytes" {
		t.Errorf("Expected witness pubkey to be 'pubkey_bytes', got %s", string(pk))
	}

	// 5. MsgUpdateWitnessRegistry (Remove)
	msgWitnessRemove := &gov_ext.MsgUpdateWitnessRegistry{
		Authority:      authority,
		WitnessAddress: "witness_addr",
		Active:         false,
	}
	err = k.ExecuteProposal(ctx, msgWitnessRemove)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateWitnessRegistry Remove: %v", err)
	}
	if _, ok := settKeeper.witnesses["witness_addr"]; ok {
		t.Error("Expected witness to be removed")
	}

	// 6. MsgUpdateBridgeRelayerSet (Add)
	msgRelayerAdd := &gov_ext.MsgUpdateBridgeRelayerSet{
		Authority:      authority,
		RelayerAddress: "relayer_addr",
		PubKey:         []byte("relayer_pubkey"),
		Active:         true,
	}
	err = k.ExecuteProposal(ctx, msgRelayerAdd)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateBridgeRelayerSet Add: %v", err)
	}
	rel, ok := bridgeKeeper.relayers["relayer_addr"]
	if !ok || string(rel.PubKey) != "relayer_pubkey" {
		t.Error("Expected relayer to be added to bridge relayer set")
	}

	// 7. MsgUpdateBridgeRelayerSet (Remove)
	msgRelayerRemove := &gov_ext.MsgUpdateBridgeRelayerSet{
		Authority:      authority,
		RelayerAddress: "relayer_addr",
		Active:         false,
	}
	err = k.ExecuteProposal(ctx, msgRelayerRemove)
	if err != nil {
		t.Fatalf("Failed to execute MsgUpdateBridgeRelayerSet Remove: %v", err)
	}
	if _, ok := bridgeKeeper.relayers["relayer_addr"]; ok {
		t.Error("Expected relayer to be removed from bridge relayer set")
	}
}
