package validator

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
		&MsgFillValidatorSlot{},
		&MsgEjectValidator{},
		&MsgUpdatePartitionScheme{},
	)
}
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := GenesisState{
		MaxValidators:    30,
		PartitionScheme:  "equal-slots-30",
		ActiveValidators: []string{},
		QueuedEjections:  []string{},
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
		return fmt.Errorf("failed to unmarshal validator genesis state: %w", err)
	}
	if state.MaxValidators == 0 {
		return fmt.Errorf("max_validators must be positive")
	}
	if state.PartitionScheme == "" {
		return fmt.Errorf("partition_scheme cannot be empty")
	}
	for _, valAddrStr := range state.ActiveValidators {
		if _, err := sdk.ValAddressFromBech32(valAddrStr); err != nil {
			return fmt.Errorf("invalid active validator address %s: %w", valAddrStr, err)
		}
	}
	for _, valAddrStr := range state.QueuedEjections {
		if _, err := sdk.ValAddressFromBech32(valAddrStr); err != nil {
			return fmt.Errorf("invalid queued ejection validator address %s: %w", valAddrStr, err)
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

func (am AppModule) RegisterServices(cfg module.Configurator) {
	govAuthority := authtypes.NewModuleAddress("gov").String()
	msgServer := NewMsgServerImpl(am.keeper, govAuthority)
	cfg.MsgServer().RegisterService(&MsgServiceDesc, msgServer)
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	am.keeper.RegisterInvariants(ir)
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var state GenesisState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(fmt.Sprintf("failed to unmarshal validator genesis state: %v", err))
	}
	am.keeper.SetMaxValidators(ctx, state.MaxValidators)
	am.keeper.SetPartitionScheme(ctx, state.PartitionScheme)

	for _, valAddrStr := range state.ActiveValidators {
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err == nil {
			am.keeper.SetValidatorActive(ctx, valAddr)
		}
	}
	for _, valAddrStr := range state.QueuedEjections {
		valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
		if err == nil {
			am.keeper.QueueEjection(ctx, valAddr)
		}
	}

	ctx.Logger().Info("validator module initialized from genesis",
		"max_validators", state.MaxValidators,
		"partition_scheme", state.PartitionScheme,
		"active_validators", len(state.ActiveValidators),
		"queued_ejections", len(state.QueuedEjections),
	)
	return nil
}
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := GenesisState{
		MaxValidators:    am.keeper.GetMaxValidators(ctx),
		PartitionScheme:  am.keeper.GetPartitionScheme(ctx),
		ActiveValidators: am.keeper.GetAllActiveValidators(ctx),
		QueuedEjections:  am.keeper.GetAllQueuedEjections(ctx),
	}
	bz, err := json.Marshal(state)
	if err != nil {
		ctx.Logger().Error("failed to marshal validator genesis state", "error", err)
		return nil
	}
	return bz
}
func (am AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	updates := am.keeper.EndBlocker(sdkCtx)
	return updates, nil
}

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {}

// RegisterStoreDecoder registers a decoder for module's types
func (AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the module's simulation operations with their respective weight.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		cosmossim.NewWeightedOperation(20, SimulateMsgFillValidatorSlot(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgEjectValidator(am.keeper)),
		cosmossim.NewWeightedOperation(5, SimulateMsgUpdatePartitionScheme(am.keeper)),
	}
}

