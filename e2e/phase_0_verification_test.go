package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhase0RootDirectories verifies that all 15 required directories exist at the root.
func TestPhase0RootDirectories(t *testing.T) {
	requiredDirs := []string{
		"chain",
		"contracts",
		"bridge",
		"relayer",
		"oracle",
		"backend",
		"proto",
		"evm",
		"explorer",
		"frontend",
		"infra",
		"nats",
		"scripts",
		"e2e",
		"db",
	}

	for _, dir := range requiredDirs {
		path := filepath.Join("..", dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("FAIL: Required directory %s is missing at root: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("FAIL: Path %s exists but is not a directory", path)
		}
		t.Logf("[PASS] Verified directory exists: /%s", dir)
	}
}

// TestPhase0DockerComposeConfig parses the docker-compose.yml file to ensure service requirements are met.
func TestPhase0DockerComposeConfig(t *testing.T) {
	composePath := filepath.Join("..", "docker-compose.yml")
	content, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("FAIL: Could not read docker-compose.yml: %v", err)
	}

	strContent := string(content)

	// 1. Verify isolated databases
	dbs := []string{"db-write", "db-read", "db-relayer"}
	for _, db := range dbs {
		if !strings.Contains(strContent, db) {
			t.Fatalf("FAIL: PostgreSQL isolated service %s is not declared in docker-compose.yml", db)
		}
		t.Logf("[PASS] Verified database service declared: %s", db)
	}

	// 2. Verify NATS 3-node cluster
	natsNodes := []string{"nats-0", "nats-1", "nats-2"}
	for _, node := range natsNodes {
		if !strings.Contains(strContent, node) {
			t.Fatalf("FAIL: NATS cluster node %s is not declared in docker-compose.yml", node)
		}
		t.Logf("[PASS] Verified NATS node declared: %s", node)
	}

	// 3. Verify Envoy and chain node
	services := []string{"envoy", "chain-node"}
	for _, service := range services {
		if !strings.Contains(strContent, service) {
			t.Fatalf("FAIL: Service %s is not declared in docker-compose.yml", service)
		}
		t.Logf("[PASS] Verified service declared: %s", service)
	}
}

// TestPhase0CIWorkflowConfig parses the .github/workflows/ci.yml configuration.
func TestPhase0CIWorkflowConfig(t *testing.T) {
	ciPath := filepath.Join("..", ".github", "workflows", "ci.yml")
	content, err := os.ReadFile(ciPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read CI configuration file: %v", err)
	}

	strContent := string(content)

	// 1. Check target branches
	if !strings.Contains(strContent, "branches: [ master ]") && !strings.Contains(strContent, "branches: [ \"master\" ]") {
		if !strings.Contains(strContent, "master") {
			t.Fatal("FAIL: CI workflow is not configured to run on master branch triggers")
		}
	}

	// 2. Check Go Version
	if !strings.Contains(strContent, "1.25.5") {
		t.Fatal("FAIL: CI workflow is not pinned to Go version 1.25.5")
	}
	t.Log("[PASS] Checked pinned Go version 1.25.5 in CI workflow.")

	// 3. Check Protobuf Lint and Breaking Check
	if !strings.Contains(strContent, "bufbuild/buf-lint-action") {
		t.Fatal("FAIL: CI workflow is missing buf lint action")
	}
	if !strings.Contains(strContent, "bufbuild/buf-breaking-action") {
		t.Fatal("FAIL: CI workflow is missing buf breaking backward compatibility action")
	}
	t.Log("[PASS] Checked Protobuf lint and backward compatibility checks in CI.")

	// 4. Check simulation execution
	if !strings.Contains(strContent, "TestAppSimulation") {
		t.Fatal("FAIL: CI workflow is missing App Simulation execution")
	}
	if !strings.Contains(strContent, "-NumBlocks=500") || !strings.Contains(strContent, "-BlockSize=200") {
		t.Fatal("FAIL: CI workflow does not execute simulation with 500 blocks and block size 200")
	}
	if !strings.Contains(strContent, "Seed=") {
		t.Fatal("FAIL: CI workflow is missing randomized seed parameters logging")
	}
	t.Log("[PASS] Checked randomized SimApp simulation steps in CI configuration.")
}

// TestPhase0ADRsAndVersionPinning verifies architecture records and dependency tracking.
func TestPhase0ADRsAndVersionPinning(t *testing.T) {
	// 1. Check all 7 ADRs exist
	adrs := []string{
		"adr-001-validator-cardinality.md",
		"adr-002-certification-liveness.md",
		"adr-003-oracle-commit-reveal.md",
		"adr-004-bridge-security-model.md",
		"adr-005-cqrs-nats-topology.md",
		"adr-006-cosmwasm-governance.md",
		"adr-007-operational-security.md",
	}

	for _, adr := range adrs {
		path := filepath.Join("..", "doc", "adr", adr)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("FAIL: Architecture Decision Record %s is missing: %v", adr, err)
		}
		t.Logf("[PASS] Verified ADR existence: %s", adr)
	}

	// 2. Verify Version Pinning inside ADR 007
	adr7Path := filepath.Join("..", "doc", "adr", "adr-007-operational-security.md")
	adrContent, err := os.ReadFile(adr7Path)
	if err != nil {
		t.Fatalf("FAIL: Could not read ADR 007: %v", err)
	}

	strAdr := string(adrContent)
	expectedVersions := []string{
		"v0.50.9", // cosmos-sdk
		"v0.38.11", // cometbft
		"v0.50.0", // wasmd
		"v1.5.0",  // wasmvm
		"v8.0.0",  // ibc-go
	}

	for _, version := range expectedVersions {
		if !strings.Contains(strAdr, version) {
			t.Fatalf("FAIL: Pinned version %s is not recorded in ADR 007", version)
		}
		t.Logf("[PASS] Verified version recorded in ADR 007: %s", version)
	}
}
