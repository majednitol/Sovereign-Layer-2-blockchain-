package oracle

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgCommitOracleHash simulates MsgCommitOracleHash transaction.
func SimulateMsgCommitOracleHash(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgCommitOracleHash", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		operator := sdk.ValAddress(simAccount.Address)

		feedID := "BTC_USD"
		roundID := uint64(r.Intn(100) + 1)
		value := uint64(90000 + r.Intn(10000))
		nonce := "random_nonce_value"

		hash := ComputeCommitHash(operator.String(), feedID, roundID, value, nonce)
		k.CommitHash(ctx, operator.String(), feedID, roundID, hash)

		return simtypes.NewOperationMsg(&MsgCommitOracleHash{
			Operator: operator.String(),
			FeedID:   feedID,
			RoundID:  roundID,
			Hash:     hash,
		}, true, ""), nil, nil
	}
}

// SimulateMsgRevealOracleReport simulates MsgRevealOracleReport transaction.
func SimulateMsgRevealOracleReport(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgRevealOracleReport", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		operator := sdk.ValAddress(simAccount.Address)

		feedID := "BTC_USD"
		roundID := uint64(r.Intn(100) + 1)
		value := uint64(90000 + r.Intn(10000))
		nonce := "random_nonce_value"

		// Commit first to satisfy checks
		hash := ComputeCommitHash(operator.String(), feedID, roundID, value, nonce)
		k.CommitHash(ctx, operator.String(), feedID, roundID, hash)

		err := k.RevealReport(ctx, operator.String(), feedID, roundID, value, nonce)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgRevealOracleReport", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&MsgRevealOracleReport{
			Operator: operator.String(),
			FeedID:   feedID,
			RoundID:  roundID,
			Value:    value,
			Nonce:    nonce,
		}, true, ""), nil, nil
	}
}

// SimulateDropOracleReveal simulates an operator committing but not revealing.
func SimulateDropOracleReveal(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "SimulateDropOracleReveal", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		operator := sdk.ValAddress(simAccount.Address)

		feedID := "BTC_USD"
		roundID := uint64(r.Intn(100) + 1)
		value := uint64(90000 + r.Intn(10000))
		nonce := "random_nonce_value"

		// We commit the hash but we do NOT call RevealReport
		hash := ComputeCommitHash(operator.String(), feedID, roundID, value, nonce)
		_ = k.CommitHash(ctx, operator.String(), feedID, roundID, hash)

		return simtypes.NewOperationMsg(&MsgCommitOracleHash{
			Operator: operator.String(),
			FeedID:   feedID,
			RoundID:  roundID,
			Hash:     hash,
		}, true, "committed but dropped reveal"), nil, nil
	}
}

// SimulateOracleRoundInsufficient tries to aggregate a round with no reveals.
func SimulateOracleRoundInsufficient(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		feedID := "BTC_USD"
		roundID := uint64(r.Intn(1000) + 1000) // use high round ID to ensure no previous reveals

		_, err := k.AggregateRound(ctx, feedID, roundID)
		if err == nil {
			return simtypes.NewOperationMsg(&MsgRevealOracleReport{}, false, "expected failure for insufficient commits"), nil, nil
		}

		return simtypes.NewOperationMsg(&MsgRevealOracleReport{FeedID: feedID, RoundID: roundID}, true, "insufficient round simulation passed"), nil, nil
	}
}
