package settlement

import (
	"crypto/sha256"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "settlement"
	StoreKey   = "settlement"
	RouterKey  = "settlement"
)

var (
	WitnessKeyPrefix = []byte{0x01}
	ParamsKey        = []byte{0x02}
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
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.TransferDest)
	return err
}

func (msg *MsgSettlement) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
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
