package certification

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error)
	IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) (stop bool)) error
}

type SlashingKeeper interface {
	Slash(ctx context.Context, consAddr sdk.ConsAddress, fraction math.LegacyDec, power, distributionHeight int64) error
	Jail(ctx context.Context, consAddr sdk.ConsAddress) error
}

type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	stakingKeeper  StakingKeeper
	slashingKeeper SlashingKeeper
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	stakingKeeper StakingKeeper,
	slashingKeeper SlashingKeeper,
) Keeper {
	return Keeper{
		storeKey:       storeKey,
		cdc:            cdc,
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
	}
}

func (k Keeper) GetConsecutiveRejectionCount(ctx sdk.Context) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ConsecutiveRejectionKey)
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

func (k Keeper) SetConsecutiveRejectionCount(ctx sdk.Context, count int64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(count))
	store.Set(ConsecutiveRejectionKey, bz)
}

func (k Keeper) IsDegradedMode(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(DegradedModeKey)
	if bz == nil {
		return false
	}
	return bz[0] == 0x01
}

func (k Keeper) SetDegradedMode(ctx sdk.Context, degraded bool) {
	store := ctx.KVStore(k.storeKey)
	if degraded {
		store.Set(DegradedModeKey, []byte{0x01})
	} else {
		store.Set(DegradedModeKey, []byte{0x00})
	}
}

func (k Keeper) GetParams(ctx sdk.Context) Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ParamsKey)
	if bz == nil {
		return Params{
			MaxConsecutiveRejections: 5,
			MissedExtensionLimit:     10,
		}
	}
	var params Params
	_ = json.Unmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(params)
	store.Set(ParamsKey, bz)
}

func (k Keeper) GetMissedExtensions(ctx sdk.Context, valAddr sdk.ValAddress) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(append(MissedExtensionsKey, valAddr.Bytes()...))
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

func (k Keeper) IncrementMissedExtensions(ctx sdk.Context, valAddr sdk.ValAddress) int64 {
	missed := k.GetMissedExtensions(ctx, valAddr) + 1
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(missed))
	store.Set(append(MissedExtensionsKey, valAddr.Bytes()...), bz)
	return missed
}

func (k Keeper) ResetMissedExtensions(ctx sdk.Context, valAddr sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(append(MissedExtensionsKey, valAddr.Bytes()...))
}

// SetValidatorAttested marks a validator's hardware attestation status.
func (k Keeper) SetValidatorAttested(ctx sdk.Context, valAddr sdk.ValAddress, attested bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(AttestationKeyPrefix, valAddr.Bytes()...)
	if attested {
		store.Set(key, []byte{0x01})
	} else {
		store.Set(key, []byte{0x00})
	}
}

// IsValidatorAttested checks if a validator is currently attested.
func (k Keeper) IsValidatorAttested(ctx sdk.Context, valAddr sdk.ValAddress) bool {
	store := ctx.KVStore(k.storeKey)
	key := append(AttestationKeyPrefix, valAddr.Bytes()...)
	bz := store.Get(key)
	if bz == nil {
		return true // Default to true if not set
	}
	return bz[0] == 0x01
}

// GetAllAttestedValidators returns all validators with explicit attestation status.
func (k Keeper) GetAllAttestedValidators(ctx sdk.Context) []string {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, AttestationKeyPrefix)
	defer iterator.Close()

	var validators []string
	for ; iterator.Valid(); iterator.Next() {
		if iterator.Value()[0] == 0x01 {
			valAddr := sdk.ValAddress(iterator.Key()[len(AttestationKeyPrefix):])
			validators = append(validators, valAddr.String())
		}
	}
	return validators
}

// Sliding window helpers for liveness window (10,000 blocks)
func (k Keeper) GetValidatorSignedCount(ctx sdk.Context, valAddr sdk.ValAddress) int64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(append(SignedCountPrefix, valAddr.Bytes()...))
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

func (k Keeper) SetValidatorSignedCount(ctx sdk.Context, valAddr sdk.ValAddress, count int64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(count))
	store.Set(append(SignedCountPrefix, valAddr.Bytes()...), bz)
}

func (k Keeper) SetValidatorSignedBit(ctx sdk.Context, valAddr sdk.ValAddress, height int64, signed bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(append(SignedBitPrefix, valAddr.Bytes()...), sdk.Uint64ToBigEndian(uint64(height))...)
	if signed {
		store.Set(key, []byte{0x01})
	} else {
		store.Set(key, []byte{0x00})
	}
}

func (k Keeper) GetValidatorSignedBit(ctx sdk.Context, valAddr sdk.ValAddress, height int64) bool {
	store := ctx.KVStore(k.storeKey)
	key := append(append(SignedBitPrefix, valAddr.Bytes()...), sdk.Uint64ToBigEndian(uint64(height))...)
	bz := store.Get(key)
	return bz != nil && bz[0] == 0x01
}

func (k Keeper) ClearValidatorSignedBit(ctx sdk.Context, valAddr sdk.ValAddress, height int64) {
	store := ctx.KVStore(k.storeKey)
	key := append(append(SignedBitPrefix, valAddr.Bytes()...), sdk.Uint64ToBigEndian(uint64(height))...)
	store.Delete(key)
}

func (k Keeper) GetRequiredSigningThreshold(ctx sdk.Context, height int64) int64 {
	W := int64(10000)
	var K int64 = 5000 // 50%
	if k.IsDegradedMode(ctx) {
		K = 3000 // 30%
	}

	if height < 100 {
		return 0
	}
	if height < W {
		return (height * K) / W
	}
	return K
}

// EndBlocker processes consecutive block proposal rejections and updates degraded mode.
func (k Keeper) EndBlocker(ctx sdk.Context, lastProposalRejected bool) {
	// 1. Process consecutive proposal rejections
	if lastProposalRejected {
		count := k.GetConsecutiveRejectionCount(ctx) + 1
		k.SetConsecutiveRejectionCount(ctx, count)

		params := k.GetParams(ctx)
		limit := params.MaxConsecutiveRejections
		if count >= limit {
			k.SetDegradedMode(ctx, true)
			ctx.EventManager().EmitEvent(sdk.NewEvent("degraded_mode_activated"))
		}
	} else {
		// Reset rejections on success block commit
		k.SetConsecutiveRejectionCount(ctx, 0)
		if k.IsDegradedMode(ctx) {
			k.SetDegradedMode(ctx, false)
			ctx.EventManager().EmitEvent(sdk.NewEvent("degraded_mode_deactivated"))
		}
	}

	// Telemetry rejection count
	telemetry.SetGauge(float32(k.GetConsecutiveRejectionCount(ctx)), "rejection_count")

	// Telemetry degraded mode active status
	var degradedVal float32
	if k.IsDegradedMode(ctx) {
		degradedVal = 1.0
	}
	telemetry.SetGauge(degradedVal, "degraded_mode_active")

	// 2. Track rolling liveness window and bootstrapping
	height := ctx.BlockHeight()
	W := int64(10000)

	// Map validator consensus addresses to signed bit from block header VoteInfos
	signingMap := make(map[string]bool)
	for _, vote := range ctx.VoteInfos() {
		signingMap[string(vote.Validator.Address)] = (vote.BlockIdFlag == tmtypes.BlockIDFlagCommit)
	}

	var totalValidators int32
	var attestedValidators int32

	// Iterate all last validators to update their rolling window
	if k.stakingKeeper != nil {
		_ = k.stakingKeeper.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
			totalValidators++
			if k.IsValidatorAttested(ctx, valAddr) {
				attestedValidators++
			}

			val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
			if err != nil {
				return false
			}
			consAddr, err := val.GetConsAddr()
			if err != nil {
				return false
			}

			// Check if they signed
			signed := signingMap[string(consAddr)]
			currentCount := k.GetValidatorSignedCount(ctx, valAddr)

			if signed {
				currentCount++
				k.SetValidatorSignedBit(ctx, valAddr, height, true)
			} else {
				k.SetValidatorSignedBit(ctx, valAddr, height, false)
			}

			// Decrement sliding window bit
			if height > W {
				oldHeight := height - W
				if k.GetValidatorSignedBit(ctx, valAddr, oldHeight) {
					currentCount--
				}
				k.ClearValidatorSignedBit(ctx, valAddr, oldHeight)
			}

			k.SetValidatorSignedCount(ctx, valAddr, currentCount)

			// Liveness / Bootstrapping Check
			threshold := k.GetRequiredSigningThreshold(ctx, height)
			if currentCount < threshold {
				telemetry.IncrCounter(1.0, "bound_violations")
				if k.slashingKeeper != nil {
					_ = k.slashingKeeper.Jail(ctx, consAddr)
				}
			}

			// 3. Attestation check
			if !k.IsValidatorAttested(ctx, valAddr) {
				if k.slashingKeeper != nil {
					_ = k.slashingKeeper.Jail(ctx, consAddr)
				}
			}

			return false
		})
	}

	// Telemetry attestation coverage
	if totalValidators > 0 {
		coverage := float32(attestedValidators) / float32(totalValidators)
		telemetry.SetGauge(coverage, "attestation_coverage")
	} else {
		telemetry.SetGauge(0.0, "attestation_coverage")
	}
}

// HandleMissedExtension handles missed extension tracking and slashing.
func (k Keeper) HandleMissedExtension(ctx sdk.Context, valAddr sdk.ValAddress) {
	missed := k.IncrementMissedExtensions(ctx, valAddr)
	params := k.GetParams(ctx)

	if missed >= params.MissedExtensionLimit {
		val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
		if err == nil {
			consAddr, err := val.GetConsAddr()
			if err == nil && k.slashingKeeper != nil {
				fraction := math.LegacyNewDecWithPrec(1, 2) // 1% slashing penalty
				_ = k.slashingKeeper.Slash(ctx, consAddr, fraction, val.GetConsensusPower(sdk.DefaultPowerReduction), ctx.BlockHeight())
				_ = k.slashingKeeper.Jail(ctx, consAddr)
			}
		}
		k.ResetMissedExtensions(ctx, valAddr)
	}
}

// CheckProcessProposalThreshold returns the attestation finality threshold ratio (strict or relaxed)
func (k Keeper) CheckProcessProposalThreshold(ctx sdk.Context) float64 {
	if k.IsDegradedMode(ctx) {
		return 0.51 // Relaxed threshold in degraded mode
	}
	return 0.67 // Strict finality threshold in normal mode
}

func (k Keeper) RegisterInvariants(ir sdk.InvariantRegistry) {
	ir.RegisterRoute(ModuleName, "window-consistency", k.WindowConsistencyInvariant)
}

func (k Keeper) WindowConsistencyInvariant(ctx sdk.Context) (string, bool) {
	if k.stakingKeeper == nil {
		return "staking keeper is nil", false
	}
	height := ctx.BlockHeight()
	W := int64(10000)
	startHeight := int64(1)
	if height > W {
		startHeight = height - W + 1
	}

	var breachMsg string
	breached := false

	_ = k.stakingKeeper.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
		storedCount := k.GetValidatorSignedCount(ctx, valAddr)
		
		// Recalculate actual signed bits in the sliding window
		var actualCount int64
		for h := startHeight; h <= height; h++ {
			if k.GetValidatorSignedBit(ctx, valAddr, h) {
				actualCount++
			}
		}
		
		if storedCount != actualCount {
			breached = true
			breachMsg = fmt.Sprintf("validator %s signed count mismatch: stored=%d, calculated=%d on height range [%d, %d]",
				valAddr.String(), storedCount, actualCount, startHeight, height)
			return true // stop iteration
		}
		return false
	})

	if breached {
		return breachMsg, true
	}
	return "certification window consistency invariant holds", false
}
