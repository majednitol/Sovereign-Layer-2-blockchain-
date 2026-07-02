package certification

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgUpdateCertificationParams simulates update certification parameters.
func SimulateMsgUpdateCertificationParams(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateCertificationParams", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)

		params := Params{
			MaxConsecutiveRejections: int64(r.Intn(10) + 1),
			MissedExtensionLimit:     int64(r.Intn(20) + 1),
		}
		k.SetParams(ctx, params)

		return simtypes.NewOperationMsg(&MsgUpdateCertificationParams{
			Authority: authority.String(),
			Params:    params,
		}, true, ""), nil, nil
	}
}

// SimulateDropValidatorAttestation simulates dropping a validator's attestation status.
func SimulateDropValidatorAttestation(k Keeper, staking StakingKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var activeVals []sdk.ValAddress
		staking.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
			activeVals = append(activeVals, valAddr)
			return false
		})
		if len(activeVals) == 0 {
			return simtypes.NoOpMsg(ModuleName, "DropValidatorAttestation", "no active validators"), nil, nil
		}
		valAddr := activeVals[r.Intn(len(activeVals))]
		k.SetValidatorAttested(ctx, valAddr, false)
		return simtypes.NewOperationMsgBasic(ModuleName, "DropValidatorAttestation", "dropped attestation", false, nil), nil, nil
	}
}

// SimulateRestoreValidatorAttestation simulates restoring a validator's attestation status.
func SimulateRestoreValidatorAttestation(k Keeper, staking StakingKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var activeVals []sdk.ValAddress
		staking.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
			activeVals = append(activeVals, valAddr)
			return false
		})
		if len(activeVals) == 0 {
			return simtypes.NoOpMsg(ModuleName, "RestoreValidatorAttestation", "no active validators"), nil, nil
		}
		valAddr := activeVals[r.Intn(len(activeVals))]
		k.SetValidatorAttested(ctx, valAddr, true)
		return simtypes.NewOperationMsgBasic(ModuleName, "RestoreValidatorAttestation", "restored attestation", true, nil), nil, nil
	}
}
