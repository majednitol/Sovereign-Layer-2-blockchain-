package settlement

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
	proto.RegisterType((*MsgSettlementResponse)(nil), "sovereign.settlement.v1.MsgSettlementResponse")

	strPtr := func(s string) *string { return &s }
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("chain/x/settlement/tx.proto"),
		Package: strPtr("sovereign.settlement.v1"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("MsgSettlement")},
			{Name: strPtr("MsgSettlementResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("Settlement"),
						InputType:  strPtr(".sovereign.settlement.v1.MsgSettlement"),
						OutputType: strPtr(".sovereign.settlement.v1.MsgSettlementResponse"),
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
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) *MsgServer {
	return &MsgServer{
		keeper: keeper,
	}
}

func (s *MsgServer) Settlement(goCtx context.Context, msg *MsgSettlement) (*MsgSettlementResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	err := s.keeper.ProcessSettlement(ctx, *msg)
	if err != nil {
		return nil, err
	}

	return &MsgSettlementResponse{}, nil
}

type MsgSettlementResponse struct{}

func (res *MsgSettlementResponse) Reset()         { *res = MsgSettlementResponse{} }
func (res *MsgSettlementResponse) String() string { return "" }
func (res *MsgSettlementResponse) ProtoMessage()  {}

type MsgServiceServer interface {
	Settlement(context.Context, *MsgSettlement) (*MsgSettlementResponse, error)
}

var MsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.settlement.v1.Msg",
	HandlerType: (*MsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Settlement",
			Handler:    _Msg_Settlement_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_Settlement_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSettlement)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServiceServer).Settlement(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.settlement.v1.Msg/Settlement",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServiceServer).Settlement(ctx, req.(*MsgSettlement))
	}
	return interceptor(ctx, in, info, handler)
}
