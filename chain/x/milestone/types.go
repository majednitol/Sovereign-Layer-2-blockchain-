package milestone

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*MsgCreateMilestone)(nil), "sovereign.milestone.v1.MsgCreateMilestone")
}

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
	Creator            string `protobuf:"bytes,1,opt,name=creator,proto3" json:"creator"`
	ID                 string `protobuf:"bytes,2,opt,name=id,proto3" json:"id"`
	FeedID             string `protobuf:"bytes,3,opt,name=feed_id,json=feedId,proto3" json:"feed_id"`
	TargetPrice        uint64 `protobuf:"varint,4,opt,name=target_price,json=targetPrice,proto3" json:"target_price"`
	DurationBlocks     int64  `protobuf:"varint,5,opt,name=duration_blocks,json=durationBlocks,proto3" json:"duration_blocks"`
	VestingPoolAddress string `protobuf:"bytes,6,opt,name=vesting_pool_address,json=vestingPoolAddress,proto3" json:"vesting_pool_address"`
	PayoutAmount       uint64 `protobuf:"varint,7,opt,name=payout_amount,json=payoutAmount,proto3" json:"payout_amount"`
}

func (msg *MsgCreateMilestone) Reset()         { *msg = MsgCreateMilestone{} }
func (msg *MsgCreateMilestone) String() string { return msg.ID }
func (msg *MsgCreateMilestone) ProtoMessage()  {}
func (msg *MsgCreateMilestone) Descriptor() ([]byte, []int) {
	return MsgCreateMilestoneDesc, []int{0}
}

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

func encodeVarint(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val|0x80))
		val >>= 7
	}
	buf = append(buf, byte(val))
	return buf
}

func decodeVarint(buf []byte) (uint64, int) {
	var val uint64
	var shift uint
	for i, b := range buf {
		val |= uint64(b&0x7f) << shift
		if b < 0x80 {
			return val, i + 1
		}
		shift += 7
	}
	return 0, 0
}

func (m *MsgCreateMilestone) Marshal() ([]byte, error) {
	var buf []byte
	if len(m.Creator) > 0 {
		buf = append(buf, (1<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.Creator)))...)
		buf = append(buf, []byte(m.Creator)...)
	}
	if len(m.ID) > 0 {
		buf = append(buf, (2<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.ID)))...)
		buf = append(buf, []byte(m.ID)...)
	}
	if len(m.FeedID) > 0 {
		buf = append(buf, (3<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.FeedID)))...)
		buf = append(buf, []byte(m.FeedID)...)
	}
	if m.TargetPrice > 0 {
		buf = append(buf, (4<<3)|0)
		buf = append(buf, encodeVarint(m.TargetPrice)...)
	}
	if m.DurationBlocks > 0 {
		buf = append(buf, (5<<3)|0)
		buf = append(buf, encodeVarint(uint64(m.DurationBlocks))...)
	}
	if len(m.VestingPoolAddress) > 0 {
		buf = append(buf, (6<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.VestingPoolAddress)))...)
		buf = append(buf, []byte(m.VestingPoolAddress)...)
	}
	if m.PayoutAmount > 0 {
		buf = append(buf, (7<<3)|0)
		buf = append(buf, encodeVarint(m.PayoutAmount)...)
	}
	return buf, nil
}

func (m *MsgCreateMilestone) Size() int {
	var size int
	if len(m.Creator) > 0 {
		l := len(m.Creator)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if len(m.ID) > 0 {
		l := len(m.ID)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if len(m.FeedID) > 0 {
		l := len(m.FeedID)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if m.TargetPrice > 0 {
		size += 1 + len(encodeVarint(m.TargetPrice))
	}
	if m.DurationBlocks > 0 {
		size += 1 + len(encodeVarint(uint64(m.DurationBlocks)))
	}
	if len(m.VestingPoolAddress) > 0 {
		l := len(m.VestingPoolAddress)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if m.PayoutAmount > 0 {
		size += 1 + len(encodeVarint(m.PayoutAmount))
	}
	return size
}

func (m *MsgCreateMilestone) Unmarshal(data []byte) error {
	var index int
	for index < len(data) {
		tag, n := decodeVarint(data[index:])
		if n == 0 {
			return fmt.Errorf("invalid tag varint")
		}
		index += n
		wireType := tag & 7
		fieldNum := tag >> 3
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("invalid wire type %d for field 1", wireType)
			}
			length, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid length varint")
			}
			index += n
			m.Creator = string(data[index : index+int(length)])
			index += int(length)
		case 2:
			if wireType != 2 {
				return fmt.Errorf("invalid wire type %d for field 2", wireType)
			}
			length, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid length varint")
			}
			index += n
			m.ID = string(data[index : index+int(length)])
			index += int(length)
		case 3:
			if wireType != 2 {
				return fmt.Errorf("invalid wire type %d for field 3", wireType)
			}
			length, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid length varint")
			}
			index += n
			m.FeedID = string(data[index : index+int(length)])
			index += int(length)
		case 4:
			if wireType != 0 {
				return fmt.Errorf("invalid wire type %d for field 4", wireType)
			}
			val, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid target price varint")
			}
			index += n
			m.TargetPrice = val
		case 5:
			if wireType != 0 {
				return fmt.Errorf("invalid wire type %d for field 5", wireType)
			}
			val, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid duration blocks varint")
			}
			index += n
			m.DurationBlocks = int64(val)
		case 6:
			if wireType != 2 {
				return fmt.Errorf("invalid wire type %d for field 6", wireType)
			}
			length, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid length varint")
			}
			index += n
			m.VestingPoolAddress = string(data[index : index+int(length)])
			index += int(length)
		case 7:
			if wireType != 0 {
				return fmt.Errorf("invalid wire type %d for field 7", wireType)
			}
			val, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid payout amount varint")
			}
			index += n
			m.PayoutAmount = val
		default:
			switch wireType {
			case 0:
				_, n := decodeVarint(data[index:])
				index += n
			case 2:
				length, n := decodeVarint(data[index:])
				index += n + int(length)
			default:
				return fmt.Errorf("unsupported wire type %d", wireType)
			}
		}
	}
	return nil
}

var MsgCreateMilestoneDesc = []byte{
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x34, 0x8f,
	0xb1, 0x4a, 0x03, 0x41, 0x10, 0x86, 0xd9, 0x24, 0xe6, 0xc8, 0x70, 0x72,
	0xb0, 0x44, 0xdc, 0xc2, 0xe2, 0xb0, 0x3a, 0x2c, 0x6c, 0xec, 0x6c, 0xad,
	0xf3, 0x0c, 0xcb, 0xe6, 0x76, 0x3c, 0x16, 0x2f, 0x3b, 0xc7, 0xcc, 0x6c,
	0xc0, 0xd6, 0x27, 0xb2, 0xf0, 0x9d, 0x7c, 0x0d, 0xb9, 0x05, 0xbb, 0x9f,
	0xaf, 0xf8, 0xf8, 0x3f, 0x68, 0xc7, 0x22, 0x4a, 0x97, 0xe7, 0x85, 0x49,
	0xc9, 0x1e, 0x84, 0xae, 0xc8, 0x98, 0xa6, 0xfc, 0xf8, 0x63, 0xc0, 0x9e,
	0x64, 0x7a, 0x63, 0x0c, 0x8a, 0xa7, 0x34, 0xa3, 0x28, 0x65, 0xb4, 0x1d,
	0x34, 0xe3, 0x8a, 0x88, 0x9d, 0xe9, 0xcd, 0x70, 0xb0, 0x00, 0x9b, 0x14,
	0xdd, 0xa6, 0xee, 0x0e, 0x9a, 0x77, 0xc4, 0xe8, 0x53, 0x74, 0xdb, 0x0a,
	0x8e, 0xd0, 0x6a, 0xe0, 0x09, 0xd5, 0x2f, 0x9c, 0x46, 0x74, 0xbb, 0xde,
	0x0c, 0x3b, 0x7b, 0x0f, 0x5d, 0x2c, 0x1c, 0x34, 0x51, 0xf6, 0xe7, 0x99,
	0xc6, 0x0f, 0x71, 0x37, 0xbd, 0x19, 0xb6, 0xf6, 0x01, 0x8e, 0x57, 0x14,
	0x4d, 0x79, 0xf2, 0x0b, 0xd1, 0xec, 0x43, 0x8c, 0x8c, 0x22, 0x6e, 0x5f,
	0x65, 0x77, 0x70, 0xbb, 0x84, 0x4f, 0x2a, 0xea, 0xc3, 0x85, 0x4a, 0x56,
	0xd7, 0xac, 0xb6, 0xd7, 0xf6, 0xeb, 0xf7, 0xfb, 0xe9, 0xff, 0xd4, 0x79,
	0x5f, 0x43, 0x5e, 0xfe, 0x02, 0x00, 0x00, 0xff, 0xff, 0x61, 0x3d, 0x00,
	0x96, 0xd8, 0x00, 0x00, 0x00,
}
