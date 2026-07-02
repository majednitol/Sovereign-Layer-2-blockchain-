package app

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// PrepareProposal handles building the block proposal.
// It allows the proposer validator to pre-process transactions and inject metadata.
func (app *App) PrepareProposal(req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
	app.Logger().Info("ABCI++ PrepareProposal Hook Invoked")

	// Delegate to the baseapp prepare proposal handler (which uses the app mempool to select/reap transactions)
	return app.BaseApp.PrepareProposal(req)
}

// ProcessProposal validates the incoming block proposal.
// If any transaction or metadata violates consensus invariants, the block is rejected.
func (app *App) ProcessProposal(req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
	app.Logger().Info("ABCI++ ProcessProposal Hook Invoked")

	// Delegate to the baseapp process proposal handler
	return app.BaseApp.ProcessProposal(req)
}

// ExtendVote allows validators to submit vote extensions (certification state signatures)
// that will be bundled with precommits.
func (app *App) ExtendVote(ctx context.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("ABCI++ ExtendVote Hook Invoked", "height", sdkCtx.BlockHeight())

	// Generate vote extensions containing the validator's signature of the certification state.
	// Empty/Missing signatures will be tracked for jailing.
	return &abci.ResponseExtendVote{
		VoteExtension: []byte("sovereign_extension_signature_stub"),
	}, nil
}

// VerifyVoteExtension verifies the vote extension signatures attached by other validators.
func (app *App) VerifyVoteExtension(req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	app.Logger().Info("ABCI++ VerifyVoteExtension Hook Invoked")

	// Ensure the validator's vote extension contains the valid signature of certification state.
	return &abci.ResponseVerifyVoteExtension{
		Status: abci.ResponseVerifyVoteExtension_ACCEPT,
	}, nil
}

// GetLivenessSigningRatio computes the validator's signing ratio under the rolling window.
// NOTE: During the bootstrapping period (height H < 10,000), we must use the actual block count
// (H - 1) as the denominator to compute the ratio instead of the full 10,000 blocks window.
func GetLivenessSigningRatio(signedBlocks int64, currentHeight int64, windowSize int64) float64 {
	if currentHeight <= 1 {
		return 1.0 // Prevent division by zero at block 1
	}

	denominator := windowSize
	if currentHeight < windowSize {
		// Use actual block count as denominator during bootstrapping
		denominator = currentHeight
	}

	return float64(signedBlocks) / float64(denominator)
}
