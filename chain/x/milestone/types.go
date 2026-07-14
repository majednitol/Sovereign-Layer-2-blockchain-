package milestone

import (
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
	return err
}

func (msg *MsgCreateMilestone) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
