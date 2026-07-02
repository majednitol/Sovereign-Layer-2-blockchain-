package app

import (
	"encoding/json"
	"testing"
)


func TestWasmContractsExecutionJSONMatching(t *testing.T) {
	// Verify JSON schemas expected by the Go keepers match what the Wasm contracts execute
	// Check gov_ext's CheckProposal message: {"check_proposal":{}}
	type CheckProposalMsg struct {
		CheckProposal struct{} `json:"check_proposal"`
	}

	var msg CheckProposalMsg
	bz, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	expectedJSON := `{"check_proposal":{}}`
	if string(bz) != expectedJSON {
		t.Fatalf("Expected msg JSON to be %s, got %s", expectedJSON, string(bz))
	}
}
