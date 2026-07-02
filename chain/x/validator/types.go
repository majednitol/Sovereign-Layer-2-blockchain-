package validator

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "validator"
	StoreKey   = "validator"
	RouterKey  = "validator"
)

var (
	SlotKeyPrefix           = []byte{0x01}
	QueuedEjectionKeyPrefix = []byte{0x02}
	MaxValidatorsKeyPrefix  = []byte{0x03}
)

// MsgFillValidatorSlot represents a message to occupy an active validator slot.
type MsgFillValidatorSlot struct {
	ValidatorAddress string `json:"validator_address"`
}

func (msg *MsgFillValidatorSlot) Reset()         { *msg = MsgFillValidatorSlot{} }
func (msg *MsgFillValidatorSlot) String() string { return msg.ValidatorAddress }
func (msg *MsgFillValidatorSlot) ProtoMessage()  {}

func (msg *MsgFillValidatorSlot) ValidateBasic() error {
	_, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	return err
}

func (msg *MsgFillValidatorSlot) GetSigners() []sdk.AccAddress {
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{sdk.AccAddress(valAddr)}
}

// MsgEjectValidator represents a governance-driven request to eject an active validator.
type MsgEjectValidator struct {
	ValidatorAddress string `json:"validator_address"`
}

func (msg *MsgEjectValidator) Reset()         { *msg = MsgEjectValidator{} }
func (msg *MsgEjectValidator) String() string { return msg.ValidatorAddress }
func (msg *MsgEjectValidator) ProtoMessage()  {}

func (msg *MsgEjectValidator) ValidateBasic() error {
	_, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	return err
}

func (msg *MsgEjectValidator) GetSigners() []sdk.AccAddress {
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{sdk.AccAddress(valAddr)}
}

// MsgUpdatePartitionScheme represents a governance proposal to update partition scheme.
type MsgUpdatePartitionScheme struct {
	Authority string `json:"authority"`
	NewScheme string `json:"new_scheme"`
}

func (msg *MsgUpdatePartitionScheme) Reset()         { *msg = MsgUpdatePartitionScheme{} }
func (msg *MsgUpdatePartitionScheme) String() string { return msg.NewScheme }
func (msg *MsgUpdatePartitionScheme) ProtoMessage()  {}

func (msg *MsgUpdatePartitionScheme) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	return err
}

func (msg *MsgUpdatePartitionScheme) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
