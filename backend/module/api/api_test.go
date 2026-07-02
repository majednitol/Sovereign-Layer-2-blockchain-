package main

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	backendv1 "github.com/sovereign-l1/chain/api/backend/v1"
)

func TestStreamBridgeEvents(t *testing.T) {
	// 1. Try to connect to local NATS (docker compose exposes NATS on 4222)
	nc, err := nats.Connect("nats://localhost:4222", nats.Timeout(1*time.Second))
	if err != nil {
		t.Skip("NATS is not running on localhost:4222. Skipping streaming integration test.")
		return
	}
	defer nc.Close()

	// 2. Setup standard gRPC server on a random port
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := &server{
		nc: nc,
	}

	grpcServer := grpc.NewServer()
	backendv1.RegisterStreamServiceServer(grpcServer, s)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	// 3. Connect gRPC client to the test server
	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := backendv1.NewStreamServiceClient(conn)

	// 4. Start subscription stream
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.StreamBridgeEvents(ctx, &backendv1.StreamBridgeEventsRequest{
		TokenAddress: "usov",
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// 5. Publish mock event to NATS in a goroutine
	go func() {
		time.Sleep(200 * time.Millisecond)
		ev := EventRecord{
			BlockHeight: 123,
			EventIndex:  1,
			EventType:   "MsgBridgeIn",
			Payload:     json.RawMessage(`{"receiver":"usov","amount":"5000000","sender":"bsc_sender"}`),
		}
		data, _ := json.Marshal(ev)
		_ = nc.Publish(NatsStreamSubject, data)
	}()

	// 6. Receive event from stream and assert
	resp, err := stream.Recv()
	if err != nil {
		t.Fatalf("failed to receive event: %v", err)
	}

	if resp.BlockHeight != 123 {
		t.Errorf("expected block height 123, got %d", resp.BlockHeight)
	}
	if resp.Amount != "5000000" {
		t.Errorf("expected amount 5000000, got %s", resp.Amount)
	}
	if resp.TokenAddress != "usov" {
		t.Errorf("expected token address usov, got %s", resp.TokenAddress)
	}
}
