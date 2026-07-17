package main

import (
	"context"
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"google.golang.org/grpc"

	sdkclient "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
)

func main() {
	conn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		panic(fmt.Errorf("failed to connect to gRPC: %w", err))
	}
	defer conn.Close()

	govClient := govtypes.NewQueryClient(conn)
	res, err := govClient.Proposal(context.Background(), &govtypes.QueryProposalRequest{ProposalId: 1})
	if err != nil {
		panic(fmt.Errorf("failed to query proposal: %w", err))
	}

	proposal := res.Proposal
	fmt.Printf("Proposal ID: %d\n", proposal.Id)
	fmt.Printf("Status: %s\n", proposal.Status.String())
	fmt.Printf("Voting End Time: %s\n", proposal.VotingEndTime)

	cmtClient := sdkclient.NewServiceClient(conn)
	blk, err := cmtClient.GetLatestBlock(context.Background(), &sdkclient.GetLatestBlockRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Current Block Time: %s, Height: %d\n", blk.Block.Header.Time, blk.Block.Header.Height)

	// Fetch votes
	votesRes, err := govClient.Votes(context.Background(), &govtypes.QueryVotesRequest{ProposalId: 1})
	if err == nil {
		fmt.Printf("Votes cast: %d\n", len(votesRes.Votes))
		for _, v := range votesRes.Votes {
			fmt.Printf("  - Voter: %s, Options: %v\n", v.Voter, v.Options)
		}
	}
}
