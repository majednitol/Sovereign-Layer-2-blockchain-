package oracle

import (
	"context"

	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

func init() {
	proto.RegisterType((*MsgCommitOracleHashResponse)(nil), "sovereign.oracle.v1.MsgCommitOracleHashResponse")
	proto.RegisterType((*MsgRevealOracleReportResponse)(nil), "sovereign.oracle.v1.MsgRevealOracleReportResponse")

	strPtr := func(s string) *string { return &s }
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:    strPtr("chain/x/oracle/tx.proto"),
		Package: strPtr("sovereign.oracle.v1"),
		Syntax:  strPtr("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: strPtr("MsgCommitOracleHash")},
			{Name: strPtr("MsgCommitOracleHashResponse")},
			{Name: strPtr("MsgRevealOracleReport")},
			{Name: strPtr("MsgRevealOracleReportResponse")},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: strPtr("Msg"),
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:       strPtr("SubmitOracleCommit"),
						InputType:  strPtr(".sovereign.oracle.v1.MsgCommitOracleHash"),
						OutputType: strPtr(".sovereign.oracle.v1.MsgCommitOracleHashResponse"),
					},
					{
						Name:       strPtr("RevealOracleReport"),
						InputType:  strPtr(".sovereign.oracle.v1.MsgRevealOracleReport"),
						OutputType: strPtr(".sovereign.oracle.v1.MsgRevealOracleReportResponse"),
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

// MsgServer implements the oracle message service handler.
// This routes incoming gRPC transactions to the keeper methods.
type MsgServer struct {
	keeper Keeper
}

func NewMsgServer(keeper Keeper) *MsgServer {
	return &MsgServer{keeper: keeper}
}

// SubmitOracleCommit handles MsgCommitOracleHash transactions.
func (s *MsgServer) SubmitOracleCommit(goCtx context.Context, msg *MsgCommitOracleHash) (*MsgCommitOracleHashResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := s.keeper.CommitHash(ctx, msg.Operator, msg.FeedID, msg.RoundID, msg.Hash); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_commit",
			sdk.NewAttribute("operator", msg.Operator),
			sdk.NewAttribute("feed_id", msg.FeedID),
		),
	)

	return &MsgCommitOracleHashResponse{}, nil
}

// RevealOracleReport handles MsgRevealOracleReport transactions.
func (s *MsgServer) RevealOracleReport(goCtx context.Context, msg *MsgRevealOracleReport) (*MsgRevealOracleReportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	if err := s.keeper.RevealReport(ctx, msg.Operator, msg.FeedID, msg.RoundID, msg.Value, msg.Nonce); err != nil {
		return nil, err
	}

	// Attempt aggregation after each reveal
	params := s.keeper.GetParams(ctx)
	values := s.keeper.GetRevealedValues(ctx, msg.FeedID, msg.RoundID)
	if int64(len(values)) >= params.MinOperatorCommits {
		price, err := s.keeper.AggregateRound(ctx, msg.FeedID, msg.RoundID)
		if err == nil {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"oracle_aggregated",
					sdk.NewAttribute("feed_id", msg.FeedID),
					sdk.NewAttribute("price", math.NewInt(int64(price)).String()),
				),
			)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"oracle_reveal",
			sdk.NewAttribute("operator", msg.Operator),
			sdk.NewAttribute("feed_id", msg.FeedID),
		),
	)

	return &MsgRevealOracleReportResponse{}, nil
}

// Response types for the message server.
type MsgCommitOracleHashResponse struct{}
func (res *MsgCommitOracleHashResponse) Reset()         { *res = MsgCommitOracleHashResponse{} }
func (res *MsgCommitOracleHashResponse) String() string { return "" }
func (res *MsgCommitOracleHashResponse) ProtoMessage()  {}

type MsgRevealOracleReportResponse struct{}
func (res *MsgRevealOracleReportResponse) Reset()         { *res = MsgRevealOracleReportResponse{} }
func (res *MsgRevealOracleReportResponse) String() string { return "" }
func (res *MsgRevealOracleReportResponse) ProtoMessage()  {}
