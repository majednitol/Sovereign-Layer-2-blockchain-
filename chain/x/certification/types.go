package certification

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "certification"
	StoreKey   = "certification"
	RouterKey  = "certification"
)

var (
	ConsecutiveRejectionKey = []byte{0x01}
	DegradedModeKey         = []byte{0x02}
	ParamsKey               = []byte{0x03}
	MissedExtensionsKey     = []byte{0x04}
	SignedBitPrefix         = []byte{0x05}
	SignedCountPrefix       = []byte{0x06}
	AttestationKeyPrefix    = []byte{0x07}
)

type Params struct {
	MaxConsecutiveRejections int64 `json:"max_consecutive_rejections"`
	MissedExtensionLimit    int64 `json:"missed_extension_limit"`
}

// MsgUpdateCertificationParams defines the message to update certification parameters via governance.
type MsgUpdateCertificationParams struct {
	Authority string `json:"authority"`
	Params    Params `json:"params"`
}

func (msg *MsgUpdateCertificationParams) Reset()         { *msg = MsgUpdateCertificationParams{} }
func (msg *MsgUpdateCertificationParams) String() string { return msg.Authority }
func (msg *MsgUpdateCertificationParams) ProtoMessage()  {}

func (msg *MsgUpdateCertificationParams) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	return err
}

func (msg *MsgUpdateCertificationParams) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// CertGenesisState defines the certification module genesis state.
type CertGenesisState struct {
	Params                   Params   `json:"params"`
	DegradedMode             bool     `json:"degraded_mode"`
	ConsecutiveRejectionCount int64   `json:"consecutive_rejection_count"`
	AttestedValidators       []string `json:"attested_validators"`
}
