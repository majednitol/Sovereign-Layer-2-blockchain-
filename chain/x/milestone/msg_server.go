package milestone

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"google.golang.org/grpc"

	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func init() {
	proto.RegisterType((*MsgCreateMilestoneResponse)(nil), "sovereign.milestone.v1.MsgCreateMilestoneResponse")

	strPtr := func(s string) *string { return &s }
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("chain/x/milestone/tx.proto"),
		Package: strPtr("sovereign.milestone.v1"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("MsgCreateMilestone")},
			{Name: strPtr("MsgCreateMilestoneResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("CreateMilestone"),
						InputType:  strPtr(".sovereign.milestone.v1.MsgCreateMilestone"),
						OutputType: strPtr(".sovereign.milestone.v1.MsgCreateMilestoneResponse"),
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

// MsgServer implements the milestone message service handler.
type MilestoneMsgServer struct {
	keeper       Keeper
	govAuthority string
}

func NewMsgServerImpl(keeper Keeper, govAuthority string) *MilestoneMsgServer {
	return &MilestoneMsgServer{
		keeper:       keeper,
		govAuthority: govAuthority,
	}
}

// CreateMilestone handles governance-authorized milestone creation.
func (s *MilestoneMsgServer) CreateMilestone(goCtx context.Context, msg *MsgCreateMilestone) (*MsgCreateMilestoneResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	// Only governance authority can create milestones
	if msg.Creator != s.govAuthority {
		// Also allow the module account itself (for genesis-seeded milestones)
		moduleAddr := authtypes.NewModuleAddress(ModuleName).String()
		if msg.Creator != moduleAddr {
			return nil, fmt.Errorf("unauthorized: only governance (%s) can create milestones, got %s", s.govAuthority, msg.Creator)
		}
	}

	// Check if milestone already exists
	if _, exists := s.keeper.GetMilestone(ctx, msg.ID); exists {
		return nil, fmt.Errorf("milestone %s already exists", msg.ID)
	}

	m := Milestone{
		ID:                 msg.ID,
		FeedID:             msg.FeedID,
		TargetPrice:        msg.TargetPrice,
		RemainingBlocks:    msg.DurationBlocks,
		State:              StatePending,
		VestingPoolAddress: msg.VestingPoolAddress,
		PayoutAmount:       msg.PayoutAmount,
	}

	s.keeper.SetMilestone(ctx, m)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"milestone_created",
			sdk.NewAttribute("milestone_id", msg.ID),
			sdk.NewAttribute("feed_id", msg.FeedID),
			sdk.NewAttribute("target_price", fmt.Sprintf("%d", msg.TargetPrice)),
			sdk.NewAttribute("duration_blocks", fmt.Sprintf("%d", msg.DurationBlocks)),
		),
	)

	return &MsgCreateMilestoneResponse{}, nil
}

// Response type
type MsgCreateMilestoneResponse struct{}

func (res *MsgCreateMilestoneResponse) Reset()         { *res = MsgCreateMilestoneResponse{} }
func (res *MsgCreateMilestoneResponse) String() string { return "" }
func (res *MsgCreateMilestoneResponse) ProtoMessage()  {}

// MilestoneMsgServiceServer interface for gRPC registration
type MilestoneMsgServiceServer interface {
	CreateMilestone(context.Context, *MsgCreateMilestone) (*MsgCreateMilestoneResponse, error)
}

// gRPC service descriptor
var MilestoneMsgServiceDesc = grpc.ServiceDesc{
	ServiceName: "sovereign.milestone.v1.Msg",
	HandlerType: (*MilestoneMsgServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateMilestone",
			Handler:    _Msg_CreateMilestone_Handler,
		},
	},
	Streams: []grpc.StreamDesc{},
}

func _Msg_CreateMilestone_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateMilestone)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MilestoneMsgServiceServer).CreateMilestone(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/sovereign.milestone.v1.Msg/CreateMilestone",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MilestoneMsgServiceServer).CreateMilestone(ctx, req.(*MsgCreateMilestone))
	}
	return interceptor(ctx, in, info, handler)
}
