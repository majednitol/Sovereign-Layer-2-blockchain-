package bridge

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func init() {
	proto.RegisterType((*MsgBridgeInResponse)(nil), "sovereign.bridge.v1.MsgBridgeInResponse")
	proto.RegisterType((*MsgBridgeOutResponse)(nil), "sovereign.bridge.v1.MsgBridgeOutResponse")

	strPtr := func(s string) *string { return &s }
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:       strPtr("chain/x/bridge/tx.proto"),
		Package:    strPtr("sovereign.bridge.v1"),
		Syntax:     strPtr("proto3"),
		Dependency: []string{"cosmos/base/v1beta1/coin.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: strPtr("MsgBridgeIn"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("submitter"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("receiver"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("amount"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".cosmos.base.v1beta1.Coin")},
					{Name: strPtr("nonce"), Number: proto.Int32(4), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()},
					{Name: strPtr("signatures"), Number: proto.Int32(5), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_BYTES.Enum()},
				},
			},
			{Name: strPtr("MsgBridgeInResponse")},
			{
				Name: strPtr("MsgBridgeOut"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: strPtr("sender"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("bsc_recipient"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum()},
					{Name: strPtr("amount"), Number: proto.Int32(3), Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), TypeName: strPtr(".cosmos.base.v1beta1.Coin")},
				},
			},
			{Name: strPtr("MsgBridgeOutResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("BridgeIn"),
						InputType:  strPtr(".sovereign.bridge.v1.MsgBridgeIn"),
						OutputType: strPtr(".sovereign.bridge.v1.MsgBridgeInResponse"),
					},
					{
						Name:       strPtr("BridgeOut"),
						InputType:  strPtr(".sovereign.bridge.v1.MsgBridgeOut"),
						OutputType: strPtr(".sovereign.bridge.v1.MsgBridgeOutResponse"),
					},
				},
			},
		},
	}

	fd, err := protodesc.NewFile(fdProto, protoregistry.GlobalFiles)
	if err != nil {
		panic(fmt.Sprintf("failed to compile dynamic file descriptor: %v", err))
	}

	_ = protoregistry.GlobalFiles.RegisterFile(fd)
}

type MsgServer interface {
	BridgeIn(ctx context.Context, msg *MsgBridgeIn) (*MsgBridgeInResponse, error)
	BridgeOut(ctx context.Context, msg *MsgBridgeOut) (*MsgBridgeOutResponse, error)
}

type msgServer struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) MsgServer {
	return &msgServer{keeper: keeper}
}

type MsgBridgeInResponse struct{}

func (res *MsgBridgeInResponse) Reset()         { *res = MsgBridgeInResponse{} }
func (res *MsgBridgeInResponse) String() string { return "" }
func (res *MsgBridgeInResponse) ProtoMessage()  {}

type MsgBridgeOutResponse struct{}

func (res *MsgBridgeOutResponse) Reset()         { *res = MsgBridgeOutResponse{} }
func (res *MsgBridgeOutResponse) String() string { return "" }
func (res *MsgBridgeOutResponse) ProtoMessage()  {}

func (s *msgServer) BridgeIn(ctx context.Context, msg *MsgBridgeIn) (*MsgBridgeInResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := s.keeper.ProcessBridgeIn(sdkCtx, *msg); err != nil {
		return nil, err
	}
	return &MsgBridgeInResponse{}, nil
}

func (s *msgServer) BridgeOut(ctx context.Context, msg *MsgBridgeOut) (*MsgBridgeOutResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	if err := s.keeper.ProcessBridgeOut(sdkCtx, *msg); err != nil {
		return nil, err
	}
	return &MsgBridgeOutResponse{}, nil
}

// Manual gRPC ServiceDesc for Msg service registration in Cosmos SDK Configurator
var MsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.bridge.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "BridgeIn",
			Handler:    _Msg_BridgeIn_Handler,
		},
		{
			MethodName: "BridgeOut",
			Handler:    _Msg_BridgeOut_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "chain/x/bridge/types.go",
}

func _Msg_BridgeIn_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgBridgeIn)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).BridgeIn(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.bridge.v1.Msg/BridgeIn",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).BridgeIn(ctx, req.(*MsgBridgeIn))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_BridgeOut_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgBridgeOut)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).BridgeOut(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.bridge.v1.Msg/BridgeOut",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).BridgeOut(ctx, req.(*MsgBridgeOut))
	}
	return interceptor(ctx, in, info, handler)
}
