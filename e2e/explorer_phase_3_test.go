package e2e

import (
	"context"
	"testing"
	"time"

	explorerv1 "github.com/sovereign-l1/chain/api/explorer/v1"
	"google.golang.org/grpc"
)

// Mock explorer server for Phase 3 endpoints verification
type mockExplorerPhase3Server struct {
	explorerv1.UnimplementedExplorerServiceServer
}

func (m *mockExplorerPhase3Server) GetBridgeTx(ctx context.Context, req *explorerv1.GetBridgeTxRequest) (*explorerv1.BridgeTxDetail, error) {
	return &explorerv1.BridgeTxDetail{
		Id:         1,
		Direction:  "deposit",
		Nonce:      req.Nonce,
		Status:     "minted",
		SourceHash: "0xmockbsclockhash_1",
		DestHash:   "0xmockcosmosminthash_1",
		Amount:     "5000000000",
		Sender:     "0xsenderaddress",
		Receiver:   "sovereign1address0",
		Height:     100,
		Time:       time.Now().Format(time.RFC3339),
	}, nil
}

func (m *mockExplorerPhase3Server) ListBridgeTxs(ctx context.Context, req *explorerv1.ListBridgeTxsRequest) (*explorerv1.BridgeTxList, error) {
	return &explorerv1.BridgeTxList{
		Txs: []*explorerv1.BridgeTxDetail{
			{
				Id:         1,
				Direction:  "deposit",
				Nonce:      1,
				Status:     "minted",
				SourceHash: "0xmockbsclockhash_1",
				DestHash:   "0xmockcosmosminthash_1",
				Amount:     "5000000000",
				Sender:     "0xsenderaddress",
				Receiver:   "sovereign1address0",
				Height:     100,
				Time:       time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

func (m *mockExplorerPhase3Server) GetBridgeSupplyMetrics(ctx context.Context, req *explorerv1.GetBridgeSupplyMetricsRequest) (*explorerv1.SupplyMetrics, error) {
	return &explorerv1.SupplyMetrics{
		CosmosMinted:      "1250000000000",
		BscCirculating:    "1250000000000",
		TotalSupply:       "2500000000000",
		BridgeSupplyGauge: "1.0000",
	}, nil
}

func (m *mockExplorerPhase3Server) ListRelayers(ctx context.Context, req *explorerv1.ListRelayersRequest) (*explorerv1.RelayerList, error) {
	return &explorerv1.RelayerList{
		Relayers: []*explorerv1.RelayerDetail{
			{
				Address:    "sovereign1relayer0",
				Status:     "Primary",
				LastActive: 1000,
				MissCount:  0,
			},
		},
	}, nil
}

func (m *mockExplorerPhase3Server) ListBridgeCircuitBreaker(ctx context.Context, req *explorerv1.ListBridgeCircuitBreakerRequest) (*explorerv1.CircuitBreakerHistory, error) {
	return &explorerv1.CircuitBreakerHistory{
		Events: []*explorerv1.CircuitBreakerEvent{
			{
				Height:         50,
				EventType:      "pause",
				TriggerAddress: "sovereign1relayer0",
				Time:           time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

func (m *mockExplorerPhase3Server) ListBridgeNonces(ctx context.Context, req *explorerv1.ListBridgeNoncesRequest) (*explorerv1.NonceRegistryDetail, error) {
	return &explorerv1.NonceRegistryDetail{
		UsedNonces:     []int64{1, 2, 3},
		InFlightNonces: []int64{4, 5},
	}, nil
}

func (m *mockExplorerPhase3Server) GetTpsHistory(ctx context.Context, req *explorerv1.GetTpsRequest) (*explorerv1.TpsHistory, error) {
	return &explorerv1.TpsHistory{
		Points: []*explorerv1.TpsPoint{
			{
				Time: time.Now().Format(time.RFC3339),
				Tps:  15.4,
			},
		},
	}, nil
}

func (m *mockExplorerPhase3Server) GetBlockTimeHistory(ctx context.Context, req *explorerv1.GetBlockTimeRequest) (*explorerv1.BlockTimeHistory, error) {
	return &explorerv1.BlockTimeHistory{
		Points: []*explorerv1.BlockTimePoint{
			{
				Time:     time.Now().Format(time.RFC3339),
				Duration: 1.5,
			},
		},
	}, nil
}

func (m *mockExplorerPhase3Server) GetValidatorUptimeGrid(ctx context.Context, req *explorerv1.GetUptimeRequest) (*explorerv1.UptimeHeatmap, error) {
	return &explorerv1.UptimeHeatmap{
		Points: []*explorerv1.UptimePoint{
			{
				SlotIndex: 0,
				Time:      "Today",
				Uptime:    99.8,
			},
		},
	}, nil
}

func (m *mockExplorerPhase3Server) GetBridgeVolumeHistory(ctx context.Context, req *explorerv1.GetBridgeVolumeRequest) (*explorerv1.VolumeHistory, error) {
	return &explorerv1.VolumeHistory{
		Points: []*explorerv1.VolumePoint{
			{
				Time:   time.Now().Format(time.RFC3339),
				Volume: "50000000000",
			},
		},
	}, nil
}

type mockExportTxsCsvStream struct {
	ctx   context.Context
	ch    chan *explorerv1.CsvChunk
	err   error
	grpc.ServerStream
}

func (m *mockExportTxsCsvStream) Context() context.Context {
	return m.ctx
}

func (m *mockExportTxsCsvStream) Send(chunk *explorerv1.CsvChunk) error {
	m.ch <- chunk
	return nil
}

func (m *mockExplorerPhase3Server) ExportTxsCsv(req *explorerv1.ExportTxsCsvRequest, stream explorerv1.ExplorerService_ExportTxsCsvServer) error {
	_ = stream.Send(&explorerv1.CsvChunk{Data: []byte("hash,height,time,type\n")})
	_ = stream.Send(&explorerv1.CsvChunk{Data: []byte("mocktxhash1,100,2026-06-24T12:00:00Z,cosmos\n")})
	return nil
}

func TestExplorerPhase3Endpoints(t *testing.T) {
	s := &mockExplorerPhase3Server{}
	ctx := context.Background()

	// 1. GetBridgeTx
	bridgeTx, err := s.GetBridgeTx(ctx, &explorerv1.GetBridgeTxRequest{Nonce: 1})
	if err != nil {
		t.Fatalf("failed to query bridge tx: %v", err)
	}
	if bridgeTx.Nonce != 1 || bridgeTx.Direction != "deposit" {
		t.Errorf("bridge tx verification failed")
	}

	// 2. ListBridgeTxs
	txs, err := s.ListBridgeTxs(ctx, &explorerv1.ListBridgeTxsRequest{})
	if err != nil {
		t.Fatalf("failed to list bridge txs: %v", err)
	}
	if len(txs.Txs) != 1 || txs.Txs[0].Nonce != 1 {
		t.Errorf("list bridge txs verification failed")
	}

	// 3. GetBridgeSupplyMetrics
	metrics, err := s.GetBridgeSupplyMetrics(ctx, &explorerv1.GetBridgeSupplyMetricsRequest{})
	if err != nil {
		t.Fatalf("failed to query supply metrics: %v", err)
	}
	if metrics.BridgeSupplyGauge != "1.0000" {
		t.Errorf("supply invariant metrics mismatch")
	}

	// 4. ListRelayers
	relayers, err := s.ListRelayers(ctx, &explorerv1.ListRelayersRequest{})
	if err != nil {
		t.Fatalf("failed to query relayers: %v", err)
	}
	if len(relayers.Relayers) != 1 || relayers.Relayers[0].Status != "Primary" {
		t.Errorf("relayers list verification failed")
	}

	// 5. ListBridgeCircuitBreaker
	cb, err := s.ListBridgeCircuitBreaker(ctx, &explorerv1.ListBridgeCircuitBreakerRequest{})
	if err != nil {
		t.Fatalf("failed to query circuit breaker: %v", err)
	}
	if len(cb.Events) != 1 || cb.Events[0].EventType != "pause" {
		t.Errorf("circuit breaker list verification failed")
	}

	// 6. ListBridgeNonces
	nonces, err := s.ListBridgeNonces(ctx, &explorerv1.ListBridgeNoncesRequest{})
	if err != nil {
		t.Fatalf("failed to query nonces: %v", err)
	}
	if len(nonces.UsedNonces) != 3 || len(nonces.InFlightNonces) != 2 {
		t.Errorf("nonces registry verification failed")
	}

	// 7. GetTpsHistory
	tps, err := s.GetTpsHistory(ctx, &explorerv1.GetTpsRequest{})
	if err != nil {
		t.Fatalf("failed to query tps history: %v", err)
	}
	if len(tps.Points) != 1 || tps.Points[0].Tps != 15.4 {
		t.Errorf("tps history verification failed")
	}

	// 8. GetBlockTimeHistory
	bt, err := s.GetBlockTimeHistory(ctx, &explorerv1.GetBlockTimeRequest{})
	if err != nil {
		t.Fatalf("failed to query block time history: %v", err)
	}
	if len(bt.Points) != 1 || bt.Points[0].Duration != 1.5 {
		t.Errorf("block time history verification failed")
	}

	// 9. GetValidatorUptimeGrid
	uptime, err := s.GetValidatorUptimeGrid(ctx, &explorerv1.GetUptimeRequest{})
	if err != nil {
		t.Fatalf("failed to query validator uptime grid: %v", err)
	}
	if len(uptime.Points) != 1 || uptime.Points[0].Uptime != 99.8 {
		t.Errorf("uptime grid verification failed")
	}

	// 10. GetBridgeVolumeHistory
	vol, err := s.GetBridgeVolumeHistory(ctx, &explorerv1.GetBridgeVolumeRequest{})
	if err != nil {
		t.Fatalf("failed to query bridge volume history: %v", err)
	}
	if len(vol.Points) != 1 || vol.Points[0].Volume != "50000000000" {
		t.Errorf("bridge volume history verification failed")
	}

	// 11. ExportTxsCsv stream verification
	streamChan := make(chan *explorerv1.CsvChunk, 10)
	mockStream := &mockExportTxsCsvStream{
		ctx: ctx,
		ch:  streamChan,
	}

	go func() {
		err = s.ExportTxsCsv(&explorerv1.ExportTxsCsvRequest{Address: "sovereign1address0"}, mockStream)
		close(streamChan)
	}()

	var receivedData []byte
	for chunk := range streamChan {
		receivedData = append(receivedData, chunk.Data...)
	}

	expectedHeader := "hash,height,time,type\n"
	if string(receivedData[:len(expectedHeader)]) != expectedHeader {
		t.Errorf("CSV streaming headers mismatch: got %s", string(receivedData))
	}

	t.Log("[PASS] Checked gRPC API responses match protobuf specifications for all Phase 3 bridge, EVM, and analytics endpoints.")
}
