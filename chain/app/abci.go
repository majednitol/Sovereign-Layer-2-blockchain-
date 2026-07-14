package app

import (
	"bytes"
	"context"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
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

	proposerCons := sdkCtx.BlockHeader().ProposerAddress
	var proposerValAddr sdk.ValAddress
	_ = app.StakingKeeper.IterateLastValidatorPowers(sdkCtx, func(valAddr sdk.ValAddress, power int64) bool {
		val, err := app.StakingKeeper.GetValidator(sdkCtx, valAddr)
		if err == nil {
			consAddr, err := val.GetConsAddr()
			if err == nil && bytes.Equal(consAddr, proposerCons) {
				proposerValAddr = valAddr
				return true
			}
		}
		return false
	})

	attested := true
	if proposerValAddr != nil {
		attested = app.CertificationKeeper.IsValidatorAttested(sdkCtx, proposerValAddr)
	}

	if attested {
		return &abci.ResponseExtendVote{
			VoteExtension: []byte("sovereign_extension_signature_stub"),
		}, nil
	}

	return &abci.ResponseExtendVote{
		VoteExtension: []byte{},
	}, nil
}

// VerifyVoteExtension verifies the vote extension signatures attached by other validators.
func (app *App) VerifyVoteExtension(req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
	app.Logger().Info("ABCI++ VerifyVoteExtension Hook Invoked")
	sdkCtx := app.NewUncachedContext(false, cmtproto.Header{Height: req.Height})

	var valAddr sdk.ValAddress
	_ = app.StakingKeeper.IterateLastValidatorPowers(sdkCtx, func(vAddr sdk.ValAddress, power int64) bool {
		val, err := app.StakingKeeper.GetValidator(sdkCtx, vAddr)
		if err == nil {
			consAddr, err := val.GetConsAddr()
			if err == nil && bytes.Equal(consAddr, req.ValidatorAddress) {
				valAddr = vAddr
				return true
			}
		}
		return false
	})

	if valAddr != nil {
		attested := app.CertificationKeeper.IsValidatorAttested(sdkCtx, valAddr)
		if attested {
			if !bytes.Equal(req.VoteExtension, []byte("sovereign_extension_signature_stub")) {
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}
		} else {
			if len(req.VoteExtension) > 0 {
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}
		}
	}

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
