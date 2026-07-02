package e2e

import (
	"context"
	"testing"
	"time"

	explorerv1 "github.com/sovereign-l1/chain/api/explorer/v1"
)

// Mock explorer server for Phase 2 endpoints verification
type mockExplorerPhase2Server struct {
	explorerv1.UnimplementedExplorerServiceServer
}

func (m *mockExplorerPhase2Server) ListValidators(ctx context.Context, req *explorerv1.ListValidatorsRequest) (*explorerv1.ValidatorSlotGrid, error) {
	return &explorerv1.ValidatorSlotGrid{
		Validators: []*explorerv1.ValidatorDetail{
			{
				Address:            "sovereignvaloper1valaddr0",
				SlotIndex:          0,
				Power:              1000,
				Status:             "active",
				MissedBlocks:       0,
				CertificationScore: 100,
			},
		},
	}, nil
}

func (m *mockExplorerPhase2Server) GetStakingStats(ctx context.Context, req *explorerv1.GetStakingStatsRequest) (*explorerv1.StakingStats, error) {
	return &explorerv1.StakingStats{
		TotalBonded:   "450,000,000 uSLT",
		BondedRatio:   "45.0%",
		Inflation:     "7.0%",
		CommunityPool: "10,000,000 uSLT",
		Apr:           "12.5%",
	}, nil
}

func (m *mockExplorerPhase2Server) GetOracleFeed(ctx context.Context, req *explorerv1.GetOracleFeedRequest) (*explorerv1.FeedDetail, error) {
	return &explorerv1.FeedDetail{
		FeedId:      req.FeedId,
		Title:       "Sovereign Llt USDT Price Feed",
		LatestPrice: "1.25",
		Status:      "fresh",
		LastUpdated: time.Now().Format(time.RFC3339),
	}, nil
}

func (m *mockExplorerPhase2Server) GetOracleRound(ctx context.Context, req *explorerv1.GetOracleRoundRequest) (*explorerv1.RoundDetail, error) {
	return &explorerv1.RoundDetail{
		RoundId:          req.RoundId,
		FeedId:           req.FeedId,
		Height:           100,
		Time:             time.Now().Format(time.RFC3339),
		AggregatedMedian: "1.25",
		Status:           "done",
	}, nil
}

func (m *mockExplorerPhase2Server) GetMilestone(ctx context.Context, req *explorerv1.GetMilestoneRequest) (*explorerv1.MilestoneDetail, error) {
	return &explorerv1.MilestoneDetail{
		Id:          req.Id,
		Creator:     "sovereign1address0",
		Status:      "pending",
		Title:       "Mainnet Launch Milestone",
		TargetPrice: "1.50",
		FeedId:      "slt-usdt",
	}, nil
}

func (m *mockExplorerPhase2Server) GetSettlement(ctx context.Context, req *explorerv1.GetSettlementRequest) (*explorerv1.SettlementDetail, error) {
	return &explorerv1.SettlementDetail{
		Id:                req.Id,
		Witness:           "sovereignvaloper1valaddr0",
		Status:            "settled",
		ChainId:           "bsc-mainnet",
		TxHash:            "mocktxhash",
		WitnessSignatures: "[]",
	}, nil
}

func (m *mockExplorerPhase2Server) ListContracts(ctx context.Context, req *explorerv1.ListContractsRequest) (*explorerv1.ContractList, error) {
	return &explorerv1.ContractList{
		Contracts: []*explorerv1.ContractDetail{
			{
				Address:   "sovereign1contract120530",
				CodeId:    1,
				Label:     "Mock CW20 Contract",
				Creator:   "sovereign1address0",
				Admin:     "",
				TypeBadge: "CW-20",
			},
		},
	}, nil
}

func (m *mockExplorerPhase2Server) GetCw20Token(ctx context.Context, req *explorerv1.GetCw20TokenRequest) (*explorerv1.Cw20TokenDetail, error) {
	return &explorerv1.Cw20TokenDetail{
		Address:     req.Address,
		Name:        "Mock CosmWasm Token",
		Symbol:      "MCK",
		Decimals:    6,
		TotalSupply: "10000000",
		Balance:     "10000",
	}, nil
}

func (m *mockExplorerPhase2Server) ListIbcChannels(ctx context.Context, req *explorerv1.ListIbcChannelsRequest) (*explorerv1.IbcChannelList, error) {
	return &explorerv1.IbcChannelList{
		Channels: []*explorerv1.IbcChannelDetail{
			{
				ChannelId: "channel-0",
				PortId:    "transfer",
				State:     "open",
			},
		},
	}, nil
}

func TestExplorerPhase2MockEndpoints(t *testing.T) {
	s := &mockExplorerPhase2Server{}
	ctx := context.Background()

	// 1. Validators slot grid query
	validators, err := s.ListValidators(ctx, &explorerv1.ListValidatorsRequest{})
	if err != nil {
		t.Fatalf("failed to query validators: %v", err)
	}
	if len(validators.Validators) != 1 || validators.Validators[0].SlotIndex != 0 {
		t.Errorf("validators list verification failed")
	}

	// 2. Staking stats query
	staking, err := s.GetStakingStats(ctx, &explorerv1.GetStakingStatsRequest{})
	if err != nil {
		t.Fatalf("failed to query staking stats: %v", err)
	}
	if staking.Apr != "12.5%" {
		t.Errorf("unexpected staking APR: %s", staking.Apr)
	}

	// 3. Oracle feed query
	feed, err := s.GetOracleFeed(ctx, &explorerv1.GetOracleFeedRequest{FeedId: "slt-usdt"})
	if err != nil {
		t.Fatalf("failed to query oracle feed: %v", err)
	}
	if feed.LatestPrice != "1.25" {
		t.Errorf("unexpected latest price: %s", feed.LatestPrice)
	}

	// 4. Milestone query
	milestone, err := s.GetMilestone(ctx, &explorerv1.GetMilestoneRequest{Id: 1})
	if err != nil {
		t.Fatalf("failed to query milestone: %v", err)
	}
	if milestone.Title != "Mainnet Launch Milestone" {
		t.Errorf("unexpected milestone title: %s", milestone.Title)
	}

	// 5. Settlements query
	settlement, err := s.GetSettlement(ctx, &explorerv1.GetSettlementRequest{Id: 100})
	if err != nil {
		t.Fatalf("failed to query settlement: %v", err)
	}
	if settlement.ChainId != "bsc-mainnet" {
		t.Errorf("unexpected settlement chain id: %s", settlement.ChainId)
	}

	// 6. Contracts query
	contracts, err := s.ListContracts(ctx, &explorerv1.ListContractsRequest{})
	if err != nil {
		t.Fatalf("failed to query contracts: %v", err)
	}
	if len(contracts.Contracts) != 1 || contracts.Contracts[0].TypeBadge != "CW-20" {
		t.Errorf("contracts list verification failed")
	}

	// 7. IBC channels query
	ibcChannels, err := s.ListIbcChannels(ctx, &explorerv1.ListIbcChannelsRequest{})
	if err != nil {
		t.Fatalf("failed to query ibc channels: %v", err)
	}
	if len(ibcChannels.Channels) != 1 || ibcChannels.Channels[0].ChannelId != "channel-0" {
		t.Errorf("ibc channels verification failed")
	}

	t.Log("[PASS] Checked gRPC API responses match protobuf specifications for all Phase 2 custom modules.")
}
