package validator

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
	proto.RegisterType((*MsgFillValidatorSlotResponse)(nil), "sovereign.validator.v1.MsgFillValidatorSlotResponse")
	proto.RegisterType((*MsgEjectValidatorResponse)(nil), "sovereign.validator.v1.MsgEjectValidatorResponse")
	proto.RegisterType((*MsgUpdatePartitionSchemeResponse)(nil), "sovereign.validator.v1.MsgUpdatePartitionSchemeResponse")

	strPtr := func(s string) *string { return &s }
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("chain/x/validator/tx.proto"),
		Package: strPtr("sovereign.validator.v1"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("MsgFillValidatorSlot"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("validator_address"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{Name: strPtr("MsgFillValidatorSlotResponse")},
			{
				Name: strPtr("MsgEjectValidator"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("validator_address"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{Name: strPtr("MsgEjectValidatorResponse")},
			{
				Name: strPtr("MsgUpdatePartitionScheme"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("authority"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("new_scheme"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
				},
			},
			{Name: strPtr("MsgUpdatePartitionSchemeResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("FillValidatorSlot"),
						InputType:  strPtr(".sovereign.validator.v1.MsgFillValidatorSlot"),
						OutputType: strPtr(".sovereign.validator.v1.MsgFillValidatorSlotResponse"),
					},
					{
						Name:       strPtr("EjectValidator"),
						InputType:  strPtr(".sovereign.validator.v1.MsgEjectValidator"),
						OutputType: strPtr(".sovereign.validator.v1.MsgEjectValidatorResponse"),
					},
					{
						Name:       strPtr("UpdatePartitionScheme"),
						InputType:  strPtr(".sovereign.validator.v1.MsgUpdatePartitionScheme"),
						OutputType: strPtr(".sovereign.validator.v1.MsgUpdatePartitionSchemeResponse"),
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

func (s *MsgServer) FillValidatorSlot(goCtx context.Context, msg *MsgFillValidatorSlot) (*MsgFillValidatorSlotResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	if s.keeper.IsValidatorActive(ctx, valAddr) {
		return nil, fmt.Errorf("validator is already active")
	}

	s.keeper.SetValidatorActive(ctx, valAddr)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_slot_filled",
			sdk.NewAttribute("validator", msg.ValidatorAddress),
		),
	)

	return &MsgFillValidatorSlotResponse{}, nil
}

func (s *MsgServer) EjectValidator(goCtx context.Context, msg *MsgEjectValidator) (*MsgEjectValidatorResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	if !s.keeper.IsValidatorActive(ctx, valAddr) {
		return nil, fmt.Errorf("validator is not active")
	}

	s.keeper.QueueEjection(ctx, valAddr)
	s.keeper.RemoveValidatorActive(ctx, valAddr)

	// Tombstone the validator CONS address if found in staking
	val, err := s.keeper.stakingKeeper.GetValidator(ctx, valAddr)
	if err == nil {
		consAddr, err := val.GetConsAddr()
		if err == nil {
			_ = s.keeper.slashingKeeper.Jail(ctx, consAddr)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_ejected",
			sdk.NewAttribute("validator", msg.ValidatorAddress),
		),
	)

	return &MsgEjectValidatorResponse{}, nil
}

func (s *MsgServer) UpdatePartitionScheme(goCtx context.Context, msg *MsgUpdatePartitionScheme) (*MsgUpdatePartitionSchemeResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	s.keeper.SetPartitionScheme(ctx, msg.NewScheme)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"validator_partition_scheme_updated",
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("new_scheme", msg.NewScheme),
		),
	)

	return &MsgUpdatePartitionSchemeResponse{}, nil
}

type MsgFillValidatorSlotResponse struct{}
func (res *MsgFillValidatorSlotResponse) Reset()         { *res = MsgFillValidatorSlotResponse{} }
func (res *MsgFillValidatorSlotResponse) String() string { return "" }
func (res *MsgFillValidatorSlotResponse) ProtoMessage()  {}

type MsgEjectValidatorResponse struct{}
func (res *MsgEjectValidatorResponse) Reset()         { *res = MsgEjectValidatorResponse{} }
func (res *MsgEjectValidatorResponse) String() string { return "" }
func (res *MsgEjectValidatorResponse) ProtoMessage()  {}

type MsgUpdatePartitionSchemeResponse struct{}
func (res *MsgUpdatePartitionSchemeResponse) Reset()         { *res = MsgUpdatePartitionSchemeResponse{} }
func (res *MsgUpdatePartitionSchemeResponse) String() string { return "" }
func (res *MsgUpdatePartitionSchemeResponse) ProtoMessage()  {}

type MsgServiceServer interface {
	FillValidatorSlot(context.Context, *MsgFillValidatorSlot) (*MsgFillValidatorSlotResponse, error)
	EjectValidator(context.Context, *MsgEjectValidator) (*MsgEjectValidatorResponse, error)
	UpdatePartitionScheme(context.Context, *MsgUpdatePartitionScheme) (*MsgUpdatePartitionSchemeResponse, error)
}

var MsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.validator.v1.Msg",
	HandlerType: (*MsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "FillValidatorSlot",
			Handler:    _Msg_FillValidatorSlot_Handler,
		},
		{
			MethodName: "EjectValidator",
			Handler:    _Msg_EjectValidator_Handler,
		},
		{
			MethodName: "UpdatePartitionScheme",
			Handler:    _Msg_UpdatePartitionScheme_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_FillValidatorSlot_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgFillValidatorSlot)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).FillValidatorSlot(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.validator.v1.Msg/FillValidatorSlot",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).FillValidatorSlot(ctx, req.(*MsgFillValidatorSlot))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_EjectValidator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgEjectValidator)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).EjectValidator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.validator.v1.Msg/EjectValidator",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).EjectValidator(ctx, req.(*MsgEjectValidator))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UpdatePartitionScheme_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdatePartitionScheme)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).UpdatePartitionScheme(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.validator.v1.Msg/UpdatePartitionScheme",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).UpdatePartitionScheme(ctx, req.(*MsgUpdatePartitionScheme))
	}
	return interceptor(ctx, in, info, handler)
}
