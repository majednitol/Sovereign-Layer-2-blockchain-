package main

import (
	"context"
	"encoding/json"
	"fmt"

	sdkclient "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"google.golang.org/grpc"
)

type AggregatePrice struct {
	Price       uint64 `json:"price"`
	BlockHeight int64  `json:"block_height"`
}

func main() {
	conn, err := grpc.Dial("127.0.0.1:9090", grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to dial gRPC: %v\n", err)
		return
	}
	defer conn.Close()

	client := sdkclient.NewServiceClient(conn)

	// Query BTC_USD aggregated price
	keyBtc := append([]byte{0x03}, []byte("BTC_USD")...)
	respBtc, err := client.ABCIQuery(context.Background(), &sdkclient.ABCIQueryRequest{
		Path: "/store/oracle/key",
		Data: keyBtc,
	})
	if err != nil {
		fmt.Printf("BTC query failed: %v\n", err)
		return
	}

	if len(respBtc.Value) > 0 {
		var priceBtc AggregatePrice
		err = json.Unmarshal(respBtc.Value, &priceBtc)
		if err != nil {
			fmt.Printf("Failed to unmarshal BTC price: %v\n", err)
		} else {
			fmt.Printf("BTC_USD Price: %d, Height: %d\n", priceBtc.Price, priceBtc.BlockHeight)
		}
	} else {
		fmt.Println("BTC_USD Price not found in store")
	}

	// Query ETH_USD aggregated price
	keyEth := append([]byte{0x03}, []byte("ETH_USD")...)
	respEth, err := client.ABCIQuery(context.Background(), &sdkclient.ABCIQueryRequest{
		Path: "/store/oracle/key",
		Data: keyEth,
	})
	if err != nil {
		fmt.Printf("ETH query failed: %v\n", err)
		return
	}

	if len(respEth.Value) > 0 {
		var priceEth AggregatePrice
		err = json.Unmarshal(respEth.Value, &priceEth)
		if err != nil {
			fmt.Printf("Failed to unmarshal ETH price: %v\n", err)
		} else {
			fmt.Printf("ETH_USD Price: %d, Height: %d\n", priceEth.Price, priceEth.BlockHeight)
		}
	} else {
		fmt.Println("ETH_USD Price not found in store")
	}
}
