package settlement

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

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
