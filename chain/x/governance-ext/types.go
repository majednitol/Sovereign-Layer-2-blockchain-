package gov_ext

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ModuleName = "govext"
	StoreKey   = "govext"
	RouterKey  = "govext"
)

var (
	ParamsKey = []byte{0x01}
)

type Params struct {
	MinGasLimit int64 `json:"min_gas_limit"` // e.g. 100,000
	MaxGasLimit int64 `json:"max_gas_limit"` // e.g. 2,000,000
}

type MsgMigrateContracts struct {
	Authority          string `json:"authority"`
	ContractAddress    string `json:"contract_address"`
	NewCodeID          uint64 `json:"new_code_id"`
	ExecutionDelaySecs int64  `json:"execution_delay_secs"` // must be >= 7 days (604,800s)
}

func (msg *MsgMigrateContracts) Reset()         { *msg = MsgMigrateContracts{} }
func (msg *MsgMigrateContracts) String() string { return msg.ContractAddress }
func (msg *MsgMigrateContracts) ProtoMessage()  {}

func (msg *MsgMigrateContracts) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.ContractAddress)
	return err
}

func (msg *MsgMigrateContracts) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type MsgUpdateGasLimit struct {
	Authority string `json:"authority"`
	GasLimit  int64  `json:"gas_limit"`
}

func (msg *MsgUpdateGasLimit) Reset()         { *msg = MsgUpdateGasLimit{} }
func (msg *MsgUpdateGasLimit) String() string { return msg.Authority }
func (msg *MsgUpdateGasLimit) ProtoMessage()  {}

func (msg *MsgUpdateGasLimit) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	return err
}

func (msg *MsgUpdateGasLimit) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type MsgUpdateValidatorSlot struct {
	Authority     string `json:"authority"`
	MaxValidators uint32 `json:"max_validators"`
}

func (msg *MsgUpdateValidatorSlot) Reset()         { *msg = MsgUpdateValidatorSlot{} }
func (msg *MsgUpdateValidatorSlot) String() string { return msg.Authority }
func (msg *MsgUpdateValidatorSlot) ProtoMessage()  {}

func (msg *MsgUpdateValidatorSlot) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	return err
}

func (msg *MsgUpdateValidatorSlot) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type MsgUpdateMilestone struct {
	Authority   string `json:"authority"`
	MilestoneID string `json:"milestone_id"`
	TargetPrice uint64 `json:"target_price"`
}

func (msg *MsgUpdateMilestone) Reset()         { *msg = MsgUpdateMilestone{} }
func (msg *MsgUpdateMilestone) String() string { return msg.Authority }
func (msg *MsgUpdateMilestone) ProtoMessage()  {}

func (msg *MsgUpdateMilestone) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	return err
}

func (msg *MsgUpdateMilestone) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type MsgUpdateOracleOperator struct {
	Authority       string `json:"authority"`
	OperatorAddress string `json:"operator_address"`
	Active          bool   `json:"active"`
}

func (msg *MsgUpdateOracleOperator) Reset()         { *msg = MsgUpdateOracleOperator{} }
func (msg *MsgUpdateOracleOperator) String() string { return msg.Authority }
func (msg *MsgUpdateOracleOperator) ProtoMessage()  {}

func (msg *MsgUpdateOracleOperator) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.OperatorAddress)
	return err
}

func (msg *MsgUpdateOracleOperator) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type MsgUpdateWitnessRegistry struct {
	Authority      string `json:"authority"`
	WitnessAddress string `json:"witness_address"`
	Active         bool   `json:"active"`
	PubKey         []byte `json:"pub_key"`
}

func (msg *MsgUpdateWitnessRegistry) Reset()         { *msg = MsgUpdateWitnessRegistry{} }
func (msg *MsgUpdateWitnessRegistry) String() string { return msg.Authority }
func (msg *MsgUpdateWitnessRegistry) ProtoMessage()  {}

func (msg *MsgUpdateWitnessRegistry) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.WitnessAddress)
	return err
}

func (msg *MsgUpdateWitnessRegistry) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}

type MsgUpdateBridgeRelayerSet struct {
	Authority      string `json:"authority"`
	RelayerAddress string `json:"relayer_address"`
	Active         bool   `json:"active"`
	PubKey         []byte `json:"pub_key"`
}

func (msg *MsgUpdateBridgeRelayerSet) Reset()         { *msg = MsgUpdateBridgeRelayerSet{} }
func (msg *MsgUpdateBridgeRelayerSet) String() string { return msg.Authority }
func (msg *MsgUpdateBridgeRelayerSet) ProtoMessage()  {}

func (msg *MsgUpdateBridgeRelayerSet) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		return err
	}
	_, err = sdk.AccAddressFromBech32(msg.RelayerAddress)
	return err
}

func (msg *MsgUpdateBridgeRelayerSet) GetSigners() []sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(msg.Authority)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{addr}
}
