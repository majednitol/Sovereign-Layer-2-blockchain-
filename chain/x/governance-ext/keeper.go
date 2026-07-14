package gov_ext

import (
	"encoding/json"
	"fmt"

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
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ParamsKey)
	if bz == nil {
		return Params{
			MinGasLimit: 100000,
			MaxGasLimit: 2000000,
		}
	}
	var params Params
	_ = json.Unmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(params)
	store.Set(ParamsKey, bz)
}

// ExecuteProposal executes a governance proposal.
// It verifies Constitution compliance via the Constitution contract query,
// except for MsgMigrateContracts and MsgUpdateGasLimit (gas limit parameter updates) which bypass the check.
func (k Keeper) ExecuteProposal(ctx sdk.Context, proposal sdk.Msg) error {
	// 1. Check if proposal bypasses the Constitution check
	bypass := false
	switch msg := proposal.(type) {
	case *MsgMigrateContracts:
		// Mandatory 7-day delay (604,800 seconds)
		if msg.ExecutionDelaySecs < 604800 {
			return fmt.Errorf("MsgMigrateContracts must have a mandatory 7-day execution delay (got %d)", msg.ExecutionDelaySecs)
		}
		bypass = true
	case *MsgUpdateGasLimit:
		// Gas limit bounds enforcement [100,000 - 2,000,000]
		params := k.GetParams(ctx)
		if msg.GasLimit < params.MinGasLimit || msg.GasLimit > params.MaxGasLimit {
			return fmt.Errorf("gas limit %d out of bounds [%d - %d]", msg.GasLimit, params.MinGasLimit, params.MaxGasLimit)
		}
		bypass = true
	}

	// 2. Perform Constitution check via Wasm contract call if not bypassed
	if !bypass && k.wasmKeeper != nil && len(k.constitutionAddr) > 0 {
		// Call Constitution check with serialized proposal payload
		type checkProposalMsg struct {
			CheckProposal struct {
				ProposalType string  `json:"proposal_type"`
				Proposal     sdk.Msg `json:"proposal"`
			} `json:"check_proposal"`
		}
		var msgPayload checkProposalMsg
		msgPayload.CheckProposal.ProposalType = fmt.Sprintf("%T", proposal)
		msgPayload.CheckProposal.Proposal = proposal
		checkMsg, err := json.Marshal(msgPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal constitution check payload: %w", err)
		}

		_, err = k.wasmKeeper.Execute(ctx, k.constitutionAddr, sdk.AccAddress([]byte("govext_module")), checkMsg, nil)
		if err != nil {
			return fmt.Errorf("constitution compliance check failed: %w", err)
		}
	}

	// 3. Execute the proposal mutation on the target module keeper
	switch msg := proposal.(type) {
	case *MsgUpdateValidatorSlot:
		if k.validatorKeeper != nil {
			k.validatorKeeper.SetMaxValidators(ctx, msg.MaxValidators)
		}
	case *MsgUpdateMilestone:
		if k.milestoneKeeper != nil {
			m, ok := k.milestoneKeeper.GetMilestone(ctx, msg.MilestoneID)
			if ok {
				m.TargetPrice = msg.TargetPrice
				k.milestoneKeeper.SetMilestone(ctx, m)
			} else {
				return fmt.Errorf("milestone ID %s not found", msg.MilestoneID)
			}
		}
	case *MsgUpdateOracleOperator:
		if k.oracleKeeper != nil {
			k.oracleKeeper.SetOperatorActive(ctx, msg.OperatorAddress, msg.Active)
		}
	case *MsgUpdateWitnessRegistry:
		if k.settlementKeeper != nil {
			if msg.Active {
				k.settlementKeeper.SetWitnessPubKey(ctx, msg.WitnessAddress, msg.PubKey)
			} else {
				k.settlementKeeper.DeleteWitnessPubKey(ctx, msg.WitnessAddress)
			}
		}
	case *MsgUpdateBridgeRelayerSet:
		if k.bridgeKeeper != nil {
			if msg.Active {
				rel := bridge.Relayer{
					Address: msg.RelayerAddress,
					PubKey:  msg.PubKey,
				}
				k.bridgeKeeper.SetRelayer(ctx, rel)
			} else {
				k.bridgeKeeper.DeleteRelayer(ctx, msg.RelayerAddress)
			}
		}
	}

	// 4. Emit success event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"proposal_executed",
		sdk.NewAttribute("proposal_type", fmt.Sprintf("%T", proposal)),
	))

	return nil
}
