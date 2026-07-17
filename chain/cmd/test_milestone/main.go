package main

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/grpc"

	sdkclient "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
)

type Milestone struct {
	ID                 string `json:"id"`
	FeedID             string `json:"feed_id"`
	TargetPrice        uint64 `json:"target_price"`
	RemainingBlocks    int64  `json:"remaining_blocks"`
	State              string `json:"state"` // "pending", "stale-blocked", "achieved", "expired"
	VestingPoolAddress string `json:"vesting_pool_address"`
	PayoutAmount       uint64 `json:"payout_amount"`
}

func main() {
	conn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		panic(fmt.Errorf("failed to connect to gRPC: %w", err))
	}
	defer conn.Close()

	cmtClient := sdkclient.NewServiceClient(conn)

	// Query milestone store after tx execution
	// Path prefix for milestone index is 0x01 (KeyPrefixMilestone)
	key := append([]byte{0x01}, []byte("test-milestone-1")...)
	respMilestone, err := cmtClient.ABCIQuery(context.Background(), &sdkclient.ABCIQueryRequest{
		Path: "/store/milestone/key",
		Data: key,
	})
	if err != nil {
		fmt.Printf("Milestone query failed: %v\n", err)
		return
	}

	if len(respMilestone.Value) > 0 {
		var m Milestone
		err = json.Unmarshal(respMilestone.Value, &m)
		if err != nil {
			fmt.Printf("Failed to json-unmarshal milestone: %v. Raw value: %s\n", err, string(respMilestone.Value))
		} else {
			fmt.Printf("Milestone ID: %s, FeedID: %s, TargetPrice: %d, State: %s, RemainingBlocks: %d\n",
				m.ID, m.FeedID, m.TargetPrice, m.State, m.RemainingBlocks)
		}
	} else {
		fmt.Println("test-milestone-1 not found in milestone store")
	}
}
