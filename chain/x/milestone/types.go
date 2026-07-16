package milestone

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "milestone"
	StoreKey   = "milestone"
	RouterKey  = "milestone"
)

var (
	MilestoneKeyPrefix          = []byte{0x01}
	ParamsKey                   = []byte{0x02}
	ActiveFeedsKeyPrefix         = []byte{0x03}
	FeedMilestoneIndexKeyPrefix  = []byte{0x04}
	FeedStaleBlockedKeyPrefix    = []byte{0x05}
)

type Params struct {
	MaxActiveMilestones int64 `json:"max_active_milestones"`
}

type Milestone struct {
	ID                 string `json:"id"`
	FeedID             string `json:"feed_id"`
	TargetPrice        uint64 `json:"target_price"`
	RemainingBlocks    int64  `json:"remaining_blocks"`
	State              string `json:"state"` // "pending", "stale-blocked", "achieved", "expired"
	VestingPoolAddress string `json:"vesting_pool_address"`
	PayoutAmount       uint64 `json:"payout_amount"`
}

const (
	StatePending      = "pending"
	StateStaleBlocked = "stale-blocked"
	StateAchieved     = "achieved"
	StateExpired      = "expired"
)

type MsgCreateMilestone struct {
	Creator            string `json:"creator"`
	ID                 string `json:"id"`
	FeedID             string `json:"feed_id"`
	TargetPrice        uint64 `json:"target_price"`
	DurationBlocks     int64  `json:"duration_blocks"`
	VestingPoolAddress string `json:"vesting_pool_address"`
	PayoutAmount       uint64 `json:"payout_amount"`
}

func (msg *MsgCreateMilestone) Reset()         { *msg = MsgCreateMilestone{} }
func (msg *MsgCreateMilestone) String() string { return msg.ID }
func (msg *MsgCreateMilestone) ProtoMessage()  {}

func (msg *MsgCreateMilestone) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}
	if msg.ID == "" {
		return fmt.Errorf("milestone ID cannot be empty")
	}
	if msg.FeedID == "" {
		return fmt.Errorf("feed_id cannot be empty")
	}
	if msg.TargetPrice == 0 {
		return fmt.Errorf("target_price must be positive")
	}
	if msg.DurationBlocks <= 0 {
		return fmt.Errorf("duration_blocks must be positive")
	}
	if msg.VestingPoolAddress == "" {
		return fmt.Errorf("vesting_pool_address cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(msg.VestingPoolAddress); err != nil {
		return fmt.Errorf("invalid vesting_pool_address: %w", err)
	}
	if msg.PayoutAmount == 0 {
		return fmt.Errorf("payout_amount must be positive")
	}
	return nil
}

func (msg *MsgCreateMilestone) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// MilestoneGenesisState defines the milestone module genesis state.
type MilestoneGenesisState struct {
	Params     Params      `json:"params"`
	Milestones []Milestone `json:"milestones"`
}
