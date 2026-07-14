package validator

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgFillValidatorSlot simulates MsgFillValidatorSlot message execution.
func SimulateMsgFillValidatorSlot(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgFillValidatorSlot", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		valAddr := sdk.ValAddress(simAccount.Address)
		k.SetValidatorActive(ctx, valAddr)

		return simtypes.NewOperationMsg(&MsgFillValidatorSlot{ValidatorAddress: valAddr.String()}, true, ""), nil, nil
	}
}

// SimulateMsgEjectValidator simulates MsgEjectValidator message execution.
func SimulateMsgEjectValidator(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgEjectValidator", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		valAddr := sdk.ValAddress(simAccount.Address)
		k.QueueEjection(ctx, valAddr)

		return simtypes.NewOperationMsg(&MsgEjectValidator{ValidatorAddress: valAddr.String()}, true, ""), nil, nil
	}
}

// SimulateMsgUpdatePartitionScheme simulates MsgUpdatePartitionScheme proposal.
func SimulateMsgUpdatePartitionScheme(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdatePartitionScheme", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)

		k.SetPartitionScheme(ctx, "equal-slots-30")

		return simtypes.NewOperationMsg(&MsgUpdatePartitionScheme{
			Authority: authority.String(),
			NewScheme: "equal-slots-30",
		}, true, ""), nil, nil
	}
}
