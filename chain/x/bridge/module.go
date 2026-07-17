package bridge

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

type AppModuleBasic struct{}

func (AppModuleBasic) Name() string { return ModuleName }
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}
func (AppModuleBasic) RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgBridgeIn{},
		&MsgBridgeOut{},
	)
}
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := GenesisState{
		Params: Params{
			StandardFinalityDepth:  15,
			LargeFinalityDepth:     50,
			LargeTransferThreshold: 5000000000,
			QuorumThreshold:        3,
			MaxUnlockPerBlock:      100000000000,
			SupplyCap:              "1000000000000",
		},
		Relayers:     []Relayer{},
		CosmosMinted: "0",
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
		return fmt.Errorf("failed to unmarshal bridge genesis state: %w", err)
	}
	p := state.Params
	supplyCap, ok := math.NewIntFromString(p.SupplyCap)
	if !ok || !supplyCap.IsPositive() {
		return fmt.Errorf("bridge supply_cap must be positive and valid, got: %s", p.SupplyCap)
	}
	if p.QuorumThreshold == 0 {
		return fmt.Errorf("bridge quorum_threshold must be positive")
	}
	if p.StandardFinalityDepth == 0 {
		return fmt.Errorf("bridge standard_finality_depth must be positive")
	}
	if p.LargeFinalityDepth == 0 {
		return fmt.Errorf("bridge large_finality_depth must be positive")
	}
	if p.CircuitBreakerAddress == "" {
		return fmt.Errorf("bridge circuit_breaker_address must be set")
	}
	if p.LockBoxAddress == "" {
		return fmt.Errorf("bridge lockbox_address must be set")
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
	cfg.MsgServer().RegisterService(&MsgServiceDesc, NewMsgServerImpl(am.keeper))
}
func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	am.keeper.RegisterInvariants(ir)
}
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var genesisState GenesisState
	// C2 FIX: Panic on malformed genesis — bridge params are security-critical
	if err := json.Unmarshal(data, &genesisState); err != nil {
		panic(fmt.Sprintf("failed to unmarshal bridge genesis state: %v", err))
	}
	am.keeper.SetParams(ctx, genesisState.Params)
	for _, r := range genesisState.Relayers {
		am.keeper.SetRelayer(ctx, r)
	}
	minted, ok := math.NewIntFromString(genesisState.CosmosMinted)
	if !ok {
		panic(fmt.Sprintf("invalid cosmos_minted in genesis: %s", genesisState.CosmosMinted))
	}
	am.keeper.SetCosmosMinted(ctx, minted)

	ctx.Logger().Info("bridge module initialized from genesis",
		"supply_cap", genesisState.Params.SupplyCap,
		"quorum", genesisState.Params.QuorumThreshold,
		"relayers", len(genesisState.Relayers),
		"circuit_breaker", genesisState.Params.CircuitBreakerAddress,
	)
	return nil
}
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genesisState := GenesisState{
		Params:       am.keeper.GetParams(ctx),
		Relayers:     am.keeper.GetRelayers(ctx),
		CosmosMinted: am.keeper.GetCosmosMinted(ctx).String(),
	}
	bz, _ := json.Marshal(genesisState)
	return bz
}
func (am AppModule) ConsensusVersion() uint64 { return 1 }
