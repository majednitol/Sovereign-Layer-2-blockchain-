package gov_ext

import (
	"fmt"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgMigrateContracts simulates a MsgMigrateContracts transaction.
func SimulateMsgMigrateContracts(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgMigrateContracts", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)

		msg := MsgMigrateContracts{
			Authority:          authority.String(),
			ContractAddress:    sdk.AccAddress(accs[r.Intn(len(accs))].Address).String(),
			NewCodeID:          uint64(r.Intn(100) + 1),
			ExecutionDelaySecs: 604800, // 7 days
		}

		err := k.ExecuteProposal(ctx, &msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgMigrateContracts", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgUpdateValidatorSlot simulates a MsgUpdateValidatorSlot transaction.
func SimulateMsgUpdateValidatorSlot(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateValidatorSlot", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)

		msg := MsgUpdateValidatorSlot{
			Authority:     authority.String(),
			MaxValidators: uint32(r.Intn(50) + 1),
		}

		err := k.ExecuteProposal(ctx, &msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateValidatorSlot", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgUpdateMilestone simulates a MsgUpdateMilestone transaction.
func SimulateMsgUpdateMilestone(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateMilestone", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)

		msg := MsgUpdateMilestone{
			Authority:   authority.String(),
			MilestoneID: fmt.Sprintf("m_%d", r.Intn(1000)),
			TargetPrice: uint64(r.Intn(100000) + 50000),
		}

		err := k.ExecuteProposal(ctx, &msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateMilestone", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgUpdateOracleOperator simulates a MsgUpdateOracleOperator transaction.
func SimulateMsgUpdateOracleOperator(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateOracleOperator", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)
		operatorAddr := sdk.AccAddress(accs[r.Intn(len(accs))].Address)

		msg := MsgUpdateOracleOperator{
			Authority:       authority.String(),
			OperatorAddress: operatorAddr.String(),
			Active:          r.Intn(2) == 0,
		}

		err := k.ExecuteProposal(ctx, &msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateOracleOperator", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgUpdateWitnessRegistry simulates a MsgUpdateWitnessRegistry transaction.
func SimulateMsgUpdateWitnessRegistry(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateWitnessRegistry", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)
		witnessAddr := sdk.AccAddress(accs[r.Intn(len(accs))].Address)

		msg := MsgUpdateWitnessRegistry{
			Authority:      authority.String(),
			WitnessAddress: witnessAddr.String(),
			Active:         r.Intn(2) == 0,
			PubKey:         []byte("mock_witness_public_key_32_bytes_len"),
		}

		err := k.ExecuteProposal(ctx, &msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateWitnessRegistry", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgUpdateBridgeRelayerSet simulates a MsgUpdateBridgeRelayerSet transaction.
func SimulateMsgUpdateBridgeRelayerSet(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateBridgeRelayerSet", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		authority := sdk.AccAddress(simAccount.Address)
		relayerAddr := sdk.AccAddress(accs[r.Intn(len(accs))].Address)

		msg := MsgUpdateBridgeRelayerSet{
			Authority:      authority.String(),
			RelayerAddress: relayerAddr.String(),
			Active:         r.Intn(2) == 0,
			PubKey:         []byte("mock_relayer_public_key_33_bytes_len"),
		}

		err := k.ExecuteProposal(ctx, &msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgUpdateBridgeRelayerSet", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}
