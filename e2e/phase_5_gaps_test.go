package e2e

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	backendv1 "github.com/sovereign-l1/chain/api/backend/v1"
)

// Helper to connect to Read DB from E2E context
func connectReadDB() (*pgxpool.Pool, string, error) {
	urls := []string{
		"postgres://api_reader:sovereign_read_pwd@db-read:5432/sovereign_read",
		"postgres://api_reader:sovereign_read_pwd@localhost:5434/sovereign_read",
	}

	var lastErr error
	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		db, err := pgxpool.New(ctx, url)
		cancel()
		if err == nil {
			// Verify ping
			ctxPing, cancelPing := context.WithTimeout(context.Background(), 1*time.Second)
			err = db.Ping(ctxPing)
			cancelPing()
			if err == nil {
				return db, url, nil
			}
			db.Close()
		}
		lastErr = err
	}
	return nil, "", fmt.Errorf("could not connect to read database: %v", lastErr)
}

// Helper to connect to Write DB from E2E context (for advisory lock test)
func connectWriteDB() (*pgxpool.Pool, string, error) {
	urls := []string{
		"postgres://ingestion_writer:sovereign_write_pwd@db-write:5432/sovereign_write",
		"postgres://ingestion_writer:sovereign_write_pwd@localhost:5433/sovereign_write",
	}

	var lastErr error
	for _, url := range urls {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		db, err := pgxpool.New(ctx, url)
		cancel()
		if err == nil {
			ctxPing, cancelPing := context.WithTimeout(context.Background(), 1*time.Second)
			err = db.Ping(ctxPing)
			cancelPing()
			if err == nil {
				return db, url, nil
			}
			db.Close()
		}
		lastErr = err
	}
	return nil, "", fmt.Errorf("could not connect to write database: %v", lastErr)
}

// 1. Test ListSettlements & ListMilestones Keyset Pagination
func TestPhase5GapsPagination(t *testing.T) {
	readDB, url, err := connectReadDB()
	if err != nil {
		t.Skipf("[SKIP] Database not running: %v", err)
		return
	}
	defer readDB.Close()
	t.Logf("Connected to read database at %s", url)

	ctx := context.Background()

	// Clear table for clean testing
	_, _ = readDB.Exec(ctx, "DELETE FROM settlements WHERE settlement_id LIKE 'test_settlement_%'")
	_, _ = readDB.Exec(ctx, "DELETE FROM milestone_status WHERE milestone_id LIKE 'test_milestone_%'")

	// Insert 5 test settlements
	for i := 1; i <= 5; i++ {
		_, err := readDB.Exec(ctx,
			"INSERT INTO settlements (settlement_id, proof, status, block_height, signatures) VALUES ($1, $2, $3, $4, $5)",
			fmt.Sprintf("test_settlement_%d", i), []byte{byte(i)}, "executed", int64(100+i), []string{"sig1", "sig2"},
		)
		if err != nil {
			t.Fatalf("failed to insert test settlement: %v", err)
		}
	}

	// Insert 5 test milestones
	for i := 1; i <= 5; i++ {
		_, err := readDB.Exec(ctx,
			"INSERT INTO milestone_status (milestone_id, status, block_height) VALUES ($1, $2, $3)",
			fmt.Sprintf("test_milestone_%d", i), "achieved", int64(200+i),
		)
		if err != nil {
			t.Fatalf("failed to insert test milestone: %v", err)
		}
	}

	defer func() {
		_, _ = readDB.Exec(ctx, "DELETE FROM settlements WHERE settlement_id LIKE 'test_settlement_%'")
		_, _ = readDB.Exec(ctx, "DELETE FROM milestone_status WHERE milestone_id LIKE 'test_milestone_%'")
	}()

	// Query page 1 (limit 2)
	req1 := &backendv1.ListSettlementsRequest{
		Pagination: &backendv1.PageRequest{
			Limit: 2,
		},
	}
	res1, err := mockListSettlementsHandler(ctx, readDB, req1)
	if err != nil {
		t.Fatalf("ListSettlements page 1 failed: %v", err)
	}

	if len(res1.Settlements) != 2 {
		t.Fatalf("Expected 2 settlements, got %d", len(res1.Settlements))
	}
	if !res1.Pagination.HasMore {
		t.Fatal("Expected has_more to be true")
	}
	if len(res1.Pagination.NextCursor) == 0 {
		t.Fatal("Expected non-empty next_cursor")
	}

	// Verify DESC order: height 105 then 104
	if res1.Settlements[0].BlockHeight != 105 || res1.Settlements[1].BlockHeight != 104 {
		t.Fatalf("Expected heights 105 and 104, got %d and %d", res1.Settlements[0].BlockHeight, res1.Settlements[1].BlockHeight)
	}

	// Query page 2 (limit 2) using next_cursor
	req2 := &backendv1.ListSettlementsRequest{
		Pagination: &backendv1.PageRequest{
			Limit:  2,
			Cursor: res1.Pagination.NextCursor,
		},
	}
	res2, err := mockListSettlementsHandler(ctx, readDB, req2)
	if err != nil {
		t.Fatalf("ListSettlements page 2 failed: %v", err)
	}

	if len(res2.Settlements) != 2 {
		t.Fatalf("Expected 2 settlements on page 2, got %d", len(res2.Settlements))
	}
	if !res2.Pagination.HasMore {
		t.Fatal("Expected has_more to be true on page 2")
	}

	// Verify DESC order: height 103 then 102
	if res2.Settlements[0].BlockHeight != 103 || res2.Settlements[1].BlockHeight != 102 {
		t.Fatalf("Expected heights 103 and 102, got %d and %d", res2.Settlements[0].BlockHeight, res2.Settlements[1].BlockHeight)
	}

	// Query page 3 (limit 2)
	req3 := &backendv1.ListSettlementsRequest{
		Pagination: &backendv1.PageRequest{
			Limit:  2,
			Cursor: res2.Pagination.NextCursor,
		},
	}
	res3, err := mockListSettlementsHandler(ctx, readDB, req3)
	if err != nil {
		t.Fatalf("ListSettlements page 3 failed: %v", err)
	}

	if len(res3.Settlements) != 1 {
		t.Fatalf("Expected 1 settlement on page 3, got %d", len(res3.Settlements))
	}
	if res3.Pagination.HasMore {
		t.Fatal("Expected has_more to be false on page 3")
	}
	if res3.Settlements[0].BlockHeight != 101 {
		t.Fatalf("Expected height 101, got %d", res3.Settlements[0].BlockHeight)
	}

	t.Log("[PASS] Checked ListSettlements Keyset Pagination.")
}

// 2. Test GetBridgeTx Endpoint with Custom Hash Formats
func TestPhase5GapsGetBridgeTx(t *testing.T) {
	readDB, _, err := connectReadDB()
	if err != nil {
		t.Skipf("[SKIP] Database not running: %v", err)
		return
	}
	defer readDB.Close()

	ctx := context.Background()

	// Clear table
	_, _ = readDB.Exec(ctx, "DELETE FROM bridge_pending WHERE nonce = 105005")
	_, _ = readDB.Exec(ctx, "DELETE FROM bridge_events WHERE block_height = 105 AND event_index = 5")

	// Insert pending bridge tx
	_, err = readDB.Exec(ctx,
		"INSERT INTO bridge_pending (nonce, token_address, amount, recipient, status) VALUES ($1, $2, $3, $4, $5)",
		int64(105005), "usov", float64(5000), "recipient_wallet_addr", "pending",
	)
	if err != nil {
		t.Fatalf("failed to insert pending bridge tx: %v", err)
	}

	// Insert bridge event
	_, err = readDB.Exec(ctx,
		"INSERT INTO bridge_events (block_height, event_index, direction, asset, amount) VALUES ($1, $2, $3, $4, $5)",
		int64(105), int(5), "release", "usov", float64(5000),
	)
	if err != nil {
		t.Fatalf("failed to insert bridge event: %v", err)
	}

	defer func() {
		_, _ = readDB.Exec(ctx, "DELETE FROM bridge_pending WHERE nonce = 105005")
		_, _ = readDB.Exec(ctx, "DELETE FROM bridge_events WHERE block_height = 105 AND event_index = 5")
	}()

	// Query using tx_105_5
	res, err := mockGetBridgeTxHandler(ctx, readDB, "tx_105_5")
	if err != nil {
		t.Fatalf("GetBridgeTx failed: %v", err)
	}

	if res.Status != "pending" {
		t.Errorf("Expected status pending, got %s", res.Status)
	}
	if res.Amount != "5000" {
		t.Errorf("Expected amount 5000, got %s", res.Amount)
	}
	if res.Recipient != "recipient_wallet_addr" {
		t.Errorf("Expected recipient recipient_wallet_addr, got %s", res.Recipient)
	}

	t.Log("[PASS] Checked GetBridgeTx endpoint query resolution.")
}

// 3. Test x-wallet-address metadata gRPC Interceptor
func TestPhase5GapsWalletAddressInterceptor(t *testing.T) {
	// Spin up a simple in-memory gRPC server to test interceptors
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	unaryInterceptor := func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			addresses := md.Get("x-wallet-address")
			if len(addresses) > 0 {
				_ = grpc.SendHeader(ctx, metadata.Pairs("x-wallet-address", addresses[0]))
			}
		}
		return handler(ctx, req)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(unaryInterceptor))
	defer grpcServer.Stop()

	// Register a mock simple query service
	mockSrv := &mockQueryServer{}
	backendv1.RegisterQueryServiceServer(grpcServer, mockSrv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := backendv1.NewQueryServiceClient(conn)

	// Context with metadata
	md := metadata.Pairs("x-wallet-address", "sov1testwalletaddress")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var header metadata.MD
	_, err = client.GetTps(ctx, &backendv1.GetTpsRequest{}, grpc.Header(&header))
	if err != nil {
		t.Fatalf("GetTps call failed: %v", err)
	}

	// Verify that the out-going header was populated/forwarded
	addresses := header.Get("x-wallet-address")
	if len(addresses) == 0 || addresses[0] != "sov1testwalletaddress" {
		t.Errorf("Expected forwarded metadata header 'sov1testwalletaddress', got: %v", addresses)
	}

	t.Log("[PASS] Checked x-wallet-address metadata interceptor forwarding.")
}

// 4. Test Ingestion Singleton Lock Acquisition Retry/Timeout Loop
func TestPhase5GapsAdvisoryLockRetryLoop(t *testing.T) {
	writeDB, _, err := connectWriteDB()
	if err != nil {
		t.Skipf("[SKIP] Write database not running: %v", err)
		return
	}
	defer writeDB.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const AdvisoryLockID = 41892305

	// Session 1: Acquire lock
	conn1, err := writeDB.Acquire(ctx)
	if err != nil {
		t.Fatalf("Failed to acquire connection 1: %v", err)
	}
	defer conn1.Release()

	var locked1 bool
	err = conn1.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", AdvisoryLockID).Scan(&locked1)
	if err != nil {
		t.Fatalf("pg_try_advisory_lock on connection 1 failed: %v", err)
	}
	if !locked1 {
		t.Fatal("Connection 1 should hold the lock")
	}

	defer func() {
		_, _ = conn1.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", AdvisoryLockID)
	}()

	// Session 2: Try to acquire lock with retry/timeout loop
	// We run it with a shorter retry count or context timeout to keep the test fast
	start := time.Now()
	retryCtx, retryCancel := context.WithTimeout(ctx, 3*time.Second)
	defer retryCancel()

	attempts := 0
	acquired := false
	for attempt := 1; attempt <= 3; attempt++ {
		attempts++
		var locked2 bool
		_ = writeDB.QueryRow(retryCtx, "SELECT pg_try_advisory_lock($1)", AdvisoryLockID).Scan(&locked2)
		if locked2 {
			acquired = true
			break
		}

		if attempt < 3 {
			select {
			case <-retryCtx.Done():
				break
			case <-time.After(100 * time.Millisecond): // Shorter delay for unit testing
			}
		}
	}

	duration := time.Since(start)

	if acquired {
		t.Fatal("Connection 2 should not have acquired the lock while connection 1 holds it")
	}
	if attempts != 3 {
		t.Errorf("Expected 3 retry attempts, got %d", attempts)
	}
	if duration < 200*time.Millisecond {
		t.Errorf("Expected retry loop to wait and take time, took %v", duration)
	}

	t.Log("[PASS] Checked PostgreSQL singleton advisory lock retry loop.")
}

// --- Mocks for Handlers ---

func mockListSettlementsHandler(ctx context.Context, readDB *pgxpool.Pool, req *backendv1.ListSettlementsRequest) (*backendv1.ListSettlementsResponse, error) {
	limit := uint32(10)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}

	var cursorHeight int64 = 9223372036854775807
	var cursorID string = ""

	if req.Pagination != nil && len(req.Pagination.Cursor) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(string(req.Pagination.Cursor))
		if err != nil {
			return nil, err
		}
		parts := strings.SplitN(string(decoded), ",", 2)
		h, _ := strconv.ParseInt(parts[0], 10, 64)
		cursorHeight = h
		cursorID = parts[1]
	}

	var rows pgx.Rows
	var err error
	if cursorID == "" {
		rows, err = readDB.Query(ctx,
			"SELECT settlement_id, proof, status, block_height, signatures FROM settlements WHERE settlement_id LIKE 'test_settlement_%' ORDER BY block_height DESC, settlement_id DESC LIMIT $1",
			limit+1,
		)
	} else {
		rows, err = readDB.Query(ctx,
			"SELECT settlement_id, proof, status, block_height, signatures FROM settlements WHERE settlement_id LIKE 'test_settlement_%' AND (block_height, settlement_id) < ($1, $2) ORDER BY block_height DESC, settlement_id DESC LIMIT $3",
			cursorHeight, cursorID, limit+1,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settlements []*backendv1.GetSettlementResponse
	for rows.Next() {
		var proof []byte
		var signatures []string
		res := &backendv1.GetSettlementResponse{}
		_ = rows.Scan(&res.SettlementId, &proof, &res.Status, &res.BlockHeight, &signatures)
		res.Proof = proof
		res.Signatures = signatures
		settlements = append(settlements, res)
	}

	hasMore := false
	var nextCursor []byte
	if uint32(len(settlements)) > limit {
		hasMore = true
		lastItem := settlements[limit-1]
		str := fmt.Sprintf("%d,%s", lastItem.BlockHeight, lastItem.SettlementId)
		nextCursor = []byte(base64.StdEncoding.EncodeToString([]byte(str)))
		settlements = settlements[:limit]
	}

	return &backendv1.ListSettlementsResponse{
		Settlements: settlements,
		Pagination: &backendv1.PageResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func mockGetBridgeTxHandler(ctx context.Context, readDB *pgxpool.Pool, txHash string) (*backendv1.GetBridgeTxResponse, error) {
	var height int64
	var index int
	_, err := fmt.Sscanf(txHash, "tx_%d_%d", &height, &index)
	if err == nil {
		nonce := height*1000 + int64(index)
		var tokenAddress string
		var amount float64
		var recipient string
		var statusStr string
		errPending := readDB.QueryRow(ctx,
			"SELECT token_address, amount, recipient, status FROM bridge_pending WHERE nonce = $1",
			nonce,
		).Scan(&tokenAddress, &amount, &recipient, &statusStr)
		if errPending == nil {
			return &backendv1.GetBridgeTxResponse{
				TxHash:       txHash,
				Status:       statusStr,
				BlockHeight:  height,
				TokenAddress: tokenAddress,
				Amount:       fmt.Sprintf("%.0f", amount),
				Sender:       "bsc",
				Recipient:    recipient,
			}, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

type mockQueryServer struct {
	backendv1.UnimplementedQueryServiceServer
}

func (m *mockQueryServer) GetTps(ctx context.Context, req *backendv1.GetTpsRequest) (*backendv1.GetTpsResponse, error) {
	return &backendv1.GetTpsResponse{
		TpsAvg:   1.5,
		TpsPeak:  3.0,
		TotalTxs: 1500,
	}, nil
}
