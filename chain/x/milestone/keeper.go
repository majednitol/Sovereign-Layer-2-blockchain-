package milestone

import (
	"context"
	"encoding/json"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type OracleKeeper interface {
	IsFeedStale(ctx sdk.Context, feedID string) bool
	GetLatestPrice(ctx sdk.Context, feedID string) (uint64, int64, error)
}

type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
}

type Keeper struct {
	storeKey     storetypes.StoreKey
	cdc          codec.BinaryCodec
	oracleKeeper OracleKeeper
	bankKeeper   BankKeeper
}

func NewKeeper(
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
	oracleKeeper OracleKeeper,
	bankKeeper BankKeeper,
) Keeper {
	return Keeper{
		storeKey:     storeKey,
		cdc:          cdc,
		oracleKeeper: oracleKeeper,
		bankKeeper:   bankKeeper,
	}
}

func (k Keeper) GetParams(ctx sdk.Context) Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ParamsKey)
	if bz == nil {
		return Params{
			MaxActiveMilestones: 500,
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

func (k Keeper) AddActiveFeed(ctx sdk.Context, feedID string) {
	store := ctx.KVStore(k.storeKey)
	key := append(ActiveFeedsKeyPrefix, []byte(feedID)...)
	store.Set(key, []byte{0x01})
}

func (k Keeper) RemoveActiveFeed(ctx sdk.Context, feedID string) {
	store := ctx.KVStore(k.storeKey)
	key := append(ActiveFeedsKeyPrefix, []byte(feedID)...)
	store.Delete(key)
}

func (k Keeper) IterateActiveFeeds(ctx sdk.Context, handler func(feedID string) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, ActiveFeedsKeyPrefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		feedID := string(iterator.Key()[len(ActiveFeedsKeyPrefix):])
		if handler(feedID) {
			break
		}
	}
}

func (k Keeper) AddMilestoneToFeedIndex(ctx sdk.Context, feedID, milestoneID string) {
	store := ctx.KVStore(k.storeKey)
	key := append(append(FeedMilestoneIndexKeyPrefix, []byte(feedID)...), append([]byte(":"), []byte(milestoneID)...)...)
	store.Set(key, []byte{0x01})
}

func (k Keeper) RemoveMilestoneFromFeedIndex(ctx sdk.Context, feedID, milestoneID string) {
	store := ctx.KVStore(k.storeKey)
	key := append(append(FeedMilestoneIndexKeyPrefix, []byte(feedID)...), append([]byte(":"), []byte(milestoneID)...)...)
	store.Delete(key)
}

func (k Keeper) IterateMilestonesByFeed(ctx sdk.Context, feedID string, handler func(milestoneID string) bool) {
	store := ctx.KVStore(k.storeKey)
	prefix := append(append(FeedMilestoneIndexKeyPrefix, []byte(feedID)...), []byte(":")...)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		key := iterator.Key()
		milestoneID := string(key[len(prefix):])
		if handler(milestoneID) {
			break
		}
	}
}

func (k Keeper) SetMilestone(ctx sdk.Context, m Milestone) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(m)
	store.Set(append(MilestoneKeyPrefix, []byte(m.ID)...), bz)

	if m.State == StatePending || m.State == StateStaleBlocked {
		k.AddActiveFeed(ctx, m.FeedID)
		k.AddMilestoneToFeedIndex(ctx, m.FeedID, m.ID)
	} else {
		k.RemoveMilestoneFromFeedIndex(ctx, m.FeedID, m.ID)
		// Check if any other milestones remain for this feed
		hasRemaining := false
		k.IterateMilestonesByFeed(ctx, m.FeedID, func(milestoneID string) bool {
			hasRemaining = true
			return true // stop iteration
		})
		if !hasRemaining {
			k.RemoveActiveFeed(ctx, m.FeedID)
		}
	}
}

func (k Keeper) GetMilestone(ctx sdk.Context, id string) (Milestone, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(append(MilestoneKeyPrefix, []byte(id)...))
	if bz == nil {
		return Milestone{}, false
	}
	var m Milestone
	_ = json.Unmarshal(bz, &m)
	return m, true
}

func (k Keeper) IterateMilestones(ctx sdk.Context, handler func(m Milestone) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, MilestoneKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var m Milestone
		_ = json.Unmarshal(iterator.Value(), &m)
		if handler(m) {
			break
		}
	}
}

// IsFeedStaleBlocked checks if a feed is currently marked as stale blocked in the milestone module.
func (k Keeper) IsFeedStaleBlocked(ctx sdk.Context, feedID string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(append([]byte("feed_stale_blocked:"), []byte(feedID)...))
}

// SetFeedStaleBlocked sets the stale blocked status of a feed.
func (k Keeper) SetFeedStaleBlocked(ctx sdk.Context, feedID string, blocked bool) {
	store := ctx.KVStore(k.storeKey)
	key := append([]byte("feed_stale_blocked:"), []byte(feedID)...)
	if blocked {
		store.Set(key, []byte{0x01})
	} else {
		store.Delete(key)
	}
}

// EndBlocker processes the milestone deadline counts and updates vesting conditions.
func (k Keeper) EndBlocker(ctx sdk.Context) {
	params := k.GetParams(ctx)
	var processedCount int64

	k.IterateActiveFeeds(ctx, func(feedID string) bool {
		if processedCount >= params.MaxActiveMilestones {
			return true // Stop iterating feeds
		}

		stale := k.oracleKeeper.IsFeedStale(ctx, feedID)
		isBlocked := k.IsFeedStaleBlocked(ctx, feedID)

		// O(1) skip for already stale-blocked feeds when they remain stale
		if stale && isBlocked {
			return false // Skip this feed entirely in O(1)
		}

		// Collect all milestone IDs for this feed first to avoid iterator conflicts during modifications
		var milestoneIDs []string
		k.IterateMilestonesByFeed(ctx, feedID, func(mID string) bool {
			milestoneIDs = append(milestoneIDs, mID)
			return false
		})

		for _, mID := range milestoneIDs {
			if processedCount >= params.MaxActiveMilestones {
				break
			}

			m, ok := k.GetMilestone(ctx, mID)
			if !ok {
				continue
			}

			if m.State == StateAchieved || m.State == StateExpired {
				continue
			}

			processedCount++

			if stale {
				if m.State == StatePending {
					m.State = StateStaleBlocked
					k.SetMilestone(ctx, m)
					ctx.EventManager().EmitEvent(sdk.NewEvent(
						"milestone_stale_blocked",
						sdk.NewAttribute("milestone_id", m.ID),
					))
				}
			} else {
				price, _, err := k.oracleKeeper.GetLatestPrice(ctx, feedID)
				if err == nil {
					if m.State == StateStaleBlocked {
						if price >= m.TargetPrice {
							m.State = StateAchieved
							k.SetMilestone(ctx, m)
							k.triggerVestingPayout(ctx, m)
						} else {
							m.State = StatePending
							k.SetMilestone(ctx, m)
						}
					} else if m.State == StatePending {
						if price >= m.TargetPrice {
							m.State = StateAchieved
							k.SetMilestone(ctx, m)
							k.triggerVestingPayout(ctx, m)
						} else {
							m.RemainingBlocks--
							if m.RemainingBlocks <= 0 {
								m.State = StateExpired
								k.SetMilestone(ctx, m)
								ctx.EventManager().EmitEvent(sdk.NewEvent(
									"milestone_expired",
									sdk.NewAttribute("milestone_id", m.ID),
								))
							} else {
								k.SetMilestone(ctx, m)
							}
						}
					}
				}
			}
		}

		// Update stale blocked flag for the feed
		if stale {
			k.SetFeedStaleBlocked(ctx, feedID, true)
		} else {
			k.SetFeedStaleBlocked(ctx, feedID, false)
		}

		return false // Continue iterating other feeds
	})
}

func (k Keeper) triggerVestingPayout(ctx sdk.Context, m Milestone) {
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"milestone_achieved",
		sdk.NewAttribute("milestone_id", m.ID),
	))

	poolAddr, err := sdk.AccAddressFromBech32(m.VestingPoolAddress)
	if err == nil && k.bankKeeper != nil {
		amount := sdk.NewCoins(sdk.NewCoin("ucsov", math.NewInt(10000000)))
		_ = k.bankKeeper.SendCoins(ctx, sdk.AccAddress([]byte("milestone_escrow")), poolAddr, amount)
	}
}
