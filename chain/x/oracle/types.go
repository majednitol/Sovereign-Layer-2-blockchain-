package oracle

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*MsgCommitOracleHash)(nil), "sovereign.oracle.v1.MsgCommitOracleHash")
	proto.RegisterType((*MsgRevealOracleReport)(nil), "sovereign.oracle.v1.MsgRevealOracleReport")
}

const (
	ModuleName = "oracle"
	StoreKey   = "oracle"
	RouterKey  = "oracle"
)

var (
	CommitKeyPrefix       = []byte{0x01}
	RevealKeyPrefix       = []byte{0x02}
	AggregateKeyPrefix    = []byte{0x03}
	ParamsKey             = []byte{0x04}
	OperatorKeyPrefix     = []byte{0x05}
	CommitHeightKeyPrefix = []byte{0x06}
	ExpiryKeyPrefix       = []byte{0x07}
)

type Params struct {
	CommitWindow             int64 `json:"commit_window"`
	RevealWindow             int64 `json:"reveal_window"`
	MinOperatorCommits       int64 `json:"min_operator_commits"`
	StalenessThresholdBlocks int64 `json:"staleness_threshold_blocks"`
}

type MsgCommitOracleHash struct {
	Operator string `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator"`
	FeedID   string `protobuf:"bytes,2,opt,name=feed_id,json=feedId,proto3" json:"feed_id"`
	RoundID  uint64 `protobuf:"varint,3,opt,name=round_id,json=roundId,proto3" json:"round_id"`
	Hash     []byte `protobuf:"bytes,4,opt,name=hash,proto3" json:"hash"`
}

func (msg *MsgCommitOracleHash) Reset()         { *msg = MsgCommitOracleHash{} }
func (msg *MsgCommitOracleHash) String() string { return msg.Operator }
func (msg *MsgCommitOracleHash) ProtoMessage()  {}
func (msg *MsgCommitOracleHash) Descriptor() ([]byte, []int) {
	return MsgCommitOracleHashDesc, []int{0}
}

func (msg *MsgCommitOracleHash) ValidateBasic() error {
	_, err := sdk.ValAddressFromBech32(msg.Operator)
	return err
}

func (msg *MsgCommitOracleHash) GetSigners() []sdk.AccAddress {
	addr, err := sdk.ValAddressFromBech32(msg.Operator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{sdk.AccAddress(addr)}
}

type MsgRevealOracleReport struct {
	Operator string `protobuf:"bytes,1,opt,name=operator,proto3" json:"operator"`
	FeedID   string `protobuf:"bytes,2,opt,name=feed_id,json=feedId,proto3" json:"feed_id"`
	RoundID  uint64 `protobuf:"varint,3,opt,name=round_id,json=roundId,proto3" json:"round_id"`
	Value    uint64 `protobuf:"varint,4,opt,name=value,proto3" json:"value"`
	Nonce    string `protobuf:"bytes,5,opt,name=nonce,proto3" json:"nonce"`
}

func (msg *MsgRevealOracleReport) Reset()         { *msg = MsgRevealOracleReport{} }
func (msg *MsgRevealOracleReport) String() string { return msg.Operator }
func (msg *MsgRevealOracleReport) ProtoMessage()  {}
func (msg *MsgRevealOracleReport) Descriptor() ([]byte, []int) {
	return MsgRevealOracleReportDesc, []int{0}
}

func (msg *MsgRevealOracleReport) ValidateBasic() error {
	_, err := sdk.ValAddressFromBech32(msg.Operator)
	return err
}

func (msg *MsgRevealOracleReport) GetSigners() []sdk.AccAddress {
	addr, err := sdk.ValAddressFromBech32(msg.Operator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{sdk.AccAddress(addr)}
}

// ComputeCommitHash generates the deterministic SHA-256 commit hash for an oracle round.
func ComputeCommitHash(operator, feedID string, roundID uint64, value uint64, nonce string) []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s:%s:%d:%d:%s", operator, feedID, roundID, value, nonce)))
	return h.Sum(nil)
}

// GenesisState defines the oracle module genesis state.
type GenesisState struct {
	Params    Params   `json:"params"`
	Operators []string `json:"operators"`
}

// ErrInvalidParams returns a formatted error for invalid genesis params.
func ErrInvalidParams(msg string) error {
	return fmt.Errorf("invalid oracle genesis params: %s", msg)
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

func (m *MsgCommitOracleHash) Marshal() ([]byte, error) {
	var buf []byte
	if len(m.Operator) > 0 {
		buf = append(buf, (1<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.Operator)))...)
		buf = append(buf, []byte(m.Operator)...)
	}
	if len(m.FeedID) > 0 {
		buf = append(buf, (2<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.FeedID)))...)
		buf = append(buf, []byte(m.FeedID)...)
	}
	if m.RoundID > 0 {
		buf = append(buf, (3<<3)|0)
		buf = append(buf, encodeVarint(m.RoundID)...)
	}
	if len(m.Hash) > 0 {
		buf = append(buf, (4<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.Hash)))...)
		buf = append(buf, m.Hash...)
	}
	return buf, nil
}

func (m *MsgCommitOracleHash) Size() int {
	var size int
	if len(m.Operator) > 0 {
		l := len(m.Operator)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if len(m.FeedID) > 0 {
		l := len(m.FeedID)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if m.RoundID > 0 {
		size += 1 + len(encodeVarint(m.RoundID))
	}
	if len(m.Hash) > 0 {
		l := len(m.Hash)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	return size
}

func (m *MsgCommitOracleHash) Unmarshal(data []byte) error {
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
			m.Operator = string(data[index : index+int(length)])
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
			m.FeedID = string(data[index : index+int(length)])
			index += int(length)
		case 3:
			if wireType != 0 {
				return fmt.Errorf("invalid wire type %d for field 3", wireType)
			}
			val, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid round ID varint")
			}
			index += n
			m.RoundID = val
		case 4:
			if wireType != 2 {
				return fmt.Errorf("invalid wire type %d for field 4", wireType)
			}
			length, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid length varint")
			}
			index += n
			m.Hash = make([]byte, length)
			copy(m.Hash, data[index:index+int(length)])
			index += int(length)
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

func (m *MsgRevealOracleReport) Marshal() ([]byte, error) {
	var buf []byte
	if len(m.Operator) > 0 {
		buf = append(buf, (1<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.Operator)))...)
		buf = append(buf, []byte(m.Operator)...)
	}
	if len(m.FeedID) > 0 {
		buf = append(buf, (2<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.FeedID)))...)
		buf = append(buf, []byte(m.FeedID)...)
	}
	if m.RoundID > 0 {
		buf = append(buf, (3<<3)|0)
		buf = append(buf, encodeVarint(m.RoundID)...)
	}
	if m.Value > 0 {
		buf = append(buf, (4<<3)|0)
		buf = append(buf, encodeVarint(m.Value)...)
	}
	if len(m.Nonce) > 0 {
		buf = append(buf, (5<<3)|2)
		buf = append(buf, encodeVarint(uint64(len(m.Nonce)))...)
		buf = append(buf, []byte(m.Nonce)...)
	}
	return buf, nil
}

func (m *MsgRevealOracleReport) Size() int {
	var size int
	if len(m.Operator) > 0 {
		l := len(m.Operator)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if len(m.FeedID) > 0 {
		l := len(m.FeedID)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	if m.RoundID > 0 {
		size += 1 + len(encodeVarint(m.RoundID))
	}
	if m.Value > 0 {
		size += 1 + len(encodeVarint(m.Value))
	}
	if len(m.Nonce) > 0 {
		l := len(m.Nonce)
		size += 1 + len(encodeVarint(uint64(l))) + l
	}
	return size
}

func (m *MsgRevealOracleReport) Unmarshal(data []byte) error {
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
			m.Operator = string(data[index : index+int(length)])
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
			m.FeedID = string(data[index : index+int(length)])
			index += int(length)
		case 3:
			if wireType != 0 {
				return fmt.Errorf("invalid wire type %d for field 3", wireType)
			}
			val, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid round ID varint")
			}
			index += n
			m.RoundID = val
		case 4:
			if wireType != 0 {
				return fmt.Errorf("invalid wire type %d for field 4", wireType)
			}
			val, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid value varint")
			}
			index += n
			m.Value = val
		case 5:
			if wireType != 2 {
				return fmt.Errorf("invalid wire type %d for field 5", wireType)
			}
			length, n := decodeVarint(data[index:])
			if n == 0 {
				return fmt.Errorf("invalid length varint")
			}
			index += n
			m.Nonce = string(data[index : index+int(length)])
			index += int(length)
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

var MsgCommitOracleHashDesc = []byte{
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xe2, 0xe2,
	0x49, 0x2e, 0x2d, 0x2e, 0xc9, 0xcf, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9,
	0x17, 0xe2, 0x2c, 0xce, 0x2f, 0x4b, 0x2d, 0x4a, 0xcd, 0x4c, 0xcf, 0x53,
	0x4a, 0xe7, 0x12, 0xf6, 0x2d, 0x4e, 0x77, 0xce, 0xcf, 0xcd, 0xcd, 0x2c,
	0xf1, 0x2f, 0x4a, 0x4c, 0xce, 0x49, 0xf5, 0x48, 0x2c, 0xce, 0x10, 0x12,
	0xe0, 0xe2, 0xc8, 0x2f, 0x48, 0x2d, 0x4a, 0x2c, 0xc9, 0x2f, 0x92, 0x60,
	0x54, 0x60, 0xd4, 0xe0, 0x14, 0xe2, 0xe7, 0x62, 0x4f, 0x4b, 0x4d, 0x4d,
	0x89, 0xcf, 0x4c, 0x91, 0x60, 0x02, 0x0b, 0x08, 0x70, 0x71, 0x14, 0xe5,
	0x97, 0xe6, 0x81, 0x45, 0x98, 0x15, 0x18, 0x35, 0x58, 0x84, 0x78, 0xb8,
	0x58, 0x32, 0x12, 0x8b, 0x33, 0x24, 0x58, 0x14, 0x18, 0x35, 0x78, 0xac,
	0x78, 0x9b, 0x9e, 0x6f, 0xd0, 0x82, 0x9b, 0x92, 0xc4, 0x06, 0xb6, 0xda,
	0x18, 0x10, 0x00, 0x00, 0xff, 0xff, 0xd9, 0xa2, 0x7d, 0xb7, 0x8a, 0x00,
	0x00, 0x00,
}

var MsgRevealOracleReportDesc = []byte{
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xe2, 0xe2,
	0x49, 0x2e, 0x2d, 0x2e, 0xc9, 0xcf, 0xd5, 0x2b, 0x28, 0xca, 0x2f, 0xc9,
	0x17, 0xe2, 0x2c, 0xce, 0x2f, 0x4b, 0x2d, 0x4a, 0xcd, 0x4c, 0xcf, 0x53,
	0xaa, 0xe4, 0x12, 0xf5, 0x2d, 0x4e, 0x0f, 0x4a, 0x2d, 0x4b, 0x4d, 0xcc,
	0xf1, 0x2f, 0x4a, 0x4c, 0xce, 0x49, 0x0d, 0x4a, 0x2d, 0xc8, 0x2f, 0x2a,
	0x11, 0x12, 0xe0, 0xe2, 0xc8, 0x2f, 0x48, 0x2d, 0x4a, 0x2c, 0xc9, 0x2f,
	0x92, 0x60, 0x54, 0x60, 0xd4, 0xe0, 0x14, 0xe2, 0xe7, 0x62, 0x4f, 0x4b,
	0x4d, 0x4d, 0x89, 0xcf, 0x4c, 0x91, 0x60, 0x02, 0x0b, 0x08, 0x70, 0x71,
	0x14, 0xe5, 0x97, 0xe6, 0x81, 0x45, 0x98, 0x15, 0x18, 0x35, 0x58, 0x84,
	0x78, 0xb9, 0x58, 0xcb, 0x12, 0x73, 0x4a, 0x53, 0x25, 0x58, 0x60, 0xdc,
	0xbc, 0xfc, 0xbc, 0xe4, 0x54, 0x09, 0x56, 0x90, 0x7a, 0x2b, 0xde, 0xa6,
	0xe7, 0x1b, 0xb4, 0xe0, 0xa6, 0x26, 0xb1, 0x81, 0x1d, 0x63, 0x0c, 0x08,
	0x00, 0x00, 0xff, 0xff, 0x36, 0x7d, 0x15, 0x17, 0x9c, 0x00, 0x00, 0x00,
}
