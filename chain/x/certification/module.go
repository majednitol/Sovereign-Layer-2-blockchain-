package certification

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
		&MsgUpdateCertificationParams{},
	)
}
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	defaultState := CertGenesisState{
		Params: Params{
			MaxConsecutiveRejections: 5,
			MissedExtensionLimit:    10,
		},
		DegradedMode:              false,
		ConsecutiveRejectionCount: 0,
		AttestedValidators:        []string{},
	}
	bz, _ := json.Marshal(defaultState)
	return bz
}
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	if bz == nil {
		return nil
	}
	var state CertGenesisState
	if err := json.Unmarshal(bz, &state); err != nil {
		return fmt.Errorf("failed to unmarshal certification genesis state: %w", err)
	}
	if state.Params.MaxConsecutiveRejections <= 0 {
		return fmt.Errorf("certification max_consecutive_rejections must be positive")
	}
	if state.Params.MissedExtensionLimit <= 0 {
		return fmt.Errorf("certification missed_extension_limit must be positive")
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

// C3 FIX: Wire MsgServer so governance can update certification params
func (am AppModule) RegisterServices(cfg module.Configurator) {
	govAuthority := authtypes.NewModuleAddress("gov").String()
	msgServer := NewMsgServerImpl(am.keeper, govAuthority)
	cfg.MsgServer().RegisterService(&CertMsgServiceDesc, msgServer)
}

func (am AppModule) RegisterInvariants(ir sdk.InvariantRegistry) {
	am.keeper.RegisterInvariants(ir)
}

// H1 FIX: Load certification state from genesis
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	if data == nil {
		return nil
	}
	var state CertGenesisState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(fmt.Sprintf("failed to unmarshal certification genesis state: %v", err))
	}

	am.keeper.SetParams(ctx, state.Params)
	am.keeper.SetDegradedMode(ctx, state.DegradedMode)
	am.keeper.SetConsecutiveRejectionCount(ctx, state.ConsecutiveRejectionCount)

	if len(state.AttestedValidators) == 0 && am.keeper.stakingKeeper != nil {
		_ = am.keeper.stakingKeeper.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
			am.keeper.SetValidatorAttested(ctx, valAddr, true)
			return false
		})
	} else {
		for _, valAddrStr := range state.AttestedValidators {
			valAddr, err := sdk.ValAddressFromBech32(valAddrStr)
			if err == nil {
				am.keeper.SetValidatorAttested(ctx, valAddr, true)
			}
		}
	}

	ctx.Logger().Info("certification module initialized from genesis",
		"max_consecutive_rejections", state.Params.MaxConsecutiveRejections,
		"missed_extension_limit", state.Params.MissedExtensionLimit,
		"degraded_mode", state.DegradedMode,
		"attested_validators", len(state.AttestedValidators),
	)
	return nil
}

// H1 FIX: Export certification state for chain snapshots
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	state := CertGenesisState{
		Params:                    am.keeper.GetParams(ctx),
		DegradedMode:              am.keeper.IsDegradedMode(ctx),
		ConsecutiveRejectionCount: am.keeper.GetConsecutiveRejectionCount(ctx),
		AttestedValidators:        am.keeper.GetAllAttestedValidators(ctx),
	}
	bz, err := json.Marshal(state)
	if err != nil {
		ctx.Logger().Error("failed to marshal certification genesis state", "error", err)
		return nil
	}
	return bz
}
func (am AppModule) ConsensusVersion() uint64 { return 1 }

func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	// Check if a proposal was rejected in this block
	lastProposalRejected := false
	for _, event := range sdkCtx.EventManager().Events() {
		if event.Type == "proposal_rejected" {
			lastProposalRejected = true
		}
	}
	am.keeper.EndBlocker(sdkCtx, lastProposalRejected)
	return nil, nil
}

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {}

// RegisterStoreDecoder registers a decoder for module's types
func (AppModule) RegisterStoreDecoder(sdr simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the module's simulation operations with their respective weight.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{
		cosmossim.NewWeightedOperation(15, SimulateDropValidatorAttestation(am.keeper, am.keeper.stakingKeeper)),
		cosmossim.NewWeightedOperation(15, SimulateRestoreValidatorAttestation(am.keeper, am.keeper.stakingKeeper)),
		cosmossim.NewWeightedOperation(3, SimulateMsgUpdateCertificationParams(am.keeper)),
	}
}

