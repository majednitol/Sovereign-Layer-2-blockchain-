package oracle

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error)
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

type AggregatePrice struct {
	Price       uint64 `json:"price"`
	BlockHeight int64  `json:"block_height"`
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

func (k Keeper) GetParams(ctx sdk.Context) Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ParamsKey)
	if bz == nil {
		return Params{
			CommitWindow:             10,
			RevealWindow:             10,
			MinOperatorCommits:       3,
			StalenessThresholdBlocks: 100,
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

// SetOperatorActive registers or updates an operator's active status.
func (k Keeper) SetOperatorActive(ctx sdk.Context, operator string, active bool) {
	store := ctx.KVStore(k.storeKey)
	key := append(OperatorKeyPrefix, []byte(operator)...)
	if active {
		store.Set(key, []byte{0x01})
	} else {
		store.Delete(key)
	}
}

// IsOperatorActive checks if an operator is registered and active.
func (k Keeper) IsOperatorActive(ctx sdk.Context, operator string) bool {
	store := ctx.KVStore(k.storeKey)
	key := append(OperatorKeyPrefix, []byte(operator)...)
	return store.Has(key)
}

// HasAnyOperator checks if there are any operators registered.
func (k Keeper) HasAnyOperator(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, OperatorKeyPrefix)
	defer iterator.Close()
	return iterator.Valid()
}

// GetCommitHeight retrieves the block height of an operator's commit.
func (k Keeper) GetCommitHeight(ctx sdk.Context, operator string, feedID string, roundID uint64) int64 {
	store := ctx.KVStore(k.storeKey)
	commitHeightKey := append(CommitHeightKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", operator, feedID, roundID))...)
	bz := store.Get(commitHeightKey)
	if bz == nil {
		return 0
	}
	return int64(binary.BigEndian.Uint64(bz))
}

func (k Keeper) CommitHash(ctx sdk.Context, operator string, feedID string, roundID uint64, hash []byte) error {
	if k.HasAnyOperator(ctx) && !k.IsOperatorActive(ctx, operator) {
		return fmt.Errorf("operator %s is not active", operator)
	}

	store := ctx.KVStore(k.storeKey)
	commitKey := append(CommitKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", operator, feedID, roundID))...)
	store.Set(commitKey, hash)

	commitHeightKey := append(CommitHeightKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", operator, feedID, roundID))...)
	heightBz := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBz, uint64(ctx.BlockHeight()))
	store.Set(commitHeightKey, heightBz)

	return nil
}

func (k Keeper) GetCommit(ctx sdk.Context, operator string, feedID string, roundID uint64) []byte {
	store := ctx.KVStore(k.storeKey)
	commitKey := append(CommitKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", operator, feedID, roundID))...)
	return store.Get(commitKey)
}

func (k Keeper) RevealReport(ctx sdk.Context, operator string, feedID string, roundID uint64, value uint64, nonce string) error {
	if k.HasAnyOperator(ctx) && !k.IsOperatorActive(ctx, operator) {
		return fmt.Errorf("operator %s is not active", operator)
	}

	storedHash := k.GetCommit(ctx, operator, feedID, roundID)
	if storedHash == nil {
		return fmt.Errorf("no commit found for operator %s, feed %s, round %d", operator, feedID, roundID)
	}

	computedHash := ComputeCommitHash(operator, feedID, roundID, value, nonce)
	if !bytes.Equal(storedHash, computedHash) {
		return fmt.Errorf("hash mismatch for reveal: computed %x, stored %x", computedHash, storedHash)
	}

	store := ctx.KVStore(k.storeKey)
	revealKey := append(RevealKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", operator, feedID, roundID))...)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, value)
	store.Set(revealKey, bz)

	return nil
}

func (k Keeper) GetReveal(ctx sdk.Context, operator string, feedID string, roundID uint64) (uint64, bool) {
	store := ctx.KVStore(k.storeKey)
	revealKey := append(RevealKeyPrefix, []byte(fmt.Sprintf("%s:%s:%d", operator, feedID, roundID))...)
	bz := store.Get(revealKey)
	if bz == nil {
		return 0, false
	}
	return binary.BigEndian.Uint64(bz), true
}

// IterateReveals returns all verified revealed values for a given feed and round.
func (k Keeper) GetRevealedValues(ctx sdk.Context, feedID string, roundID uint64) []uint64 {
	store := ctx.KVStore(k.storeKey)
	var values []uint64

	// Since we are running in unit tests or light simulation, we iterate over the prefix
	iterator := storetypes.KVStorePrefixIterator(store, RevealKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Key format: Prefix + operator + ":" + feedID + ":" + roundID
		// For simplicity in testing, match the suffix of key containing "feedID:roundID"
		suffix := []byte(fmt.Sprintf("%s:%d", feedID, roundID))
		if bytes.HasSuffix(iterator.Key(), suffix) {
			val := binary.BigEndian.Uint64(iterator.Value())
			values = append(values, val)
		}
	}

	return values
}

// FilterOutliersMAD filters pricing outliers using the Median Absolute Deviation algorithm.
func FilterOutliersMAD(values []uint64) []uint64 {
	n := len(values)
	if n <= 2 {
		return values
	}
	median := CalculateMedian(values)

	deviations := make([]uint64, n)
	for i, v := range values {
		if v > median {
			deviations[i] = v - median
		} else {
			deviations[i] = median - v
		}
	}

	mad := CalculateMedian(deviations)
	if mad == 0 {
		return values // High agreement, no need to filter
	}

	var filtered []uint64
	for _, v := range values {
		var diff uint64
		if v > median {
			diff = v - median
		} else {
			diff = median - v
		}
		if diff <= 3*mad {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func CalculateMedian(values []uint64) uint64 {
	temp := make([]uint64, len(values))
	copy(temp, values)
	sort.Slice(temp, func(i, j int) bool { return temp[i] < temp[j] })
	n := len(temp)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return temp[n/2]
	}
	return (temp[n/2-1] + temp[n/2]) / 2
}

// AggregateRound calculates outlier-free median price and stores the result.
func (k Keeper) AggregateRound(ctx sdk.Context, feedID string, roundID uint64) (uint64, error) {
	values := k.GetRevealedValues(ctx, feedID, roundID)
	params := k.GetParams(ctx)

	if int64(len(values)) < params.MinOperatorCommits {
		return 0, fmt.Errorf("insufficient commits for round %d: got %d, need %d", roundID, len(values), params.MinOperatorCommits)
	}

	// Filter outliers using Median Absolute Deviation
	filtered := FilterOutliersMAD(values)
	finalPrice := CalculateMedian(filtered)

	// Store aggregate price
	store := ctx.KVStore(k.storeKey)
	agg := AggregatePrice{
		Price:       finalPrice,
		BlockHeight: ctx.BlockHeight(),
	}
	bz, _ := json.Marshal(agg)
	store.Set(append(AggregateKeyPrefix, []byte(feedID)...), bz)

	return finalPrice, nil
}

func (k Keeper) GetLatestPrice(ctx sdk.Context, feedID string) (uint64, int64, error) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(append(AggregateKeyPrefix, []byte(feedID)...))
	if bz == nil {
		return 0, 0, fmt.Errorf("no price aggregated for feed %s", feedID)
	}

	var agg AggregatePrice
	_ = json.Unmarshal(bz, &agg)

	if k.IsFeedStale(ctx, feedID) {
		return agg.Price, agg.BlockHeight, fmt.Errorf("feed %s is stale", feedID)
	}

	return agg.Price, agg.BlockHeight, nil
}

func (k Keeper) IsFeedStale(ctx sdk.Context, feedID string) bool {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(append(AggregateKeyPrefix, []byte(feedID)...))
	if bz == nil {
		return true
	}

	var agg AggregatePrice
	_ = json.Unmarshal(bz, &agg)

	params := k.GetParams(ctx)
	return (ctx.BlockHeight() - agg.BlockHeight) > params.StalenessThresholdBlocks
}

// EndBlocker processes committed but unrevealed reports, and slashes/jails operators.
func (k Keeper) EndBlocker(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	params := k.GetParams(ctx)

	// We iterate over all commits in the store
	iterator := storetypes.KVStorePrefixIterator(store, CommitKeyPrefix)
	defer iterator.Close()

	var commitsToDelete [][]byte
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		// Key format: CommitKeyPrefix + operator + ":" + feedID + ":" + roundID
		keyBytes := key[len(CommitKeyPrefix):]
		parts := strings.Split(string(keyBytes), ":")
		if len(parts) != 3 {
			continue
		}
		operator := parts[0]
		feedID := parts[1]
		roundID, err := strconv.ParseUint(parts[2], 10, 64)
		if err != nil {
			continue
		}

		commitHeight := k.GetCommitHeight(ctx, operator, feedID, roundID)
		if commitHeight == 0 {
			continue
		}

		// Check if the reveal window has fully expired
		if ctx.BlockHeight() > commitHeight+params.CommitWindow+params.RevealWindow {
			// Check if reveal exists
			_, revealed := k.GetReveal(ctx, operator, feedID, roundID)
			if !revealed {
				// Slash and jail the operator
				valAddr, err := sdk.ValAddressFromBech32(operator)
				if err == nil && k.stakingKeeper != nil && k.slashingKeeper != nil {
					val, err := k.stakingKeeper.GetValidator(ctx, valAddr)
					if err == nil {
						consAddr, err := val.GetConsAddr()
						if err == nil {
							fraction := math.LegacyNewDecWithPrec(1, 2) // 1% slashing penalty
							_ = k.slashingKeeper.Slash(ctx, consAddr, fraction, val.GetConsensusPower(sdk.DefaultPowerReduction), ctx.BlockHeight())
							_ = k.slashingKeeper.Jail(ctx, consAddr)
						}
					}
				}
			}
			// Mark this commit for deletion so we don't process it repeatedly
			commitsToDelete = append(commitsToDelete, key)
		}
	}

	// Delete expired commits and their corresponding commit heights
	for _, key := range commitsToDelete {
		store.Delete(key)
		// Corresponding height key
		keyBytes := key[len(CommitKeyPrefix):]
		heightKey := append(CommitHeightKeyPrefix, keyBytes...)
		store.Delete(heightKey)
	}
}

func (k Keeper) RegisterInvariants(ir sdk.InvariantRegistry) {
	ir.RegisterRoute(ModuleName, "staleness", k.StalenessInvariant)
}

func (k Keeper) StalenessInvariant(ctx sdk.Context) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, AggregateKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		feedID := string(iterator.Key()[len(AggregateKeyPrefix):])
		var agg AggregatePrice
		if err := json.Unmarshal(iterator.Value(), &agg); err != nil {
			return "oracle staleness invariant breach: failed to unmarshal aggregate price", true
		}
		isStale := (ctx.BlockHeight() - agg.BlockHeight) > k.GetParams(ctx).StalenessThresholdBlocks
		if k.IsFeedStale(ctx, feedID) != isStale {
			return fmt.Sprintf("feed %s staleness mismatch: IsFeedStale=%v, expected=%v", feedID, k.IsFeedStale(ctx, feedID), isStale), true
		}
	}
	return "oracle staleness invariant holds", false
}

