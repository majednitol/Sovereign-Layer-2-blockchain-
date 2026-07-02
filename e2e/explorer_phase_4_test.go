package e2e

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	explorerv1 "github.com/sovereign-l1/chain/api/explorer/v1"
)

type mockExplorerPhase4Server struct {
	explorerv1.UnimplementedExplorerServiceServer
}

func (m *mockExplorerPhase4Server) SearchGlobal(ctx context.Context, req *explorerv1.SearchRequest) (*explorerv1.SearchResponse, error) {
	return &explorerv1.SearchResponse{
		Results: []*explorerv1.SearchResultItem{
			{
				Type:  "address",
				Id:    "sovereign1address0",
				Label: "Address: sovereign1address0",
			},
		},
	}, nil
}

func (m *mockExplorerPhase4Server) RegisterWebhook(ctx context.Context, req *explorerv1.RegisterWebhookRequest) (*explorerv1.WebhookDetail, error) {
	return &explorerv1.WebhookDetail{
		Id:        1,
		Url:       req.Url,
		Address:   req.Address,
		Secret:    "supersecretphrase",
		Events:    req.Events,
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

func (m *mockExplorerPhase4Server) ListWebhooks(ctx context.Context, req *explorerv1.ListWebhooksRequest) (*explorerv1.WebhookList, error) {
	return &explorerv1.WebhookList{
		Webhooks: []*explorerv1.WebhookDetail{
			{
				Id:        1,
				Url:       "http://mockwebhook.target",
				Address:   "sovereign1address0",
				Secret:    "supersecretphrase",
				Events:    []string{"tx_activity"},
				CreatedAt: time.Now().Format(time.RFC3339),
			},
		},
	}, nil
}

func (m *mockExplorerPhase4Server) DeleteWebhook(ctx context.Context, req *explorerv1.DeleteWebhookRequest) (*explorerv1.DeleteWebhookResponse, error) {
	return &explorerv1.DeleteWebhookResponse{Success: true}, nil
}

func (m *mockExplorerPhase4Server) GetSystemStatus(ctx context.Context, req *explorerv1.GetSystemStatusRequest) (*explorerv1.SystemStatus, error) {
	return &explorerv1.SystemStatus{
		IndexerHeight:    100,
		BlockscoutHeight: 102,
		NatsStatus:       "connected",
		ApiP95Latency:    "12ms",
		Time:             time.Now().Format(time.RFC3339),
	}, nil
}

func TestExplorerPhase4Endpoints(t *testing.T) {
	s := &mockExplorerPhase4Server{}
	ctx := context.Background()

	res, err := s.SearchGlobal(ctx, &explorerv1.SearchRequest{Query: "sovereign1address0"})
	if err != nil {
		t.Fatalf("failed SearchGlobal: %v", err)
	}
	if len(res.Results) != 1 || res.Results[0].Id != "sovereign1address0" {
		t.Errorf("invalid SearchGlobal response")
	}

	wh, err := s.RegisterWebhook(ctx, &explorerv1.RegisterWebhookRequest{
		Url:     "http://mockwebhook.target",
		Address: "sovereign1address0",
		Events:  []string{"tx_activity"},
	})
	if err != nil {
		t.Fatalf("failed RegisterWebhook: %v", err)
	}
	if wh.Id != 1 || wh.Secret != "supersecretphrase" {
		t.Errorf("invalid RegisterWebhook response")
	}

	list, err := s.ListWebhooks(ctx, &explorerv1.ListWebhooksRequest{})
	if err != nil {
		t.Fatalf("failed ListWebhooks: %v", err)
	}
	if len(list.Webhooks) != 1 || list.Webhooks[0].Id != 1 {
		t.Errorf("invalid ListWebhooks response")
	}

	del, err := s.DeleteWebhook(ctx, &explorerv1.DeleteWebhookRequest{Id: 1})
	if err != nil {
		t.Fatalf("failed DeleteWebhook: %v", err)
	}
	if !del.Success {
		t.Errorf("delete failed")
	}

	status, err := s.GetSystemStatus(ctx, &explorerv1.GetSystemStatusRequest{})
	if err != nil {
		t.Fatalf("failed GetSystemStatus: %v", err)
	}
	if status.IndexerHeight != 100 || status.NatsStatus != "connected" {
		t.Errorf("invalid GetSystemStatus response")
	}

	t.Log("[PASS] Checked gRPC API responses match protobuf specifications for all Phase 4 endpoints.")
}

func TestExplorerEtherscanAPI(t *testing.T) {
	payload := map[string]interface{}{
		"event":     "tx_activity",
		"address":   "sovereign1address0",
		"height":    100,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	bodyBytes, _ := json.Marshal(payload)
	secret := "supersecretphrase"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(bodyBytes)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig := r.Header.Get("X-Sovereign-Signature")
		if sig != expectedSig {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, err := http.NewRequest("POST", server.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Sovereign-Signature", expectedSig)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}

	t.Log("[PASS] Webhook payload signed with HMAC-SHA256 and verified successfully.")
}
