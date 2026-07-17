package gov_ext

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"

	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func init() {
	proto.RegisterType((*MsgMigrateContractsResponse)(nil), "sovereign.govext.v1.MsgMigrateContractsResponse")
	proto.RegisterType((*MsgUpdateGasLimitResponse)(nil), "sovereign.govext.v1.MsgUpdateGasLimitResponse")
	proto.RegisterType((*MsgUpdateValidatorSlotResponse)(nil), "sovereign.govext.v1.MsgUpdateValidatorSlotResponse")
	proto.RegisterType((*MsgUpdateMilestoneResponse)(nil), "sovereign.govext.v1.MsgUpdateMilestoneResponse")
	proto.RegisterType((*MsgUpdateOracleOperatorResponse)(nil), "sovereign.govext.v1.MsgUpdateOracleOperatorResponse")
	proto.RegisterType((*MsgUpdateWitnessRegistryResponse)(nil), "sovereign.govext.v1.MsgUpdateWitnessRegistryResponse")
	proto.RegisterType((*MsgUpdateBridgeRelayerSetResponse)(nil), "sovereign.govext.v1.MsgUpdateBridgeRelayerSetResponse")

	strPtr := func(s string) *string { return &s }
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("chain/x/governance-ext/tx.proto"),
		Package: strPtr("sovereign.govext.v1"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("MsgMigrateContracts")},
			{Name: strPtr("MsgMigrateContractsResponse")},
			{Name: strPtr("MsgUpdateGasLimit")},
			{Name: strPtr("MsgUpdateGasLimitResponse")},
			{Name: strPtr("MsgUpdateValidatorSlot")},
			{Name: strPtr("MsgUpdateValidatorSlotResponse")},
			{Name: strPtr("MsgUpdateMilestone")},
			{Name: strPtr("MsgUpdateMilestoneResponse")},
			{Name: strPtr("MsgUpdateOracleOperator")},
			{Name: strPtr("MsgUpdateOracleOperatorResponse")},
			{Name: strPtr("MsgUpdateWitnessRegistry")},
			{Name: strPtr("MsgUpdateWitnessRegistryResponse")},
			{Name: strPtr("MsgUpdateBridgeRelayerSet")},
			{Name: strPtr("MsgUpdateBridgeRelayerSetResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("MigrateContracts"),
						InputType:  strPtr(".sovereign.govext.v1.MsgMigrateContracts"),
						OutputType: strPtr(".sovereign.govext.v1.MsgMigrateContractsResponse"),
					},
					{
						Name:       strPtr("UpdateGasLimit"),
						InputType:  strPtr(".sovereign.govext.v1.MsgUpdateGasLimit"),
						OutputType: strPtr(".sovereign.govext.v1.MsgUpdateGasLimitResponse"),
					},
					{
						Name:       strPtr("UpdateValidatorSlot"),
						InputType:  strPtr(".sovereign.govext.v1.MsgUpdateValidatorSlot"),
						OutputType: strPtr(".sovereign.govext.v1.MsgUpdateValidatorSlotResponse"),
					},
					{
						Name:       strPtr("UpdateMilestone"),
						InputType:  strPtr(".sovereign.govext.v1.MsgUpdateMilestone"),
						OutputType: strPtr(".sovereign.govext.v1.MsgUpdateMilestoneResponse"),
					},
					{
						Name:       strPtr("UpdateOracleOperator"),
						InputType:  strPtr(".sovereign.govext.v1.MsgUpdateOracleOperator"),
						OutputType: strPtr(".sovereign.govext.v1.MsgUpdateOracleOperatorResponse"),
					},
					{
						Name:       strPtr("UpdateWitnessRegistry"),
						InputType:  strPtr(".sovereign.govext.v1.MsgUpdateWitnessRegistry"),
						OutputType: strPtr(".sovereign.govext.v1.MsgUpdateWitnessRegistryResponse"),
					},
					{
						Name:       strPtr("UpdateBridgeRelayerSet"),
						InputType:  strPtr(".sovereign.govext.v1.MsgUpdateBridgeRelayerSet"),
						OutputType: strPtr(".sovereign.govext.v1.MsgUpdateBridgeRelayerSetResponse"),
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdProto, nil)
	if err != nil {
		panic(fmt.Sprintf("failed to compile dynamic file descriptor: %v", err))
	}

	_ = protoregistry.GlobalFiles.RegisterFile(fd)
}

type MsgServer struct {
	keeper       Keeper
	govAuthority string
}

func NewMsgServerImpl(keeper Keeper, govAuthority string) *MsgServer {
	return &MsgServer{
		keeper:       keeper,
		govAuthority: govAuthority,
	}
}

// MigrateContracts handles governance proposals to migrate Wasm contracts.
func (s *MsgServer) MigrateContracts(goCtx context.Context, msg *MsgMigrateContracts) (*MsgMigrateContractsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgMigrateContractsResponse{}, nil
}

// UpdateGasLimit handles governance proposals to update gas limit bounds.
func (s *MsgServer) UpdateGasLimit(goCtx context.Context, msg *MsgUpdateGasLimit) (*MsgUpdateGasLimitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgUpdateGasLimitResponse{}, nil
}

// UpdateValidatorSlot handles governance proposals to update max validator count.
func (s *MsgServer) UpdateValidatorSlot(goCtx context.Context, msg *MsgUpdateValidatorSlot) (*MsgUpdateValidatorSlotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgUpdateValidatorSlotResponse{}, nil
}

// UpdateMilestone handles governance proposals to update milestone parameters.
func (s *MsgServer) UpdateMilestone(goCtx context.Context, msg *MsgUpdateMilestone) (*MsgUpdateMilestoneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgUpdateMilestoneResponse{}, nil
}

// UpdateOracleOperator handles governance proposals to manage oracle operators.
func (s *MsgServer) UpdateOracleOperator(goCtx context.Context, msg *MsgUpdateOracleOperator) (*MsgUpdateOracleOperatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgUpdateOracleOperatorResponse{}, nil
}

// UpdateWitnessRegistry handles governance proposals to manage settlement witnesses.
func (s *MsgServer) UpdateWitnessRegistry(goCtx context.Context, msg *MsgUpdateWitnessRegistry) (*MsgUpdateWitnessRegistryResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgUpdateWitnessRegistryResponse{}, nil
}

// UpdateBridgeRelayerSet handles governance proposals to manage bridge relayers.
func (s *MsgServer) UpdateBridgeRelayerSet(goCtx context.Context, msg *MsgUpdateBridgeRelayerSet) (*MsgUpdateBridgeRelayerSetResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	if err := s.keeper.ExecuteProposal(ctx, msg); err != nil {
		return nil, err
	}

	return &MsgUpdateBridgeRelayerSetResponse{}, nil
}

// Response types
type MsgMigrateContractsResponse struct{}

func (res *MsgMigrateContractsResponse) Reset()         { *res = MsgMigrateContractsResponse{} }
func (res *MsgMigrateContractsResponse) String() string { return "" }
func (res *MsgMigrateContractsResponse) ProtoMessage()  {}

type MsgUpdateGasLimitResponse struct{}

func (res *MsgUpdateGasLimitResponse) Reset()         { *res = MsgUpdateGasLimitResponse{} }
func (res *MsgUpdateGasLimitResponse) String() string { return "" }
func (res *MsgUpdateGasLimitResponse) ProtoMessage()  {}

type MsgUpdateValidatorSlotResponse struct{}

func (res *MsgUpdateValidatorSlotResponse) Reset()         { *res = MsgUpdateValidatorSlotResponse{} }
func (res *MsgUpdateValidatorSlotResponse) String() string { return "" }
func (res *MsgUpdateValidatorSlotResponse) ProtoMessage()  {}

type MsgUpdateMilestoneResponse struct{}

func (res *MsgUpdateMilestoneResponse) Reset()         { *res = MsgUpdateMilestoneResponse{} }
func (res *MsgUpdateMilestoneResponse) String() string { return "" }
func (res *MsgUpdateMilestoneResponse) ProtoMessage()  {}

type MsgUpdateOracleOperatorResponse struct{}

func (res *MsgUpdateOracleOperatorResponse) Reset()         { *res = MsgUpdateOracleOperatorResponse{} }
func (res *MsgUpdateOracleOperatorResponse) String() string { return "" }
func (res *MsgUpdateOracleOperatorResponse) ProtoMessage()  {}

type MsgUpdateWitnessRegistryResponse struct{}

func (res *MsgUpdateWitnessRegistryResponse) Reset() {
	*res = MsgUpdateWitnessRegistryResponse{}
}
func (res *MsgUpdateWitnessRegistryResponse) String() string { return "" }
func (res *MsgUpdateWitnessRegistryResponse) ProtoMessage()  {}

type MsgUpdateBridgeRelayerSetResponse struct{}

func (res *MsgUpdateBridgeRelayerSetResponse) Reset() {
	*res = MsgUpdateBridgeRelayerSetResponse{}
}
func (res *MsgUpdateBridgeRelayerSetResponse) String() string { return "" }
func (res *MsgUpdateBridgeRelayerSetResponse) ProtoMessage()  {}

// MsgServiceServer interface for all governance extension message handlers.
type GovExtMsgServiceServer interface {
	MigrateContracts(context.Context, *MsgMigrateContracts) (*MsgMigrateContractsResponse, error)
	UpdateGasLimit(context.Context, *MsgUpdateGasLimit) (*MsgUpdateGasLimitResponse, error)
	UpdateValidatorSlot(context.Context, *MsgUpdateValidatorSlot) (*MsgUpdateValidatorSlotResponse, error)
	UpdateMilestone(context.Context, *MsgUpdateMilestone) (*MsgUpdateMilestoneResponse, error)
	UpdateOracleOperator(context.Context, *MsgUpdateOracleOperator) (*MsgUpdateOracleOperatorResponse, error)
	UpdateWitnessRegistry(context.Context, *MsgUpdateWitnessRegistry) (*MsgUpdateWitnessRegistryResponse, error)
	UpdateBridgeRelayerSet(context.Context, *MsgUpdateBridgeRelayerSet) (*MsgUpdateBridgeRelayerSetResponse, error)
}

var GovExtMsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.govext.v1.Msg",
	HandlerType: (*GovExtMsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "MigrateContracts",
			Handler:    _GovExt_MigrateContracts_Handler,
		},
		{
			MethodName: "UpdateGasLimit",
			Handler:    _GovExt_UpdateGasLimit_Handler,
		},
		{
			MethodName: "UpdateValidatorSlot",
			Handler:    _GovExt_UpdateValidatorSlot_Handler,
		},
		{
			MethodName: "UpdateMilestone",
			Handler:    _GovExt_UpdateMilestone_Handler,
		},
		{
			MethodName: "UpdateOracleOperator",
			Handler:    _GovExt_UpdateOracleOperator_Handler,
		},
		{
			MethodName: "UpdateWitnessRegistry",
			Handler:    _GovExt_UpdateWitnessRegistry_Handler,
		},
		{
			MethodName: "UpdateBridgeRelayerSet",
			Handler:    _GovExt_UpdateBridgeRelayerSet_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

func _GovExt_MigrateContracts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgMigrateContracts)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).MigrateContracts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/MigrateContracts"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).MigrateContracts(ctx, req.(*MsgMigrateContracts))
	}
	return interceptor(ctx, in, info, handler)
}

func _GovExt_UpdateGasLimit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateGasLimit)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).UpdateGasLimit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/UpdateGasLimit"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).UpdateGasLimit(ctx, req.(*MsgUpdateGasLimit))
	}
	return interceptor(ctx, in, info, handler)
}

func _GovExt_UpdateValidatorSlot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateValidatorSlot)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).UpdateValidatorSlot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/UpdateValidatorSlot"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).UpdateValidatorSlot(ctx, req.(*MsgUpdateValidatorSlot))
	}
	return interceptor(ctx, in, info, handler)
}

func _GovExt_UpdateMilestone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateMilestone)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).UpdateMilestone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/UpdateMilestone"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).UpdateMilestone(ctx, req.(*MsgUpdateMilestone))
	}
	return interceptor(ctx, in, info, handler)
}

func _GovExt_UpdateOracleOperator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateOracleOperator)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).UpdateOracleOperator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/UpdateOracleOperator"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).UpdateOracleOperator(ctx, req.(*MsgUpdateOracleOperator))
	}
	return interceptor(ctx, in, info, handler)
}

func _GovExt_UpdateWitnessRegistry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateWitnessRegistry)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).UpdateWitnessRegistry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/UpdateWitnessRegistry"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).UpdateWitnessRegistry(ctx, req.(*MsgUpdateWitnessRegistry))
	}
	return interceptor(ctx, in, info, handler)
}

func _GovExt_UpdateBridgeRelayerSet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateBridgeRelayerSet)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GovExtMsgServiceServer).UpdateBridgeRelayerSet(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/sovereign.govext.v1.Msg/UpdateBridgeRelayerSet"}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GovExtMsgServiceServer).UpdateBridgeRelayerSet(ctx, req.(*MsgUpdateBridgeRelayerSet))
	}
	return interceptor(ctx, in, info, handler)
}
