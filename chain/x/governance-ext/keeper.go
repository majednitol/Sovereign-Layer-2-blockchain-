package gov_ext

import (
	"encoding/json"
	"fmt"

	"strings"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/chain/x/milestone"
)

type WasmKeeper interface {
	Execute(ctx sdk.Context, contractAddr sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error)
	QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
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
	govKeeper        *govkeeper.Keeper
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

func (k *Keeper) SetGovKeeper(govKeeper *govkeeper.Keeper) {
	k.govKeeper = govKeeper
}

func (k Keeper) SubmitProposal(ctx sdk.Context, messages []sdk.Msg, metadata string, title string, summary string, proposer sdk.AccAddress, expedited bool) (govv1.Proposal, error) {
	if k.govKeeper == nil {
		return govv1.Proposal{}, fmt.Errorf("govKeeper not set")
	}
	return k.govKeeper.SubmitProposal(ctx, messages, metadata, title, summary, proposer, expedited)
}

func (k Keeper) AddDeposit(ctx sdk.Context, proposalID uint64, depositor sdk.AccAddress, amount sdk.Coins) (bool, error) {
	if k.govKeeper == nil {
		return false, fmt.Errorf("govKeeper not set")
	}
	return k.govKeeper.AddDeposit(ctx, proposalID, depositor, amount)
}

func (k Keeper) AddVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress, options govv1.WeightedVoteOptions, metadata string) error {
	if k.govKeeper == nil {
		return fmt.Errorf("govKeeper not set")
	}
	return k.govKeeper.AddVote(ctx, proposalID, voter, options, metadata)
}

func (k Keeper) TallyProposal(ctx sdk.Context, proposalID uint64) (passes bool, burnDeposits bool, tallyResults govv1.TallyResult, err error) {
	if k.govKeeper == nil {
		return false, false, govv1.TallyResult{}, fmt.Errorf("govKeeper not set")
	}
	proposal, err := k.govKeeper.Proposals.Get(ctx, proposalID)
	if err != nil {
		return false, false, govv1.TallyResult{}, err
	}
	return k.govKeeper.Tally(ctx, proposal)
}

func (k Keeper) HasGovKeeper() bool {
	return k.govKeeper != nil
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
	if err := json.Unmarshal(bz, &params); err != nil {
		panic(fmt.Sprintf("failed to unmarshal govext params: %v", err))
	}
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal govext params: %v", err))
	}
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

	// 2. Perform Constitution check via Wasm contract query if not bypassed
	if !bypass && k.wasmKeeper != nil && len(k.constitutionAddr) > 0 {
		// Call Constitution check with serialized proposal payload
		summaryBytes, err := json.Marshal(proposal)
		if err != nil {
			return fmt.Errorf("failed to marshal proposal: %w", err)
		}
		summaryStr := string(summaryBytes)

		type checkProposalMsg struct {
			CheckProposal struct {
				ProposalType string `json:"proposal_type"`
				Summary      string `json:"summary"`
			} `json:"check_proposal"`
		}
		var msgPayload checkProposalMsg
		msgPayload.CheckProposal.ProposalType = fmt.Sprintf("%T", proposal)
		msgPayload.CheckProposal.Summary = summaryStr
		checkMsg, err := json.Marshal(msgPayload)
		if err != nil {
			return fmt.Errorf("failed to marshal constitution check payload: %w", err)
		}

		resBytes, queryErr := k.wasmKeeper.QuerySmart(ctx, k.constitutionAddr, checkMsg)
		if queryErr != nil {
			// H-03: Circuit-breaker fallback if contract query fails due to pause or unavailability
			errMsg := queryErr.Error()
			if strings.Contains(errMsg, "paused") || strings.Contains(errMsg, "unavailable") {
				ctx.Logger().Warn("constitution contract query failed (paused/unavailable); proceeding in degraded mode", "error", queryErr)
			} else {
				return fmt.Errorf("constitution compliance check failed: %w", queryErr)
			}
		} else {
			type checkProposalResponse struct {
				IsValid bool   `json:"is_valid"`
				Reason  string `json:"reason"`
			}
			var res checkProposalResponse
			if err := json.Unmarshal(resBytes, &res); err != nil {
				return fmt.Errorf("failed to unmarshal constitution query response: %w", err)
			}
			if !res.IsValid {
				return fmt.Errorf("constitution compliance check failed: %s", res.Reason)
			}
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
