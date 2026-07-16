package milestone

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	cosmossim "github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}
)

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}
func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateMilestone{},
	)
}
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := MilestoneGenesisState{
		Params: Params{
			MaxActiveMilestones: 500,
		},
		Milestones: []Milestone{},
	}
	bz, _ := json.Marshal(defaultState)
	return bz
}
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	if bz == nil {
		return nil
	}
	var state MilestoneGenesisState
	if err := json.Unmarshal(bz, &state); err != nil {
		return fmt.Errorf("failed to unmarshal milestone genesis state: %w", err)
	}
	if state.Params.MaxActiveMilestones <= 0 {
		return fmt.Errorf("milestone max_active_milestones must be positive")
	}
	for _, m := range state.Milestones {
		if m.ID == "" {
			return fmt.Errorf("milestone ID cannot be empty")
		}
		if m.FeedID == "" {
			return fmt.Errorf("milestone %s: feed_id cannot be empty", m.ID)
		}
	}
	return nil
}
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {}

type AppModule struct {
	AppModuleBasic
	keeper Keeper
}

func NewAppModule(keeper Keeper) AppModule {
	return AppModule{
		keeper: keeper,
	}
}

func (AppModule) IsOnePerModuleType() {}
func (AppModule) IsAppModule()        {}

// H2 FIX: Wire MsgServer so governance can create milestones
func (am AppModule) RegisterServices(cfg module.Configurator) {
	govAuthority := authtypes.NewModuleAddress("gov").String()
	msgServer := NewMsgServerImpl(am.keeper, govAuthority)
	cfg.MsgServer().RegisterService(&MilestoneMsgServiceDesc, msgServer)
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
}

// H3 FIX: Load milestone state from genesis
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var state MilestoneGenesisState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(fmt.Sprintf("failed to unmarshal milestone genesis state: %v", err))
	}

	am.keeper.SetParams(ctx, state.Params)

	for _, m := range state.Milestones {
		am.keeper.SetMilestone(ctx, m)
	}

	ctx.Logger().Info("milestone module initialized from genesis",
		"max_active_milestones", state.Params.MaxActiveMilestones,
		"milestones", len(state.Milestones),
	)
	return nil
}

// H3 FIX: Export milestone state for chain snapshots
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := MilestoneGenesisState{
		Params:     am.keeper.GetParams(ctx),
		Milestones: am.keeper.GetAllMilestones(ctx),
	}
	bz, err := json.Marshal(state)
	if err != nil {
		ctx.Logger().Error("failed to marshal milestone genesis state", "error", err)
		return nil
	}
	return bz
}
func (am AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	am.keeper.EndBlocker(sdkCtx)
	return nil, nil
}

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {}

// RegisterStoreDecoder registers a decoder for module's types
func (AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the module's simulation operations with their respective weight.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		cosmossim.NewWeightedOperation(20, SimulateMsgCreateMilestone(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgAchieveMilestone(am.keeper, am.keeper.oracleKeeper)),
		cosmossim.NewWeightedOperation(5, SimulateMilestoneExpiry(am.keeper)),
		cosmossim.NewWeightedOperation(5, SimulateMilestoneStaleRecovery(am.keeper)),
	}
}

