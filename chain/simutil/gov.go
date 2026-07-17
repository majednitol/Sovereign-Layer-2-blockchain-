package simutil

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// SimGov is a helper wrapper for simulating governance actions in tests.
// It routes proposals through a governance-like lifecycle instead of
// executing them directly via keeper bypass.
type SimGov struct {
	Keeper ExecuteProposalKeeper
}

// ExecuteProposalKeeper is the minimal interface a keeper must implement
// to be usable in governance simulations.
type ExecuteProposalKeeper interface {
	ExecuteProposal(ctx sdk.Context, msg sdk.Msg) error
}

// GovKeeper is an optional interface for keepers that support the full
// x/gov proposal lifecycle (submit → deposit → vote → tally → execute).
type GovKeeper interface {
	SubmitProposal(ctx sdk.Context, messages []sdk.Msg, metadata string, title string, summary string, proposer sdk.AccAddress, expedited bool) (govv1.Proposal, error)
	AddDeposit(ctx sdk.Context, proposalID uint64, depositor sdk.AccAddress, amount sdk.Coins) (bool, error)
	AddVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress, options govv1.WeightedVoteOptions, metadata string) error
	TallyProposal(ctx sdk.Context, proposalID uint64) (passes bool, burnDeposits bool, tallyResults govv1.TallyResult, err error)
	HasGovKeeper() bool
}

func NewSimGov(keeper ExecuteProposalKeeper) SimGov {
	return SimGov{Keeper: keeper}
}

// ProposeAndExecute simulates the full lifecycle of a governance proposal.
//
// If the keeper also implements GovKeeper (i.e., it has access to x/gov),
// the proposal is submitted, deposited, voted on, tallied, and only then
// executed — matching the real governance path.
//
// Otherwise, it falls back to direct execution with a warning log,
// clearly indicating the governance path was NOT exercised.
func (s SimGov) ProposeAndExecute(ctx sdk.Context, msg sdk.Msg) error {
	if govKeeper, ok := s.Keeper.(GovKeeper); ok && govKeeper.HasGovKeeper() {
		return s.proposeViaGov(ctx, govKeeper, msg)
	}

	// Fallback: direct execution (the keeper does not implement GovKeeper).
	// This is an explicit, auditable compromise — the simulation does NOT
	// exercise the governance vote/tally path. It is functionally equivalent
	// to the old k.ExecuteProposal() call but isolated behind this wrapper
	// so that upgrading to the full path requires only implementing GovKeeper.
	fmt.Printf("[SimGov] WARNING: Keeper does not implement GovKeeper interface. "+
		"Falling back to direct ExecuteProposal for msg type %T. "+
		"The governance submit/deposit/vote/tally path is NOT exercised.\n", msg)
	return s.Keeper.ExecuteProposal(ctx, msg)
}

// proposeViaGov routes the message through x/gov's full proposal lifecycle.
func (s SimGov) proposeViaGov(ctx sdk.Context, govKeeper GovKeeper, msg sdk.Msg) error {
	// Use a deterministic simulation proposer address
	proposer := sdk.AccAddress([]byte("simgov_proposer_addr"))

	// 1. Submit proposal
	messages := []sdk.Msg{msg}
	proposal, err := govKeeper.SubmitProposal(
		ctx,
		messages,
		"",
		fmt.Sprintf("SimGov auto-proposal for %T", msg),
		fmt.Sprintf("Automated governance simulation proposal wrapping message type %T", msg),
		proposer,
		false,
	)
	if err != nil {
		return fmt.Errorf("SimGov: failed to submit proposal: %w", err)
	}

	// 2. Deposit minimum (use 1ucsov as simulation deposit)
	deposit := sdk.NewCoins(sdk.NewInt64Coin("ucsov", 1000000))
	activated, err := govKeeper.AddDeposit(ctx, proposal.Id, proposer, deposit)
	if err != nil {
		return fmt.Errorf("SimGov: failed to add deposit to proposal %d: %w", proposal.Id, err)
	}
	if !activated {
		fmt.Printf("[SimGov] Proposal %d deposited but not yet activated (below min deposit)\n", proposal.Id)
	}

	// 3. Vote YES
	voteOptions := govv1.NewNonSplitVoteOption(govv1.OptionYes)
	err = govKeeper.AddVote(ctx, proposal.Id, proposer, voteOptions, "")
	if err != nil {
		return fmt.Errorf("SimGov: failed to vote on proposal %d: %w", proposal.Id, err)
	}

	// 4. Tally
	passes, _, _, err := govKeeper.TallyProposal(ctx, proposal.Id)
	if err != nil {
		return fmt.Errorf("SimGov: failed to tally proposal %d: %w", proposal.Id, err)
	}
	if !passes {
		return fmt.Errorf("SimGov: proposal %d did not pass tally", proposal.Id)
	}

	// 5. Execute the actual message (proposal passed)
	fmt.Printf("[SimGov] Proposal %d passed tally. Executing via keeper.\n", proposal.Id)
	return s.Keeper.ExecuteProposal(ctx, msg)
}
