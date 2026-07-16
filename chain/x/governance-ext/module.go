package gov_ext

import (
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
		&MsgMigrateContracts{},
		&MsgUpdateGasLimit{},
		&MsgUpdateValidatorSlot{},
		&MsgUpdateMilestone{},
		&MsgUpdateOracleOperator{},
		&MsgUpdateWitnessRegistry{},
		&MsgUpdateBridgeRelayerSet{},
	)
}
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := GenesisState{
		Params: Params{
			MinGasLimit: 100000,
			MaxGasLimit: 2000000,
		},
	}
	bz, _ := json.Marshal(defaultState)
	return bz
}
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	if bz == nil {
		return nil
	}
	var state GenesisState
	if err := json.Unmarshal(bz, &state); err != nil {
		return fmt.Errorf("failed to unmarshal govext genesis state: %w", err)
	}
	if state.Params.MinGasLimit <= 0 {
		return fmt.Errorf("min_gas_limit must be positive, got %d", state.Params.MinGasLimit)
	}
	if state.Params.MaxGasLimit <= 0 {
		return fmt.Errorf("max_gas_limit must be positive, got %d", state.Params.MaxGasLimit)
	}
	if state.Params.MinGasLimit > state.Params.MaxGasLimit {
		return fmt.Errorf("min_gas_limit (%d) must be <= max_gas_limit (%d)", state.Params.MinGasLimit, state.Params.MaxGasLimit)
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

func (am AppModule) RegisterServices(cfg module.Configurator) {
	govAuthority := authtypes.NewModuleAddress("gov").String()
	msgServer := NewMsgServerImpl(am.keeper, govAuthority)
	cfg.MsgServer().RegisterService(&GovExtMsgServiceDesc, msgServer)
}
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var state GenesisState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(fmt.Sprintf("failed to unmarshal govext genesis state: %v", err))
	}
	am.keeper.SetParams(ctx, state.Params)

	ctx.Logger().Info("govext module initialized from genesis",
		"min_gas_limit", state.Params.MinGasLimit,
		"max_gas_limit", state.Params.MaxGasLimit,
	)
	return nil
}
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := GenesisState{
		Params: am.keeper.GetParams(ctx),
	}
	bz, err := json.Marshal(state)
	if err != nil {
		ctx.Logger().Error("failed to marshal govext genesis state", "error", err)
		return nil
	}
	return bz
}
func (am AppModule) ConsensusVersion() uint64 { return 1 }

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {}

// RegisterStoreDecoder registers a decoder for module's types
func (AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the module's simulation operations with their respective weight.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		cosmossim.NewWeightedOperation(10, SimulateMsgMigrateContracts(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgUpdateValidatorSlot(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgUpdateMilestone(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgUpdateOracleOperator(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgUpdateWitnessRegistry(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgUpdateBridgeRelayerSet(am.keeper)),
	}
}

