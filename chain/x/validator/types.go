package validator

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
)

func init() {
	proto.RegisterType((*MsgFillValidatorSlot)(nil), "sovereign.validator.v1.MsgFillValidatorSlot")
	proto.RegisterType((*MsgEjectValidator)(nil), "sovereign.validator.v1.MsgEjectValidator")
	proto.RegisterType((*MsgUpdatePartitionScheme)(nil), "sovereign.validator.v1.MsgUpdatePartitionScheme")
}

const (
	ModuleName = "validator"
	StoreKey   = "validator"
	RouterKey  = "validator"
)

var (
	SlotKeyPrefix           = []byte{0x01}
	QueuedEjectionKeyPrefix = []byte{0x02}
	MaxValidatorsKeyPrefix  = []byte{0x03}
	PartitionSchemeKeyPrefix = []byte{0x04}
)

// MsgFillValidatorSlot represents a message to occupy an active validator slot.
type MsgFillValidatorSlot struct {
	ValidatorAddress string `json:"validator_address"`
}

func (msg *MsgFillValidatorSlot) Reset()         { *msg = MsgFillValidatorSlot{} }
func (msg *MsgFillValidatorSlot) String() string { return msg.ValidatorAddress }
func (msg *MsgFillValidatorSlot) ProtoMessage()  {}

func (msg *MsgFillValidatorSlot) ValidateBasic() error {
	if msg.ValidatorAddress == "" {
		return fmt.Errorf("validator address cannot be empty")
	}
	_, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	return nil
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
	if msg.ValidatorAddress == "" {
		return fmt.Errorf("validator address cannot be empty")
	}
	_, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	return nil
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
	if msg.Authority == "" {
		return fmt.Errorf("authority cannot be empty")
	}
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if strings.TrimSpace(msg.NewScheme) == "" {
		return fmt.Errorf("new scheme cannot be empty")
	}
	return nil
}

func (msg *MsgUpdatePartitionScheme) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

// GenesisState defines the validator module genesis state.
type GenesisState struct {
	MaxValidators    uint32   `json:"max_validators"`
	PartitionScheme  string   `json:"partition_scheme"`
	ActiveValidators []string `json:"active_validators"`
	QueuedEjections  []string `json:"queued_ejections"`
}
