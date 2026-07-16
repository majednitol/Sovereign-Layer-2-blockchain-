package settlement

import (
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
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
		&MsgSettlement{},
	)
}
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := GenesisState{
		Params: Params{
			TimestampToleranceSeconds: 30,
		},
		Witnesses:       []Witness{},
		ProcessedNonces: [][]byte{},
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
		return fmt.Errorf("failed to unmarshal settlement genesis state: %w", err)
	}
	if state.Params.TimestampToleranceSeconds <= 0 {
		return fmt.Errorf("timestamp tolerance must be positive")
	}
	for _, w := range state.Witnesses {
		if w.ID == "" {
			return fmt.Errorf("witness ID cannot be empty")
		}
		if len(w.PubKey) == 0 {
			return fmt.Errorf("witness %s public key cannot be empty", w.ID)
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
	msgServer := NewMsgServerImpl(am.keeper)
	cfg.MsgServer().RegisterService(&MsgServiceDesc, msgServer)
}
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var state GenesisState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(fmt.Sprintf("failed to unmarshal settlement genesis state: %v", err))
	}
	am.keeper.SetParams(ctx, state.Params)
	for _, w := range state.Witnesses {
		am.keeper.SetWitnessPubKey(ctx, w.ID, w.PubKey)
	}
	for _, nonce := range state.ProcessedNonces {
		am.keeper.MarkSettlementProcessed(ctx, nonce)
	}

	ctx.Logger().Info("settlement module initialized from genesis",
		"tolerance", state.Params.TimestampToleranceSeconds,
		"witnesses", len(state.Witnesses),
		"nonces", len(state.ProcessedNonces),
	)
	return nil
}
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := GenesisState{
		Params:          am.keeper.GetParams(ctx),
		Witnesses:       am.keeper.GetAllWitnesses(ctx),
		ProcessedNonces: am.keeper.GetAllProcessedNonces(ctx),
	}
	bz, err := json.Marshal(state)
	if err != nil {
		ctx.Logger().Error("failed to marshal settlement genesis state", "error", err)
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
		cosmossim.NewWeightedOperation(20, SimulateMsgSettlement(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgInvalidWitnessSettlement(am.keeper)),
		cosmossim.NewWeightedOperation(10, SimulateMsgExpiredTimestampSettlement(am.keeper)),
	}
}

