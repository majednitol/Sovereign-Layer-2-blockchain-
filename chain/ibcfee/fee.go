package fee

import (
	"encoding/json"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sdk "github.com/cosmos/cosmos-sdk/types"
	porttypes "github.com/cosmos/ibc-go/v11/modules/core/05-port/types"
	ibcfeekeeper "github.com/cosmos/ibc-go/v11/modules/apps/fee/keeper"
	gwruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
)

type AppModuleBasic struct{}

var _ module.AppModuleBasic = AppModuleBasic{}

func (AppModuleBasic) Name() string { return "ibcfee" }
func (AppModuleBasic) RegisterLegacyAminoCodec(*codec.LegacyAmino) {}
func (AppModuleBasic) RegisterInterfaces(types.InterfaceRegistry) {}
func (AppModuleBasic) DefaultGenesis(codec.JSONCodec) json.RawMessage { return nil }
func (AppModuleBasic) ValidateGenesis(codec.JSONCodec, client.TxEncodingConfig, json.RawMessage) error { return nil }
func (AppModuleBasic) RegisterGRPCGatewayRoutes(client.Context, *gwruntime.ServeMux) {}

type AppModule struct {
	AppModuleBasic
	keeper ibcfeekeeper.Keeper
}

var _ module.AppModule = AppModule{}

func NewAppModule(k ibcfeekeeper.Keeper) AppModule {
	return AppModule{keeper: k}
}

func (AppModule) IsAppModule() {}
func (AppModule) IsOnePerModuleType() {}

func (AppModule) Name() string { return "ibcfee" }
func (AppModule) RegisterServices(module.Configurator) {}
func (AppModule) ConsensusVersion() uint64 { return 1 }

func (AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	return nil
}
func (AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	return nil
}

func NewIBCMiddleware(app porttypes.IBCModule, k ibcfeekeeper.Keeper) porttypes.IBCModule {
	return app
}
