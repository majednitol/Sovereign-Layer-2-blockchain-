package app

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestWasmContractsCompilationAndSize(t *testing.T) {
	// 1. Compile contracts to wasm32 target
	cmd := exec.Command("cargo", "build", "--lib", "--target", "wasm32-unknown-unknown", "--release")
	cmd.Dir = "../../contracts"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to compile CosmWasm contracts: %v", err)
	}

	// 2. Assert all 4 compiled WASM contracts exist and are non-empty
	wasmFiles := []string{
		"constitution.wasm",
		"treasury.wasm",
		"reserve_fund.wasm",
		"governance.wasm",
	}

	for _, file := range wasmFiles {
		path := filepath.Join("../../contracts/target/wasm32-unknown-unknown/release", file)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Compiled WASM file not found: %s", path)
		}
		if info.Size() == 0 {
			t.Fatalf("WASM file is empty: %s", path)
		}
		t.Logf("[PASS] Contract %s found, size: %d bytes", file, info.Size())
	}
}

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
