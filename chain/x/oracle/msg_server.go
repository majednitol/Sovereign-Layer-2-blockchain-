package oracle

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

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
type MsgRevealOracleReportResponse struct{}
