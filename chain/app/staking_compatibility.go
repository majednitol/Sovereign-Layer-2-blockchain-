package app

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// StakingCompatibilityKeeper defines the wrapper shim required to translate equalized
// slot-based validator powers to the structure expected by standard Cosmos SDK modules.
type StakingCompatibilityKeeper struct {
	stakingKeeper StakingKeeperStub
	distrKeeper   DistrKeeperStub
	MaxValidators uint32
}

// StakingKeeperStub represents the underlying staking keeper interface
type StakingKeeperStub interface {
	GetLastValidatorPower(ctx sdk.Context, valAddr sdk.ValAddress) int64
	GetLastTotalPower(ctx sdk.Context) math.Int
}

// DistrKeeperStub is the distribution keeper interface for rewards allocation hooks
type DistrKeeperStub interface {
	AllocateTokensToValidator(ctx sdk.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins)
	GetValidatorOutstandingRewards(ctx sdk.Context, valAddr sdk.ValAddress) distrtypes.ValidatorOutstandingRewards
}

// GetEqualizedValidatorPower overrides the stake-weighted power of active validators.
// All active validator slots return exactly 1,000,000 voting power (per ADR-001).
func (k StakingCompatibilityKeeper) GetEqualizedValidatorPower(ctx sdk.Context, valAddr sdk.ValAddress) int64 {
	// If the validator has any raw power, it is considered in the active set.
	// Return the fixed equalized power constant.
	rawPower := k.stakingKeeper.GetLastValidatorPower(ctx, valAddr)
	if rawPower > 0 {
		return 1_000_000 // Equal voting power for active slots
	}
	return 0
}

// GetEqualizedTotalPower returns the cumulative voting power of the active set.
func (k StakingCompatibilityKeeper) GetEqualizedTotalPower(_ sdk.Context) math.Int {
	// Total consensus power = MaxValidators (30) * 1,000,000
	return math.NewInt(int64(k.MaxValidators) * 1_000_000)
}

// AllocateTokens implements the equal-slot reward split hook.
// Rather than using stake-weighted distribution, each of the 30 active validator
// slots receives an identical share of the block provision.
//
// Called by the distribution module's BeginBlock hook on each new block.
func (k StakingCompatibilityKeeper) AllocateTokens(
	ctx sdk.Context,
	totalRewards sdk.DecCoins,
	activeValidators []stakingtypes.ValidatorI,
) {
	if len(activeValidators) == 0 {
		return
	}

	// Divide the total block provision equally among active validator slots.
	// Use integer division with remainder going to the first validator.
	slotCount := int64(len(activeValidators))
	perValidatorRewards := totalRewards.QuoDec(math.LegacyNewDec(slotCount))

	for i, val := range activeValidators {
		rewardSlice := perValidatorRewards
		// Give the rounding remainder to the first validator
		if i == 0 {
			allocated := perValidatorRewards.MulDec(math.LegacyNewDec(slotCount))
			remainder := totalRewards.Sub(allocated)
			rewardSlice = perValidatorRewards.Add(remainder...)
		}
		if k.distrKeeper != nil {
			k.distrKeeper.AllocateTokensToValidator(ctx, val, rewardSlice)
		}
	}
}

// GetHistoricalEqualizedPower returns the equalized voting power for a validator
// for use in HistoricalInfo storage required by IBC light client compatibility.
// All active validators are stored with exactly 1,000,000 power regardless of stake.
func GetHistoricalEqualizedPower(rawPower int64) int64 {
	if rawPower > 0 {
		return 1_000_000
	}
	return 0
}

// OverrideHistoricalInfo intercepts the historical info stored for the current block
// and overrides the validator powers to 1,000,000 to maintain IBC light client compatibility.
func (app *App) OverrideHistoricalInfo(ctx sdk.Context) {
	height := ctx.BlockHeight()
	hi, err := app.StakingKeeper.GetHistoricalInfo(ctx, height)
	if err != nil {
		return
	}
	powerReduction := app.StakingKeeper.PowerReduction(ctx)
	for i := range hi.Valset {
		if hi.Valset[i].IsBonded() {
			hi.Valset[i].Tokens = powerReduction.Mul(math.NewInt(1_000_000))
		}
	}
	_ = app.StakingKeeper.SetHistoricalInfo(ctx, height, &hi)
}

// GetEqualizedValidatorUpdates overrides all non-zero validator consensus updates
// returned from EndBlock to exactly 1,000,000 power.
func (app *App) GetEqualizedValidatorUpdates(ctx sdk.Context, updates []abci.ValidatorUpdate) []abci.ValidatorUpdate {
	equalizedUpdates := make([]abci.ValidatorUpdate, len(updates))
	for i, update := range updates {
		equalizedUpdates[i] = update
		if update.Power > 0 {
			equalizedUpdates[i].Power = 1_000_000
		}
	}
	return equalizedUpdates
}
