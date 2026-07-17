package settlement

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*MsgSettlement)(nil), "sovereign.settlement.v1.MsgSettlement")
}

const (
	ModuleName = "settlement"
	StoreKey   = "settlement"
	RouterKey  = "settlement"
)

var (
	WitnessKeyPrefix        = []byte{0x01}
	ParamsKey               = []byte{0x02}
	SettlementNonceKeyPrefix = []byte{0x03}
)

type Params struct {
	TimestampToleranceSeconds int64 `json:"timestamp_tolerance_seconds"`
}

type MsgSettlement struct {
	Submitter    string    `json:"submitter"`
	WitnessID    string    `json:"witness_id"`
	Timestamp    int64     `json:"timestamp"`
	PayloadHash  []byte    `json:"payload_hash"`
	Signature    []byte    `json:"signature"`
	TransferAmt  sdk.Coins `json:"transfer_amt"`
	TransferDest string    `json:"transfer_dest"`
}

func (msg *MsgSettlement) Reset()         { *msg = MsgSettlement{} }
func (msg *MsgSettlement) String() string { return msg.WitnessID }
func (msg *MsgSettlement) ProtoMessage()  {}

func (msg *MsgSettlement) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	_, err = sdk.AccAddressFromBech32(msg.TransferDest)
	if err != nil {
		return fmt.Errorf("invalid transfer destination address: %w", err)
	}
	if msg.WitnessID == "" {
		return fmt.Errorf("witness ID cannot be empty")
	}
	if msg.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}
	if len(msg.PayloadHash) == 0 {
		return fmt.Errorf("payload hash cannot be empty")
	}
	if len(msg.Signature) == 0 {
		return fmt.Errorf("signature cannot be empty")
	}
	if msg.TransferAmt.Empty() || !msg.TransferAmt.IsValid() || !msg.TransferAmt.IsAllPositive() {
		return fmt.Errorf("transfer amount must be valid and positive")
	}
	return nil
}

func (msg *MsgSettlement) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type Witness struct {
	ID     string `json:"id"`
	PubKey []byte `json:"pub_key"`
}

type GenesisState struct {
	Params          Params    `json:"params"`
	Witnesses       []Witness `json:"witnesses"`
	ProcessedNonces [][]byte  `json:"processed_nonces"`
}

// ComputeDomainSeparator computes the domain separator bound to chain_id.
func ComputeDomainSeparator(chainID string, payloadHash []byte) []byte {
	h := sha256.New()
	h.Write([]byte(chainID))
	h.Write([]byte("x/settlement"))
	h.Write([]byte("WitnessPayload"))
	h.Write(payloadHash)
	return h.Sum(nil)
}
