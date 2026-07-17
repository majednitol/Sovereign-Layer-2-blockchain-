package oracle

import (
	"context"
	"encoding/json"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	cosmossim "github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
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
		&MsgCommitOracleHash{},
		&MsgRevealOracleReport{},
	)
}

func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := GenesisState{
		Params: Params{
			CommitWindow:             10,
			RevealWindow:             10,
			MinOperatorCommits:       3,
			StalenessThresholdBlocks: 100,
		},
		Operators: []string{},
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
		return err
	}
	if state.Params.CommitWindow <= 0 {
		return ErrInvalidParams("commit_window must be positive")
	}
	if state.Params.RevealWindow <= 0 {
		return ErrInvalidParams("reveal_window must be positive")
	}
	if state.Params.MinOperatorCommits <= 0 {
		return ErrInvalidParams("min_operator_commits must be positive")
	}
	if state.Params.StalenessThresholdBlocks <= 0 {
		return ErrInvalidParams("staleness_threshold_blocks must be positive")
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

// RegisterServices wires the MsgServer so that oracle commit/reveal
// transactions are actually routed to the keeper.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	msgServer := NewMsgServer(am.keeper)
	// Register the message handlers using the gRPC service descriptor
	cfg.MsgServer().RegisterService(&OracleMsgServiceDesc, msgServer)
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	am.keeper.RegisterInvariants(ir)
}

// InitGenesis loads oracle params and operator set from genesis state.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var state GenesisState
	if err := json.Unmarshal(data, &state); err != nil {
		ctx.Logger().Error("failed to unmarshal oracle genesis state", "error", err)
		return nil
	}

	// Set params from genesis
	am.keeper.SetParams(ctx, state.Params)

	// Register operators from genesis
	for _, operator := range state.Operators {
		am.keeper.SetOperatorActive(ctx, operator, true)
	}

	ctx.Logger().Info("oracle module initialized from genesis",
		"commit_window", state.Params.CommitWindow,
		"reveal_window", state.Params.RevealWindow,
		"min_commits", state.Params.MinOperatorCommits,
		"staleness_threshold", state.Params.StalenessThresholdBlocks,
		"operators", len(state.Operators),
	)

	return nil
}

// ExportGenesis exports the current oracle state for chain snapshots.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	params := am.keeper.GetParams(ctx)
	operators := am.keeper.GetAllActiveOperators(ctx)

	state := GenesisState{
		Params:    params,
		Operators: operators,
	}

	bz, err := json.Marshal(state)
	if err != nil {
		ctx.Logger().Error("failed to marshal oracle genesis state for export", "error", err)
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
		cosmossim.NewWeightedOperation(20, SimulateMsgCommitOracleHash(am.keeper)),
		cosmossim.NewWeightedOperation(20, SimulateMsgRevealOracleReport(am.keeper)),
		cosmossim.NewWeightedOperation(5, SimulateDropOracleReveal(am.keeper)),
		cosmossim.NewWeightedOperation(3, SimulateOracleRoundInsufficient(am.keeper)),
	}
}

// gRPC service descriptor for the oracle message service.
// This enables RegisterServices to properly route commit/reveal messages.
var OracleMsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.oracle.v1.Msg",
	HandlerType: (*MsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SubmitOracleCommit",
			Handler:    _Msg_SubmitOracleCommit_Handler,
		},
		{
			MethodName: "RevealOracleReport",
			Handler:    _Msg_RevealOracleReport_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

// MsgServiceServer is the server API for the oracle Msg service.
type MsgServiceServer interface {
	SubmitOracleCommit(context.Context, *MsgCommitOracleHash) (*MsgCommitOracleHashResponse, error)
	RevealOracleReport(context.Context, *MsgRevealOracleReport) (*MsgRevealOracleReportResponse, error)
}

func _Msg_SubmitOracleCommit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCommitOracleHash)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).SubmitOracleCommit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.oracle.v1.Msg/SubmitOracleCommit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).SubmitOracleCommit(ctx, req.(*MsgCommitOracleHash))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_RevealOracleReport_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgRevealOracleReport)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).RevealOracleReport(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.oracle.v1.Msg/RevealOracleReport",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).RevealOracleReport(ctx, req.(*MsgRevealOracleReport))
	}
	return interceptor(ctx, in, info, handler)
}

