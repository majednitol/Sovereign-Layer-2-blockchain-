package certification

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc"
)

// MsgServer implements the certification message service handler.
type CertMsgServer struct {
	keeper         Keeper
	govAuthority   string // Governance module address (only authority allowed to update params)
}

func NewMsgServerImpl(keeper Keeper, govAuthority string) *CertMsgServer {
	return &CertMsgServer{
		keeper:       keeper,
		govAuthority: govAuthority,
	}
}

// UpdateCertificationParams handles governance-authorized parameter updates.
func (s *CertMsgServer) UpdateCertificationParams(goCtx context.Context, msg *MsgUpdateCertificationParams) (*MsgUpdateCertificationParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Only governance authority can update params
	if msg.Authority != s.govAuthority {
		return nil, fmt.Errorf("unauthorized: expected %s, got %s", s.govAuthority, msg.Authority)
	}

	// Validate params
	if msg.Params.MaxConsecutiveRejections <= 0 {
		return nil, fmt.Errorf("max_consecutive_rejections must be positive")
	}
	if msg.Params.MissedExtensionLimit <= 0 {
		return nil, fmt.Errorf("missed_extension_limit must be positive")
	}

	s.keeper.SetParams(ctx, msg.Params)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"certification_params_updated",
			sdk.NewAttribute("authority", msg.Authority),
			sdk.NewAttribute("max_consecutive_rejections", fmt.Sprintf("%d", msg.Params.MaxConsecutiveRejections)),
			sdk.NewAttribute("missed_extension_limit", fmt.Sprintf("%d", msg.Params.MissedExtensionLimit)),
		),
	)

	return &MsgUpdateCertificationParamsResponse{}, nil
}

// Response type
type MsgUpdateCertificationParamsResponse struct{}

func (res *MsgUpdateCertificationParamsResponse) Reset()         { *res = MsgUpdateCertificationParamsResponse{} }
func (res *MsgUpdateCertificationParamsResponse) String() string { return "" }
func (res *MsgUpdateCertificationParamsResponse) ProtoMessage()  {}

// MsgServiceServer interface for gRPC registration
type CertMsgServiceServer interface {
	UpdateCertificationParams(context.Context, *MsgUpdateCertificationParams) (*MsgUpdateCertificationParamsResponse, error)
}

// gRPC service descriptor
var CertMsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.certification.v1.Msg",
	HandlerType: (*CertMsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateCertificationParams",
			Handler:    _Msg_UpdateCertificationParams_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_UpdateCertificationParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateCertificationParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CertMsgServiceServer).UpdateCertificationParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.certification.v1.Msg/UpdateCertificationParams",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CertMsgServiceServer).UpdateCertificationParams(ctx, req.(*MsgUpdateCertificationParams))
	}
	return interceptor(ctx, in, info, handler)
}
