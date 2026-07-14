package validator

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	GetLastValidatorPower(ctx context.Context, valAddr sdk.ValAddress) (int64, error)
	GetLastTotalPower(ctx context.Context) (math.Int, error)
	IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) (stop bool)) error
	GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error)
}

type SlashingKeeper interface {
	Tombstone(ctx context.Context, valAddr sdk.ConsAddress) error
	HasValidatorSigningInfo(ctx context.Context, consAddr sdk.ConsAddress) bool
	SetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) error
}

type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type DistrKeeper interface {
	GetValidatorOutstandingRewards(ctx context.Context, valAddr sdk.ValAddress) (distrtypes.ValidatorOutstandingRewards, error)
}

type Keeper struct {
	storeKey       storetypes.StoreKey
	cdc            codec.BinaryCodec
	stakingKeeper  StakingKeeper
	slashingKeeper SlashingKeeper
	bankKeeper     BankKeeper
	distrKeeper    DistrKeeper
	MaxValidators  uint32
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	stakingKeeper StakingKeeper,
	slashingKeeper SlashingKeeper,
	bankKeeper BankKeeper,
	distrKeeper DistrKeeper,
	maxValidators uint32,
) Keeper {
	return Keeper{
		storeKey:       storeKey,
		cdc:            cdc,
		stakingKeeper:  stakingKeeper,
		slashingKeeper: slashingKeeper,
		bankKeeper:     bankKeeper,
		distrKeeper:    distrKeeper,
		MaxValidators:  maxValidators,
	}
}

// GetMaxValidators gets the max validators limit from store, falling back to k.MaxValidators if not set.
func (k Keeper) GetMaxValidators(ctx sdk.Context) uint32 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(MaxValidatorsKeyPrefix)
	if bz == nil {
		return k.MaxValidators
	}
	if len(bz) < 4 {
		return k.MaxValidators
	}
	return binary.BigEndian.Uint32(bz)
}

// SetMaxValidators sets the max validators limit in store.
func (k Keeper) SetMaxValidators(ctx sdk.Context, max uint32) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 4)
	binary.BigEndian.PutUint32(bz, max)
	store.Set(MaxValidatorsKeyPrefix, bz)
}

// GetPartitionScheme gets the partition scheme from store.
func (k Keeper) GetPartitionScheme(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(PartitionSchemeKeyPrefix)
	if bz == nil {
		return "equal-slots-30" // default fallback
	}
	return string(bz)
}

// SetPartitionScheme sets the partition scheme in store.
func (k Keeper) SetPartitionScheme(ctx sdk.Context, scheme string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(PartitionSchemeKeyPrefix, []byte(scheme))
}

// IsValidatorActive checks if a validator is in the active top MaxValidators slots.
func (k Keeper) IsValidatorActive(ctx sdk.Context, valAddr sdk.ValAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(append(SlotKeyPrefix, valAddr.Bytes()...))
}

// SetValidatorActive marks a validator as active.
func (k Keeper) SetValidatorActive(ctx sdk.Context, valAddr sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(append(SlotKeyPrefix, valAddr.Bytes()...), []byte{0x01})
}

// RemoveValidatorActive removes a validator from the active set.
func (k Keeper) RemoveValidatorActive(ctx sdk.Context, valAddr sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(append(SlotKeyPrefix, valAddr.Bytes()...))
}

// QueueEjection stores an ejection flag to be processed after the unbonding period.
func (k Keeper) QueueEjection(ctx sdk.Context, valAddr sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(append(QueuedEjectionKeyPrefix, valAddr.Bytes()...), []byte{0x01})
}

// IsEjectionQueued checks if a validator has a queued ejection.
func (k Keeper) IsEjectionQueued(ctx sdk.Context, valAddr sdk.ValAddress) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(append(QueuedEjectionKeyPrefix, valAddr.Bytes()...))
}

// EndBlocker overrides x/staking's power updates and returns equalized powers.
func (k Keeper) EndBlocker(ctx sdk.Context) []abci.ValidatorUpdate {
	var updates []abci.ValidatorUpdate

	// 1. Gather all validators from x/staking
	var activeCount uint32
	maxVals := k.GetMaxValidators(ctx)
	_ = k.stakingKeeper.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
		if activeCount < maxVals {
			// Validator qualifies for active slot
			if !k.IsValidatorActive(ctx, valAddr) {
				k.SetValidatorActive(ctx, valAddr)
				val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
				if err == nil {
					consAddr, err := val.GetConsAddr()
					if err == nil {
						if !k.slashingKeeper.HasValidatorSigningInfo(ctx, consAddr) {
							info := slashingtypes.NewValidatorSigningInfo(
								consAddr,
								ctx.BlockHeight(),
								0,
								time.Unix(0, 0),
								false,
								0,
							)
							_ = k.slashingKeeper.SetValidatorSigningInfo(ctx, consAddr, info)
						}
					}
				}
			}
			// Retrieve public key from staking state
			var pk crypto.PublicKey
			val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
			if err == nil {
				consPk, err := val.ConsPubKey()
				if err == nil {
					pk = crypto.PublicKey{
						Sum: &crypto.PublicKey_Ed25519{
							Ed25519: consPk.Bytes(),
						},
					}
				}
			}
			// Equalized voting power mapping rule: 1,000,000 for active slot
			updates = append(updates, abci.ValidatorUpdate{
				PubKey: pk,
				Power:  1000000,
			})
			activeCount++
		} else {
			// Exceeds max validator slots - validate ejection / unbonding status
			if k.IsValidatorActive(ctx, valAddr) {
				// Queue ejection to avoid hooks panics during 21-day unbonding
				k.QueueEjection(ctx, valAddr)
				k.RemoveValidatorActive(ctx, valAddr)
				val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
				if err == nil {
					consAddr, err := val.GetConsAddr()
					if err == nil {
						_ = k.slashingKeeper.Tombstone(ctx, consAddr)
					}
				}
				// Tombstone/inactive updates
				updates = append(updates, abci.ValidatorUpdate{
					Power: 0,
				})
			}
		}
		return false
	})

	return updates
}

func (k Keeper) RegisterInvariants(ir sdk.InvariantRegistry) {
	ir.RegisterRoute(ModuleName, "rewards-bucket", k.RewardsBucketInvariant)
}

func (k Keeper) RewardsBucketInvariant(ctx sdk.Context) (string, bool) {
	if k.bankKeeper == nil || k.distrKeeper == nil {
		return "bank or distribution keeper is nil", false
	}
	distrAddr := authtypes.NewModuleAddress(distrtypes.ModuleName)
	distrBalance := k.bankKeeper.GetBalance(ctx, distrAddr, "ucsov")

	totalRewards := sdk.NewDecCoins()
	_ = k.stakingKeeper.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
		rewards, err := k.distrKeeper.GetValidatorOutstandingRewards(ctx, valAddr)
		if err == nil {
			totalRewards = totalRewards.Add(rewards.Rewards...)
		}
		return false
	})

	totalRewardsCoins, _ := totalRewards.TruncateDecimal()
	ucsovRewards := totalRewardsCoins.AmountOf("ucsov")
	if ucsovRewards.GT(distrBalance.Amount) {
		return fmt.Sprintf("validator rewards bucket invariant breach: outstanding rewards %s exceed distribution module balance %s", ucsovRewards, distrBalance.Amount), true
	}
	return "validator rewards bucket invariant holds", false
}
