package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	backendv1 "github.com/sovereign-l1/chain/api/backend/v1"
	relayerv1 "github.com/sovereign-l1/chain/api/relayer/v1"
)

const (
	NatsStreamSubject = "account:stream"
)

type Config struct {
	ReadDBURL   string
	NatsURL     string
	GrpcPort    string
	RestPort    string
}

type EventRecord struct {
	BlockHeight int64           `json:"block_height"`
	EventIndex  int             `json:"event_index"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}

type server struct {
	backendv1.UnimplementedQueryServiceServer
	backendv1.UnimplementedStreamServiceServer
	relayerv1.UnimplementedRelayerServiceServer
	readDB *pgxpool.Pool
	nc     *nats.Conn
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.ReadDBURL, "read-db-url", os.Getenv("READ_DB_URL"), "Read DB URL")
	flag.StringVar(&cfg.NatsURL, "nats-url", os.Getenv("NATS_URL"), "NATS URL")
	flag.StringVar(&cfg.GrpcPort, "grpc-port", "9090", "gRPC port")
	flag.StringVar(&cfg.RestPort, "rest-port", "8081", "REST gateway port")
	flag.Parse()

	if cfg.ReadDBURL == "" {
		cfg.ReadDBURL = "postgres://api_reader:sovereign_read_pwd@db-read:5432/sovereign_read"
	}
	if cfg.NatsURL == "" {
		cfg.NatsURL = nats.DefaultURL
	}

	log.Printf("Starting API Gateway Server...")
	log.Printf("Read DB URL: %s", cfg.ReadDBURL)
	log.Printf("NATS URL: %s", cfg.NatsURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Read DB
	readDB, err := pgxpool.New(ctx, cfg.ReadDBURL)
	if err != nil {
		log.Fatalf("failed to connect to Read DB: %v", err)
	}
	defer readDB.Close()

	nkeyOpt, err := getNatsNkeyOption("stream")
	if err != nil {
		log.Fatalf("failed to configure NATS NKey: %v", err)
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nkeyOpt,
	)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	s := &server{
		readDB: readDB,
		nc:     nc,
	}

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
				log.Printf("[Interceptor] Unary Request: %s, wallet address: %s", info.FullMethod, addresses[0])
				ctx = metadata.AppendToOutgoingContext(ctx, "x-wallet-address", addresses[0])
			}
		}
		return handler(ctx, req)
	}

	streamInterceptor := func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if ok {
			addresses := md.Get("x-wallet-address")
			if len(addresses) > 0 {
				log.Printf("[Interceptor] Stream Request: %s, wallet address: %s", info.FullMethod, addresses[0])
			}
		}
		return handler(srv, ss)
	}

	// 1. Start gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GrpcPort)
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", cfg.GrpcPort, err)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)
	backendv1.RegisterQueryServiceServer(grpcServer, s)
	backendv1.RegisterStreamServiceServer(grpcServer, s)
	relayerv1.RegisterRelayerServiceServer(grpcServer, s)

	go func() {
		log.Printf("Serving gRPC on port %s", cfg.GrpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// 2. Start grpc-gateway REST proxy sidecar
	go func() {
		// Wait a brief moment for gRPC to boot
		time.Sleep(100 * time.Millisecond)

		mux := runtime.NewServeMux(
			runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
				if strings.ToLower(key) == "x-wallet-address" {
					return "x-wallet-address", true
				}
				return runtime.DefaultHeaderMatcher(key)
			}),
		)
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

		endpoint := "localhost:" + cfg.GrpcPort
		err := backendv1.RegisterQueryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
		if err != nil {
			log.Fatalf("failed to register query REST handler: %v", err)
		}
		err = backendv1.RegisterStreamServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
		if err != nil {
			log.Fatalf("failed to register stream REST handler: %v", err)
		}
		err = relayerv1.RegisterRelayerServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
		if err != nil {
			log.Fatalf("failed to register relayer REST handler: %v", err)
		}

		log.Printf("Serving REST Gateway on port %s", cfg.RestPort)
		if err := http.ListenAndServe(":"+cfg.RestPort, mux); err != nil {
			log.Fatalf("REST gateway server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Gracefully stopping API Gateway...")
	grpcServer.GracefulStop()
}

// --- QueryService Handlers ---

func (s *server) GetBridgeVolume(ctx context.Context, req *backendv1.GetBridgeVolumeRequest) (*backendv1.GetBridgeVolumeResponse, error) {
	timeframe := req.Timeframe
	if timeframe == "" {
		timeframe = "all"
	}

	var totalMinted string
	var totalBurned string
	var txCount int64

	err := s.readDB.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_minted), 0)::text, COALESCE(SUM(total_burned), 0)::text, COALESCE(SUM(transaction_count), 0)
		 FROM bridge_volume
		 WHERE (token_address = $1 OR $1 = '') AND (chain_id = $2 OR $2 = '') AND timeframe = $3`,
		req.TokenAddress, req.ChainId, timeframe,
	).Scan(&totalMinted, &totalBurned, &txCount)

	if err != nil {
		return nil, fmt.Errorf("failed to query bridge volume: %w", err)
	}

	return &backendv1.GetBridgeVolumeResponse{
		TotalMinted:      totalMinted,
		TotalBurned:      totalBurned,
		VolumeUsd:        "0.0", // Mock/Stub value for USD volume
		TransactionCount: txCount,
	}, nil
}

func (s *server) GetValidatorUptime(ctx context.Context, req *backendv1.GetValidatorUptimeRequest) (*backendv1.GetValidatorUptimeResponse, error) {
	var totalBlocks int64
	var missedBlocks int64
	var uptime float64

	err := s.readDB.QueryRow(ctx,
		"SELECT total_blocks, missed_blocks, uptime_percentage FROM validator_uptime WHERE validator_address = $1",
		req.ValidatorAddress,
	).Scan(&totalBlocks, &missedBlocks, &uptime)

	if err != nil {
		if err == sql.ErrNoRows {
			return &backendv1.GetValidatorUptimeResponse{
				ValidatorAddress: req.ValidatorAddress,
				TotalBlocks:      0,
				MissedBlocks:     0,
				UptimePercentage: 100.0,
			}, nil
		}
		// Try using pgx direct error check
		return nil, fmt.Errorf("failed to query validator uptime: %w", err)
	}

	return &backendv1.GetValidatorUptimeResponse{
		ValidatorAddress: req.ValidatorAddress,
		TotalBlocks:      totalBlocks,
		MissedBlocks:     missedBlocks,
		UptimePercentage: uptime,
	}, nil
}

func (s *server) GetOracleParticipation(ctx context.Context, req *backendv1.GetOracleParticipationRequest) (*backendv1.GetOracleParticipationResponse, error) {
	var totalRequests int64
	var successfulReveals int64
	var participationRate float64

	err := s.readDB.QueryRow(ctx,
		"SELECT total_requests, successful_reveals, participation_rate FROM oracle_participation WHERE oracle_address = $1",
		req.OracleAddress,
	).Scan(&totalRequests, &successfulReveals, &participationRate)

	if err != nil {
		return &backendv1.GetOracleParticipationResponse{
			OracleAddress:     req.OracleAddress,
			TotalRequests:     0,
			SuccessfulReveals: 0,
			ParticipationRate: 100.0,
		}, nil
	}

	return &backendv1.GetOracleParticipationResponse{
		OracleAddress:     req.OracleAddress,
		TotalRequests:     totalRequests,
		SuccessfulReveals: successfulReveals,
		ParticipationRate: participationRate,
	}, nil
}

func (s *server) GetSettlement(ctx context.Context, req *backendv1.GetSettlementRequest) (*backendv1.GetSettlementResponse, error) {
	var proof []byte
	var status string
	var blockHeight int64
	var signatures []string

	err := s.readDB.QueryRow(ctx,
		"SELECT proof, status, block_height, signatures FROM settlements WHERE settlement_id = $1",
		req.SettlementId,
	).Scan(&proof, &status, &blockHeight, &signatures)

	if err != nil {
		return nil, fmt.Errorf("settlement not found: %w", err)
	}

	return &backendv1.GetSettlementResponse{
		SettlementId: req.SettlementId,
		Proof:        proof,
		Status:       status,
		BlockHeight:  blockHeight,
		Signatures:   signatures,
	}, nil
}

func (s *server) GetMilestoneStatus(ctx context.Context, req *backendv1.GetMilestoneStatusRequest) (*backendv1.GetMilestoneStatusResponse, error) {
	var status string
	var blockHeight int64

	err := s.readDB.QueryRow(ctx,
		"SELECT status, block_height FROM milestone_status WHERE milestone_id = $1",
		req.MilestoneId,
	).Scan(&status, &blockHeight)

	if err != nil {
		return &backendv1.GetMilestoneStatusResponse{
			MilestoneId: req.MilestoneId,
			Status:      "unknown",
			BlockHeight: 0,
		}, nil
	}

	return &backendv1.GetMilestoneStatusResponse{
		MilestoneId: req.MilestoneId,
		Status:      status,
		BlockHeight: blockHeight,
	}, nil
}

func (s *server) GetBridgePending(ctx context.Context, req *backendv1.GetBridgePendingRequest) (*backendv1.GetBridgePendingResponse, error) {
	var tokenAddress string
	var amount float64
	var recipient string
	var status string

	err := s.readDB.QueryRow(ctx,
		"SELECT token_address, amount, recipient, status FROM bridge_pending WHERE nonce = $1",
		req.Nonce,
	).Scan(&tokenAddress, &amount, &recipient, &status)

	if err != nil {
		return nil, fmt.Errorf("bridge transfer not found for nonce %d: %w", req.Nonce, err)
	}

	return &backendv1.GetBridgePendingResponse{
		Nonce:        req.Nonce,
		TokenAddress: tokenAddress,
		Amount:       fmt.Sprintf("%.0f", amount),
		Recipient:    recipient,
		Status:       status,
	}, nil
}

// --- StreamService Handlers ---

func (s *server) StreamBridgeEvents(req *backendv1.StreamBridgeEventsRequest, srv backendv1.StreamService_StreamBridgeEventsServer) error {
	clientChan := make(chan *nats.Msg, 64)
	errChan := make(chan error, 1)

	sub, err := s.nc.Subscribe(NatsStreamSubject, func(msg *nats.Msg) {
		select {
		case clientChan <- msg:
		default:
			select {
			case errChan <- status.Error(codes.ResourceExhausted, "slow consumer channel buffer full"):
			default:
			}
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-srv.Context().Done():
			return nil
		case err := <-errChan:
			return err
		case msg := <-clientChan:
			var ev EventRecord
			if err := json.Unmarshal(msg.Data, &ev); err != nil {
				continue
			}

			if ev.EventType == "MsgBridgeIn" || ev.EventType == "MsgBridgeOut" {
				var attrs map[string]string
				_ = json.Unmarshal(ev.Payload, &attrs)

				token := attrs["receiver"]
				if token == "" {
					token = "usov"
				}
				if req.TokenAddress != "" && req.TokenAddress != token {
					continue
				}

				amount := attrs["amount"]
				sender := attrs["sender"]
				if sender == "" {
					sender = "bsc"
				}
				recipient := attrs["receiver"]
				if recipient == "" {
					recipient = attrs["bsc_recipient"]
				}

				err := srv.Send(&backendv1.BridgeEvent{
					EventType:    ev.EventType,
					TokenAddress: token,
					Amount:       amount,
					Sender:       sender,
					Recipient:    recipient,
					BlockHeight:  ev.BlockHeight,
					TxHash:       fmt.Sprintf("tx_%d_%d", ev.BlockHeight, ev.EventIndex),
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func (s *server) StreamOracleEvents(req *backendv1.StreamOracleEventsRequest, srv backendv1.StreamService_StreamOracleEventsServer) error {
	clientChan := make(chan *nats.Msg, 64)
	errChan := make(chan error, 1)

	sub, err := s.nc.Subscribe(NatsStreamSubject, func(msg *nats.Msg) {
		select {
		case clientChan <- msg:
		default:
			select {
			case errChan <- status.Error(codes.ResourceExhausted, "slow consumer channel buffer full"):
			default:
			}
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-srv.Context().Done():
			return nil
		case err := <-errChan:
			return err
		case msg := <-clientChan:
			var ev EventRecord
			if err := json.Unmarshal(msg.Data, &ev); err != nil {
				continue
			}

			if ev.EventType == "oracle_commit" || ev.EventType == "oracle_reveal" {
				var attrs map[string]string
				_ = json.Unmarshal(ev.Payload, &attrs)

				operator := attrs["operator"]
				if req.OracleAddress != "" && req.OracleAddress != operator {
					continue
				}

				err := srv.Send(&backendv1.OracleEvent{
					OracleAddress: operator,
					RequestId:     fmt.Sprintf("feed_%s_round_%s", attrs["feed_id"], attrs["round_id"]),
					DataType:      "price",
					Payload:       attrs["value"],
					BlockHeight:   ev.BlockHeight,
					Status:        ev.EventType,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func (s *server) StreamMilestoneEvents(req *backendv1.StreamMilestoneEventsRequest, srv backendv1.StreamService_StreamMilestoneEventsServer) error {
	clientChan := make(chan *nats.Msg, 64)
	errChan := make(chan error, 1)

	sub, err := s.nc.Subscribe(NatsStreamSubject, func(msg *nats.Msg) {
		select {
		case clientChan <- msg:
		default:
			select {
			case errChan <- status.Error(codes.ResourceExhausted, "slow consumer channel buffer full"):
			default:
			}
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-srv.Context().Done():
			return nil
		case err := <-errChan:
			return err
		case msg := <-clientChan:
			var ev EventRecord
			if err := json.Unmarshal(msg.Data, &ev); err != nil {
				continue
			}

			if ev.EventType == "milestone_achieved" || ev.EventType == "milestone_expired" || ev.EventType == "milestone_stale_blocked" {
				var attrs map[string]string
				_ = json.Unmarshal(ev.Payload, &attrs)

				milestoneID := attrs["milestone_id"]
				if req.MilestoneId != "" && req.MilestoneId != milestoneID {
					continue
				}

				status := "achieved"
				if ev.EventType == "milestone_expired" {
					status = "expired"
				} else if ev.EventType == "milestone_stale_blocked" {
					status = "stale"
				}

				err := srv.Send(&backendv1.MilestoneEvent{
					MilestoneId: milestoneID,
					Status:      status,
					BlockHeight: ev.BlockHeight,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

// --- RelayerService Handlers ---

func (s *server) SubmitSignature(ctx context.Context, req *relayerv1.SubmitSignatureRequest) (*relayerv1.SubmitSignatureResponse, error) {
	// 1. Insert/Update settlements appending signature to signatures array
	_, err := s.readDB.Exec(ctx,
		`INSERT INTO settlements (settlement_id, proof, status, block_height, signatures)
		 VALUES ($1, $2, 'pending', 0, $3)
		 ON CONFLICT (settlement_id)
		 DO UPDATE SET signatures = array_append(settlements.signatures, $4)`,
		req.CommitteeId, []byte{}, []string{req.Signature}, req.Signature,
	)
	if err != nil {
		return &relayerv1.SubmitSignatureResponse{
			Success: false,
			Message: fmt.Sprintf("failed to save signature: %v", err),
		}, nil
	}

	return &relayerv1.SubmitSignatureResponse{
		Success: true,
		Message: "Signature submitted successfully",
	}, nil
}

func (s *server) GetCommitteeNonces(ctx context.Context, req *relayerv1.GetCommitteeNoncesRequest) (*relayerv1.GetCommitteeNoncesResponse, error) {
	var maxNonce int64
	err := s.readDB.QueryRow(ctx, "SELECT COALESCE(MAX(nonce), 0) FROM bridge_pending").Scan(&maxNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to query max nonce: %w", err)
	}

	return &relayerv1.GetCommitteeNoncesResponse{
		CommitteeId:   req.CommitteeId,
		CurrentNonce:  uint64(maxNonce),
		ExpectedNonce: uint64(maxNonce + 1),
	}, nil
}

func (s *server) GetTps(ctx context.Context, req *backendv1.GetTpsRequest) (*backendv1.GetTpsResponse, error) {
	var tpsAvg float64
	var tpsPeak float64
	var totalTxs int64

	err := s.readDB.QueryRow(ctx,
		`SELECT COALESCE(tps_avg, 0.0), COALESCE(tps_peak, 0.0), COALESCE(total_txs, 0)
		 FROM tps_1h
		 ORDER BY period DESC
		 LIMIT 1`,
	).Scan(&tpsAvg, &tpsPeak, &totalTxs)

	if err != nil {
		return &backendv1.GetTpsResponse{
			TpsAvg:   0.0,
			TpsPeak:  0.0,
			TotalTxs: 0,
		}, nil
	}

	return &backendv1.GetTpsResponse{
		TpsAvg:   tpsAvg,
		TpsPeak:  tpsPeak,
		TotalTxs: totalTxs,
	}, nil
}

func (s *server) GetBlockStats(ctx context.Context, req *backendv1.GetBlockStatsRequest) (*backendv1.GetBlockStatsResponse, error) {
	var avgMs float64
	var maxMs int64

	err := s.readDB.QueryRow(ctx,
		`SELECT COALESCE(avg_ms, 0.0), COALESCE(max_ms, 0)
		 FROM block_time_1h
		 ORDER BY period DESC
		 LIMIT 1`,
	).Scan(&avgMs, &maxMs)

	if err != nil {
		return &backendv1.GetBlockStatsResponse{
			AvgMs: 0.0,
			MaxMs: 0,
		}, nil
	}

	return &backendv1.GetBlockStatsResponse{
		AvgMs: avgMs,
		MaxMs: maxMs,
	}, nil
}

func (s *server) GetOraclePrice(ctx context.Context, req *backendv1.GetOraclePriceRequest) (*backendv1.GetOraclePriceResponse, error) {
	var open float64
	var high float64
	var low float64
	var close float64
	var submissionCount int64

	assetID := req.AssetId
	if assetID == "" {
		assetID = "BTC-USD"
	}

	err := s.readDB.QueryRow(ctx,
		`SELECT COALESCE(open, 0.0), COALESCE(high, 0.0), COALESCE(low, 0.0), COALESCE(close, 0.0), COALESCE(submission_count, 0)
		 FROM oracle_price_1h
		 WHERE asset_id = $1
		 ORDER BY period DESC
		 LIMIT 1`,
		assetID,
	).Scan(&open, &high, &low, &close, &submissionCount)

	if err != nil {
		return &backendv1.GetOraclePriceResponse{
			AssetId:         assetID,
			Open:            0.0,
			High:            0.0,
			Low:             0.0,
			Close:           0.0,
			SubmissionCount: 0,
		}, nil
	}

	return &backendv1.GetOraclePriceResponse{
		AssetId:         assetID,
		Open:            open,
		High:            high,
		Low:             low,
		Close:           close,
		SubmissionCount: submissionCount,
	}, nil
}

func (s *server) StreamChainStats(req *backendv1.StreamChainStatsRequest, srv backendv1.StreamService_StreamChainStatsServer) error {
	clientChan := make(chan *nats.Msg, 64)
	errChan := make(chan error, 1)

	sub, err := s.nc.Subscribe(NatsStreamSubject, func(msg *nats.Msg) {
		select {
		case clientChan <- msg:
		default:
			select {
			case errChan <- status.Error(codes.ResourceExhausted, "slow consumer channel buffer full"):
			default:
			}
		}
	})
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-srv.Context().Done():
			return nil
		case err := <-errChan:
			return err
		case msg := <-clientChan:
			var ev EventRecord
			if err := json.Unmarshal(msg.Data, &ev); err != nil {
				continue
			}

			if ev.EventType == "validator_uptime" {
				type ValidatorStatus struct {
					Address string `json:"address"`
					Signed  bool   `json:"signed"`
				}
				type ValidatorUptimePayload struct {
					Proposer   string            `json:"proposer"`
					Validators []ValidatorStatus `json:"validators"`
				}
				var payload ValidatorUptimePayload
				_ = json.Unmarshal(ev.Payload, &payload)

				err := srv.Send(&backendv1.ChainStatsEvent{
					TpsAvg:          1.0,
					TpsPeak:         5.0,
					TotalTxs:        int64(len(payload.Validators)),
					AvgBlockTimeMs: 6000.0,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func (s *server) ListSettlements(ctx context.Context, req *backendv1.ListSettlementsRequest) (*backendv1.ListSettlementsResponse, error) {
	limit := uint32(10)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
	}

	var cursorHeight int64 = 9223372036854775807
	var cursorID string = ""

	if req.Pagination != nil && len(req.Pagination.Cursor) > 0 {
		h, id, err := decodeCursor(req.Pagination.Cursor)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid cursor: %v", err)
		}
		cursorHeight = h
		cursorID = id
	}

	var rows pgx.Rows
	var err error
	if cursorID == "" {
		rows, err = s.readDB.Query(ctx,
			`SELECT settlement_id, proof, status, block_height, signatures
			 FROM settlements
			 ORDER BY block_height DESC, settlement_id DESC
			 LIMIT $1`,
			limit+1,
		)
	} else {
		rows, err = s.readDB.Query(ctx,
			`SELECT settlement_id, proof, status, block_height, signatures
			 FROM settlements
			 WHERE (block_height, settlement_id) < ($1, $2)
			 ORDER BY block_height DESC, settlement_id DESC
			 LIMIT $3`,
			cursorHeight, cursorID, limit+1,
		)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query settlements: %v", err)
	}
	defer rows.Close()

	var settlements []*backendv1.GetSettlementResponse
	for rows.Next() {
		var proof []byte
		var signatures []string
		res := &backendv1.GetSettlementResponse{}
		err := rows.Scan(&res.SettlementId, &proof, &res.Status, &res.BlockHeight, &signatures)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan settlements: %v", err)
		}
		res.Proof = proof
		res.Signatures = signatures
		settlements = append(settlements, res)
	}

	hasMore := false
	var nextCursor []byte
	if uint32(len(settlements)) > limit {
		hasMore = true
		lastItem := settlements[limit-1]
		nextCursor = []byte(encodeCursor(lastItem.BlockHeight, lastItem.SettlementId))
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

func (s *server) ListMilestones(ctx context.Context, req *backendv1.ListMilestonesRequest) (*backendv1.ListMilestonesResponse, error) {
	limit := uint32(10)
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
		if limit > 100 {
			limit = 100
		}
	}

	var cursorHeight int64 = 9223372036854775807
	var cursorID string = ""

	if req.Pagination != nil && len(req.Pagination.Cursor) > 0 {
		h, id, err := decodeCursor(req.Pagination.Cursor)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid cursor: %v", err)
		}
		cursorHeight = h
		cursorID = id
	}

	var rows pgx.Rows
	var err error
	if cursorID == "" {
		rows, err = s.readDB.Query(ctx,
			`SELECT milestone_id, status, block_height
			 FROM milestone_status
			 ORDER BY block_height DESC, milestone_id DESC
			 LIMIT $1`,
			limit+1,
		)
	} else {
		rows, err = s.readDB.Query(ctx,
			`SELECT milestone_id, status, block_height
			 FROM milestone_status
			 WHERE (block_height, milestone_id) < ($1, $2)
			 ORDER BY block_height DESC, milestone_id DESC
			 LIMIT $3`,
			cursorHeight, cursorID, limit+1,
		)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query milestones: %v", err)
	}
	defer rows.Close()

	var milestones []*backendv1.GetMilestoneStatusResponse
	for rows.Next() {
		res := &backendv1.GetMilestoneStatusResponse{}
		err := rows.Scan(&res.MilestoneId, &res.Status, &res.BlockHeight)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to scan milestones: %v", err)
		}
		milestones = append(milestones, res)
	}

	hasMore := false
	var nextCursor []byte
	if uint32(len(milestones)) > limit {
		hasMore = true
		lastItem := milestones[limit-1]
		nextCursor = []byte(encodeCursor(lastItem.BlockHeight, lastItem.MilestoneId))
		milestones = milestones[:limit]
	}

	return &backendv1.ListMilestonesResponse{
		Milestones: milestones,
		Pagination: &backendv1.PageResponse{
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	}, nil
}

func (s *server) GetBridgeTx(ctx context.Context, req *backendv1.GetBridgeTxRequest) (*backendv1.GetBridgeTxResponse, error) {
	var height int64
	var index int
	_, err := fmt.Sscanf(req.TxHash, "tx_%d_%d", &height, &index)
	if err == nil {
		nonce := height*1000 + int64(index)
		var tokenAddress string
		var amount float64
		var recipient string
		var statusStr string
		errPending := s.readDB.QueryRow(ctx,
			"SELECT token_address, amount, recipient, status FROM bridge_pending WHERE nonce = $1",
			nonce,
		).Scan(&tokenAddress, &amount, &recipient, &statusStr)
		if errPending == nil {
			return &backendv1.GetBridgeTxResponse{
				TxHash:       req.TxHash,
				Status:       statusStr,
				BlockHeight:  height,
				TokenAddress: tokenAddress,
				Amount:       fmt.Sprintf("%.0f", amount),
				Sender:       "bsc",
				Recipient:    recipient,
			}, nil
		}
	}

	var direction string
	var asset string
	var amount float64
	err = s.readDB.QueryRow(ctx,
		"SELECT direction, asset, amount FROM bridge_events WHERE block_height = $1 AND event_index = $2",
		height, index,
	).Scan(&direction, &asset, &amount)
	if err == nil {
		statusStr := "confirmed"
		if direction == "lock" {
			statusStr = "executed"
		}
		return &backendv1.GetBridgeTxResponse{
			TxHash:       req.TxHash,
			Status:       statusStr,
			BlockHeight:  height,
			TokenAddress: asset,
			Amount:       fmt.Sprintf("%.0f", amount),
			Sender:       "bsc",
			Recipient:    "cosmos",
		}, nil
	}

	return &backendv1.GetBridgeTxResponse{
		TxHash:       req.TxHash,
		Status:       "unknown",
		BlockHeight:  0,
		TokenAddress: "usov",
		Amount:       "0",
		Sender:       "unknown",
		Recipient:    "unknown",
	}, nil
}

func encodeCursor(height int64, extraID string) string {
	str := fmt.Sprintf("%d,%s", height, extraID)
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func decodeCursor(cursorBytes []byte) (int64, string, error) {
	if len(cursorBytes) == 0 {
		return 0, "", nil
	}
	decoded, err := base64.StdEncoding.DecodeString(string(cursorBytes))
	if err != nil {
		// fallback to raw bytes decoding if not string
		decoded, err = base64.StdEncoding.DecodeString(base64.StdEncoding.EncodeToString(cursorBytes))
		if err != nil {
			return 0, "", err
		}
	}
	parts := strings.SplitN(string(decoded), ",", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid cursor format")
	}
	h, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, "", err
	}
	return h, parts[1], nil
}

func getNatsNkeyOption(role string) (nats.Option, error) {
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")
	
	var seed string

	if vaultAddr != "" && vaultToken != "" {
		url := fmt.Sprintf("%s/v1/secret/data/sovereign/nats", vaultAddr)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Vault-Token", vaultToken)
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == 200 {
			var result struct {
				Data struct {
					Data map[string]interface{} `json:"data"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				if keyVal, ok := result.Data.Data[role+"_nkey"]; ok {
					seed = fmt.Sprintf("%v", keyVal)
				}
			}
		}
	}
	
	if seed == "" {
		envName := strings.ToUpper(role) + "_NKEY_SEED"
		seed = os.Getenv(envName)
	}
	
	if seed == "" {
		if role == "ingestion" {
			seed = "SUAFFNTD6H6ST7VGTZDXYQDC5BPNGYRTEFY4TZM32TJEMBTFN5TJO4WNXU"
		} else if role == "projection" {
			seed = "SUAINVHHXAR4PZTQC4VEME4P3HB2CQ3QNQY4WK3YNULE2IJZLNOLNDGBUE"
		} else if role == "stream" {
			seed = "SUAO6IIZLMQHQYVKKHJIEXIC5T6XNKM2PUVF4EGZW23UALD7WTFFE7R2LQ"
		}
	}

	if seed == "" {
		return nil, fmt.Errorf("NKey seed not found for role %s", role)
	}

	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, fmt.Errorf("failed to parse NKey seed: %w", err)
	}
	pubKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from NKey seed: %w", err)
	}
	opt := nats.Nkey(pubKey, func(nonce []byte) ([]byte, error) {
		return kp.Sign(nonce)
	})
	return opt, nil
}

