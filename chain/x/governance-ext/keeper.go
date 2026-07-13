package gov_ext

import (
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/chain/x/milestone"
)

type WasmKeeper interface {
	Execute(ctx sdk.Context, contractAddr sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error)
}

type ValidatorKeeper interface {
	SetMaxValidators(ctx sdk.Context, max uint32)
}

type MilestoneKeeper interface {
	GetMilestone(ctx sdk.Context, milestoneID string) (milestone.Milestone, bool)
	SetMilestone(ctx sdk.Context, m milestone.Milestone)
}

type OracleKeeper interface {
	SetOperatorActive(ctx sdk.Context, operator string, active bool)
}

type SettlementKeeper interface {
	SetWitnessPubKey(ctx sdk.Context, witnessID string, pubKey []byte)
	DeleteWitnessPubKey(ctx sdk.Context, witnessID string)
}

type BridgeKeeper interface {
	SetRelayer(ctx sdk.Context, r bridge.Relayer)
	DeleteRelayer(ctx sdk.Context, address string)
}

type Keeper struct {
	storeKey         storetypes.StoreKey
	cdc              codec.BinaryCodec
	wasmKeeper       WasmKeeper
	constitutionAddr sdk.AccAddress
	validatorKeeper  ValidatorKeeper
	milestoneKeeper  MilestoneKeeper
	oracleKeeper     OracleKeeper
	settlementKeeper SettlementKeeper
	bridgeKeeper     BridgeKeeper
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	wasmKeeper WasmKeeper,
	constitutionAddr sdk.AccAddress,
	validatorKeeper ValidatorKeeper,
	milestoneKeeper MilestoneKeeper,
	oracleKeeper OracleKeeper,
	settlementKeeper SettlementKeeper,
	bridgeKeeper BridgeKeeper,
) Keeper {
	return Keeper{
		storeKey:         storeKey,
		cdc:              cdc,
		wasmKeeper:       wasmKeeper,
		constitutionAddr: constitutionAddr,
		validatorKeeper:  validatorKeeper,
		milestoneKeeper:  milestoneKeeper,
		oracleKeeper:     oracleKeeper,
		settlementKeeper: settlementKeeper,
		bridgeKeeper:     bridgeKeeper,
	}
}

func (k Keeper) GetParams(ctx sdk.Context) Params {
	return Params{
		MinGasLimit: 100000,
		MaxGasLimit: 2000000,
	}
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {}

func (k Keeper) ExecuteProposal(ctx sdk.Context, proposal sdk.Msg) error {
	return nil
}
