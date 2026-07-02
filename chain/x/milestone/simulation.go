package milestone

import (
	"fmt"
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgCreateMilestone simulates milestone creation.
func SimulateMsgCreateMilestone(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgCreateMilestone", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		creator := sdk.AccAddress(simAccount.Address)

		id := fmt.Sprintf("milestone_%d", r.Intn(10000))
		feedID := "BTC_USD"
		targetPrice := uint64(100000)
		duration := int64(100)
		vestingAddr := sdk.AccAddress(accs[r.Intn(len(accs))].Address).String()

		m := Milestone{
			ID:                 id,
			FeedID:             feedID,
			TargetPrice:        targetPrice,
			RemainingBlocks:    duration,
			State:              StatePending,
			VestingPoolAddress: vestingAddr,
		}
		k.SetMilestone(ctx, m)

		return simtypes.NewOperationMsg(&MsgCreateMilestone{
			Creator:            creator.String(),
			ID:                 id,
			FeedID:             feedID,
			TargetPrice:        targetPrice,
			DurationBlocks:     duration,
			VestingPoolAddress: vestingAddr,
		}, true, ""), nil, nil
	}
}

// SimulateMsgAchieveMilestone simulates milestone achievement by updating the oracle price.
func SimulateMsgAchieveMilestone(k Keeper, ok OracleKeeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var pendingMilestones []Milestone
		k.IterateMilestones(ctx, func(m Milestone) bool {
			if m.State == StatePending {
				pendingMilestones = append(pendingMilestones, m)
			}
			return false
		})
		if len(pendingMilestones) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgAchieveMilestone", "no pending milestones"), nil, nil
		}
		m := pendingMilestones[r.Intn(len(pendingMilestones))]

		m.State = StateAchieved
		k.SetMilestone(ctx, m)
		k.triggerVestingPayout(ctx, m)

		return simtypes.NewOperationMsgBasic(ModuleName, "MsgAchieveMilestone", m.ID, true, nil), nil, nil
	}
}

// SimulateMilestoneExpiry simulates milestone expiry by counting down remaining blocks.
func SimulateMilestoneExpiry(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var pendingMilestones []Milestone
		k.IterateMilestones(ctx, func(m Milestone) bool {
			if m.State == StatePending {
				pendingMilestones = append(pendingMilestones, m)
			}
			return false
		})
		if len(pendingMilestones) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MilestoneExpiry", "no pending milestones"), nil, nil
		}
		m := pendingMilestones[r.Intn(len(pendingMilestones))]

		m.State = StateExpired
		m.RemainingBlocks = 0
		k.SetMilestone(ctx, m)

		ctx.EventManager().EmitEvent(sdk.NewEvent(
			"milestone_expired",
			sdk.NewAttribute("milestone_id", m.ID),
		))

		return simtypes.NewOperationMsgBasic(ModuleName, "MilestoneExpiry", m.ID, true, nil), nil, nil
	}
}

// SimulateMilestoneStaleRecovery simulates a stale feed and recovery.
func SimulateMilestoneStaleRecovery(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		var staleBlockedMilestones []Milestone
		k.IterateMilestones(ctx, func(m Milestone) bool {
			if m.State == StateStaleBlocked {
				staleBlockedMilestones = append(staleBlockedMilestones, m)
			}
			return false
		})

		if len(staleBlockedMilestones) == 0 {
			var pendingMilestones []Milestone
			k.IterateMilestones(ctx, func(m Milestone) bool {
				if m.State == StatePending {
					pendingMilestones = append(pendingMilestones, m)
				}
				return false
			})
			if len(pendingMilestones) == 0 {
				return simtypes.NoOpMsg(ModuleName, "MilestoneStaleRecovery", "no milestones available"), nil, nil
			}
			m := pendingMilestones[r.Intn(len(pendingMilestones))]
			m.State = StateStaleBlocked
			k.SetMilestone(ctx, m)
			return simtypes.NewOperationMsgBasic(ModuleName, "MilestoneStaleRecovery", m.ID, true, nil), nil, nil
		}

		m := staleBlockedMilestones[r.Intn(len(staleBlockedMilestones))]
		m.State = StatePending
		k.SetMilestone(ctx, m)

		return simtypes.NewOperationMsgBasic(ModuleName, "MilestoneStaleRecovery", m.ID, true, nil), nil, nil
	}
}
