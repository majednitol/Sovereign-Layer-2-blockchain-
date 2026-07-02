package bridge

import (
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*MsgBridgeIn)(nil), "sovereign.bridge.v1.MsgBridgeIn")
	proto.RegisterType((*MsgBridgeOut)(nil), "sovereign.bridge.v1.MsgBridgeOut")
}

const (
	ModuleName = "bridge"
	StoreKey   = "bridge"
	RouterKey  = "bridge"
)

var (
	NonceKeyPrefix   = []byte{0x01}
	RelayerKeyPrefix = []byte{0x02}
	ParamsKey        = []byte{0x03}
)

type Params struct {
	StandardFinalityDepth uint32    `json:"standard_finality_depth"`
	LargeFinalityDepth    uint32    `json:"large_finality_depth"`
	LargeTransferThreshold uint64   `json:"large_transfer_threshold"`
	QuorumThreshold       uint32    `json:"quorum_threshold"`
	MaxUnlockPerBlock     uint64    `json:"max_unlock_per_block"`
	CircuitBreakerAddress string    `json:"circuit_breaker_address"`
	GnosisSafeAddress     string    `json:"gnosis_safe_address"`
	SupplyCap             uint64    `json:"supply_cap"`
	LockBoxAddress        string    `json:"lockbox_address"`
}

type Relayer struct {
	Address string `json:"address"`
	PubKey  []byte `json:"pub_key"` // secp256k1 public key
}

type GenesisState struct {
	Params       Params    `json:"params"`
	Relayers     []Relayer `json:"relayers"`
	CosmosMinted uint64    `json:"cosmos_minted"`
}

// MsgBridgeIn represents a deposit confirmation payload submitted by a relayer.
type MsgBridgeIn struct {
	Submitter  string    `json:"submitter"`
	Receiver   string    `json:"receiver"`
	Amount     sdk.Coins `json:"amount"`
	Nonce      []byte    `json:"nonce"`
	Signatures [][]byte  `json:"signatures"`
}

func (msg *MsgBridgeIn) Reset()         { *msg = MsgBridgeIn{} }
func (msg *MsgBridgeIn) String() string { return msg.Receiver }
func (msg *MsgBridgeIn) ProtoMessage()  {}

func (msg *MsgBridgeIn) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return err
	}
	if msg.Amount.IsZero() || !msg.Amount.IsValid() {
		return fmt.Errorf("amount must be valid and positive")
	}
	if len(msg.Nonce) == 0 {
		return fmt.Errorf("nonce cannot be empty")
	}
	if len(msg.Signatures) == 0 {
		return fmt.Errorf("signatures cannot be empty")
	}
	return nil
}

func (msg *MsgBridgeIn) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Submitter)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// MsgBridgeOut represents a burn action to initiate a withdrawal to BSC.
type MsgBridgeOut struct {
	Sender       string    `json:"sender"`
	BscRecipient string    `json:"bsc_recipient"` // hex address on BSC
	Amount       sdk.Coins `json:"amount"`
}

func (msg *MsgBridgeOut) Reset()         { *msg = MsgBridgeOut{} }
func (msg *MsgBridgeOut) String() string { return msg.BscRecipient }
func (msg *MsgBridgeOut) ProtoMessage()  {}

func (msg *MsgBridgeOut) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return err
	}
	if len(msg.BscRecipient) == 0 {
		return fmt.Errorf("BSC recipient cannot be empty")
	}
	if msg.Amount.IsZero() || !msg.Amount.IsValid() {
		return fmt.Errorf("amount must be valid and positive")
	}
	return nil
}

func (msg *MsgBridgeOut) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// ComputeBridgeMessageHash computes the message hash that relayers sign.
func ComputeBridgeMessageHash(receiver string, amount sdk.Coins, nonce []byte) []byte {
	h := sha256.New()
	h.Write([]byte(receiver))
	h.Write([]byte(amount.String()))
	h.Write(nonce)
	return h.Sum(nil)
}
