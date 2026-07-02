package main

import (
	"encoding/json"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1000000usov", 1000000},
		{"500000000", 500000000},
		{"0", 0},
		{"abc", 0},
		{"12.34", 12}, // extracts first digit sequence
		{"1234usov", 1234},
	}

	for _, tc := range tests {
		got := parseAmount(tc.input)
		if got != tc.expected {
			t.Errorf("parseAmount(%q) = %f; expected %f", tc.input, got, tc.expected)
		}
	}
}

func TestEventParsingStructures(t *testing.T) {
	// Test MsgBridgeIn parsing
	inPayload := []byte(`{"receiver":"sov123","amount":"5000000usov","nonce":"0x1A"}`)
	var attrs map[string]string
	if err := json.Unmarshal(inPayload, &attrs); err != nil {
		t.Fatalf("failed to parse MsgBridgeIn payload: %v", err)
	}

	if attrs["receiver"] != "sov123" {
		t.Errorf("expected receiver sov123, got %s", attrs["receiver"])
	}
	if attrs["amount"] != "5000000usov" {
		t.Errorf("expected amount 5000000usov, got %s", attrs["amount"])
	}
	if attrs["nonce"] != "0x1A" {
		t.Errorf("expected nonce 0x1A, got %s", attrs["nonce"])
	}

	// Test validator uptime payload
	uptimePayloadStr := `{"proposer":"addr1","validators":[{"address":"val1","signed":true},{"address":"val2","signed":false}]}`
	type ValidatorStatus struct {
		Address string `json:"address"`
		Signed  bool   `json:"signed"`
	}
	type ValidatorUptimePayload struct {
		Proposer   string            `json:"proposer"`
		Validators []ValidatorStatus `json:"validators"`
	}

	var uptimePayload ValidatorUptimePayload
	if err := json.Unmarshal([]byte(uptimePayloadStr), &uptimePayload); err != nil {
		t.Fatalf("failed to parse validator uptime payload: %v", err)
	}

	if uptimePayload.Proposer != "addr1" {
		t.Errorf("expected proposer addr1, got %s", uptimePayload.Proposer)
	}
	if len(uptimePayload.Validators) != 2 {
		t.Errorf("expected 2 validators, got %d", len(uptimePayload.Validators))
	}
	if !uptimePayload.Validators[0].Signed || uptimePayload.Validators[1].Signed {
		t.Errorf("validator signing flags mismatch")
	}
}
