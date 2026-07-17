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
	Operator string `json:"operator"`
	FeedID   string `json:"feed_id"`
	RoundID  uint64 `json:"round_id"`
	Hash     []byte `json:"hash"`
}

func (msg *MsgCommitOracleHash) Reset()         { *msg = MsgCommitOracleHash{} }
func (msg *MsgCommitOracleHash) String() string { return msg.Operator }
func (msg *MsgCommitOracleHash) ProtoMessage()  {}

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
	Operator string `json:"operator"`
	FeedID   string `json:"feed_id"`
	RoundID  uint64 `json:"round_id"`
	Value    uint64 `json:"value"`
	Nonce    string `json:"nonce"`
}

func (msg *MsgRevealOracleReport) Reset()         { *msg = MsgRevealOracleReport{} }
func (msg *MsgRevealOracleReport) String() string { return msg.Operator }
func (msg *MsgRevealOracleReport) ProtoMessage()  {}

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
