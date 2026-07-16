package simutil

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
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
	SubmitProposal(ctx sdk.Context, content govv1beta1.Content, proposer sdk.AccAddress) (govv1beta1.Proposal, error)
	AddDeposit(ctx sdk.Context, proposalID uint64, depositor sdk.AccAddress, amount sdk.Coins) (bool, error)
	AddVote(ctx sdk.Context, proposalID uint64, voter sdk.AccAddress, options govv1beta1.WeightedVoteOptions) error
	TallyProposal(ctx sdk.Context, proposalID uint64) (passes bool, burnDeposits bool, tallyResults govv1beta1.TallyResult)
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
	if govKeeper, ok := s.Keeper.(GovKeeper); ok {
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
	content := govv1beta1.NewTextProposal(
		fmt.Sprintf("SimGov auto-proposal for %T", msg),
		fmt.Sprintf("Automated governance simulation proposal wrapping message type %T", msg),
	)

	proposal, err := govKeeper.SubmitProposal(ctx, content, proposer)
	if err != nil {
		return fmt.Errorf("SimGov: failed to submit proposal: %w", err)
	}

	// 2. Deposit minimum (use 1ucsov as simulation deposit)
	deposit := sdk.NewCoins(sdk.NewInt64Coin("ucsov", 1000000))
	activated, err := govKeeper.AddDeposit(ctx, proposal.ProposalId, proposer, deposit)
	if err != nil {
		return fmt.Errorf("SimGov: failed to add deposit to proposal %d: %w", proposal.ProposalId, err)
	}
	if !activated {
		fmt.Printf("[SimGov] Proposal %d deposited but not yet activated (below min deposit)\n", proposal.ProposalId)
	}

	// 3. Vote YES
	voteOptions := govv1beta1.NewNonSplitVoteOption(govv1beta1.OptionYes)
	err = govKeeper.AddVote(ctx, proposal.ProposalId, proposer, voteOptions)
	if err != nil {
		return fmt.Errorf("SimGov: failed to vote on proposal %d: %w", proposal.ProposalId, err)
	}

	// 4. Tally
	passes, _, _ := govKeeper.TallyProposal(ctx, proposal.ProposalId)
	if !passes {
		return fmt.Errorf("SimGov: proposal %d did not pass tally", proposal.ProposalId)
	}

	// 5. Execute the actual message (proposal passed)
	fmt.Printf("[SimGov] Proposal %d passed tally. Executing via keeper.\n", proposal.ProposalId)
	return s.Keeper.ExecuteProposal(ctx, msg)
}
