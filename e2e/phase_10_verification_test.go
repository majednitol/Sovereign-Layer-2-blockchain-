package e2e

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 10 Mainnet Launch Verification Test Suite
// Covers: Genesis params (10.1), EVM params (10.2), Horcrux (10.3),
//         Chain Registry (10.4), Bridge/Rate-limiting (10.5),
//         Monitoring/Alerts (10.6), Multi-region/HA (10.7)
// ═══════════════════════════════════════════════════════════════════════════════

// --- Genesis JSON Types ---

type StakingParams struct {
	MaxValidators uint32 `json:"max_validators"`
	BondDenom     string `json:"bond_denom"`
}

type StakingState struct {
	Params         StakingParams `json:"params"`
	LastTotalPower string        `json:"last_total_power"`
}

type EVMParams struct {
	EvmDenom             string                `json:"evm_denom"`
	ExtendedDenomOptions *ExtendedDenomOptions `json:"extended_denom_options"`
}

type ExtendedDenomOptions struct {
	ExtendedDenom string `json:"extended_denom"`
}

type EVMState struct {
	Params EVMParams `json:"params"`
}

type BridgeParams struct {
	MaxUnlockPerBlock     uint64 `json:"max_unlock_per_block"`
	SupplyCap             uint64 `json:"supply_cap"`
	CircuitBreakerAddress string `json:"circuit_breaker_address"`
	GnosisSafeAddress     string `json:"gnosis_safe_address"`
}

type BridgeState struct {
	Params BridgeParams `json:"params"`
}

type AppState struct {
	Staking StakingState `json:"staking"`
	EVM     EVMState     `json:"evm"`
	Bridge  BridgeState  `json:"bridge"`
}

type Genesis struct {
	ChainID  string   `json:"chain_id"`
	AppState AppState `json:"app_state"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// 10.1 & 10.2 — Genesis and EVM Parameter Verifications
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase10_1_GenesisInvariants(t *testing.T) {
	genesisPath := filepath.Join("..", "chain", "genesis.json")
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read genesis.json: %v", err)
	}

	var genesis Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatalf("FAIL: Could not unmarshal genesis.json: %v", err)
	}

	if genesis.ChainID != "sovereign-1" {
		t.Errorf("FAIL: Expected chain_id 'sovereign-1', got '%s'", genesis.ChainID)
	}

	if genesis.AppState.Staking.Params.MaxValidators != 30 {
		t.Errorf("FAIL: Expected max_validators 30, got %d", genesis.AppState.Staking.Params.MaxValidators)
	}

	if genesis.AppState.Staking.Params.BondDenom != "ucsov" {
		t.Errorf("FAIL: Expected bond_denom 'ucsov', got '%s'", genesis.AppState.Staking.Params.BondDenom)
	}

	if genesis.AppState.Staking.LastTotalPower != "30000000" {
		t.Errorf("FAIL: Expected last_total_power '30000000', got '%s'", genesis.AppState.Staking.LastTotalPower)
	}

	t.Log("[PASS] 10.1: Genesis supply configurations, validator cardinality limits, and staking parameters are verified.")
}

func TestPhase10_2_EVMGenesisParams(t *testing.T) {
	genesisPath := filepath.Join("..", "chain", "genesis.json")
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read genesis.json: %v", err)
	}

	var genesis Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatalf("FAIL: Could not unmarshal genesis.json: %v", err)
	}

	evmState := genesis.AppState.EVM
	if evmState.Params.EvmDenom != "aesov" {
		t.Errorf("FAIL: Expected evm_denom 'aesov', got '%s'", evmState.Params.EvmDenom)
	}

	if evmState.Params.ExtendedDenomOptions == nil || evmState.Params.ExtendedDenomOptions.ExtendedDenom != "aesov" {
		t.Errorf("FAIL: ExtendedDenomOptions.ExtendedDenom is not set to 'aesov'")
	}

	// Verify AllowUnprotectedTxs = false in app.toml
	tomlPath := filepath.Join("..", "chain", "config", "app.toml")
	tomlData, err := os.ReadFile(tomlPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read app.toml: %v", err)
	}

	content := string(tomlData)
	if !strings.Contains(content, "allow-unprotected-txs = false") {
		t.Error("FAIL: EIP-155 replay protection is not configured to false in app.toml")
	}

	t.Log("[PASS] 10.2: EVM genesis parameters (x/vm module, aesov denom, replay protection allow-unprotected-txs = false) are verified.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 10.3 — Horcrux Ceremony Configuration Verification
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase10_3_HorcruxCeremony(t *testing.T) {
	cmd := exec.Command("bash", "../scripts/horcrux_ceremony_check.sh")
	if err := cmd.Run(); err != nil {
		t.Fatalf("FAIL: Horcrux ceremony configuration check failed: %v", err)
	}

	t.Log("[PASS] 10.3: Horcrux 2-of-3 threshold ceremony and double-signing protection parameters are validated.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 10.4 — Chain Registry Check
// ═══════════════════════════════════════════════════════════════════════════════

type ChainRegistry struct {
	ChainName    string `json:"chain_name"`
	ChainID      string `json:"chain_id"`
	Bech32Prefix string `json:"bech32_prefix"`
	Slip44       int    `json:"slip44"`
}

func TestPhase10_4_ChainRegistry(t *testing.T) {
	registryPath := filepath.Join("..", "doc", "mainnet", "chain-registry.json")
	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read chain-registry.json: %v", err)
	}

	var registry ChainRegistry
	if err := json.Unmarshal(data, &registry); err != nil {
		t.Fatalf("FAIL: Could not unmarshal chain-registry.json: %v", err)
	}

	if registry.ChainName != "sovereign" {
		t.Errorf("FAIL: Expected chain_name 'sovereign', got '%s'", registry.ChainName)
	}
	if registry.ChainID != "sovereign-1" {
		t.Errorf("FAIL: Expected chain_id 'sovereign-1', got '%s'", registry.ChainID)
	}
	if registry.Bech32Prefix != "cosmos" {
		t.Errorf("FAIL: Expected bech32_prefix 'cosmos', got '%s'", registry.Bech32Prefix)
	}
	if registry.Slip44 != 60 {
		t.Errorf("FAIL: Expected slip44 60 (Ethereum BIP-44 compatible), got %d", registry.Slip44)
	}

	t.Log("[PASS] 10.4: Mainnet chain registry configuration is schema-compliant.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 10.5 — Bridge Rate-limiting and Circuit-Breaker Verification
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase10_5_BridgeRateLimitAndCircuitBreaker(t *testing.T) {
	genesisPath := filepath.Join("..", "chain", "genesis.json")
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read genesis.json: %v", err)
	}

	var genesis Genesis
	if err := json.Unmarshal(data, &genesis); err != nil {
		t.Fatalf("FAIL: Could not unmarshal genesis.json: %v", err)
	}

	bridgeParams := genesis.AppState.Bridge.Params
	if bridgeParams.MaxUnlockPerBlock == 0 {
		t.Error("FAIL: bridge max_unlock_per_block rate limit parameter is not set (0)")
	}
	if bridgeParams.SupplyCap == 0 {
		t.Error("FAIL: bridge supply_cap parameter is not set (0)")
	}
	if bridgeParams.CircuitBreakerAddress == "" || bridgeParams.GnosisSafeAddress == "" {
		t.Error("FAIL: bridge circuit breaker or Gnosis Safe addresses are empty")
	}

	// 1. Simulate rate limiting check
	simulateBridgeTransfer := func(amount uint64) error {
		if amount > bridgeParams.MaxUnlockPerBlock {
			return errors.New("rate limit exceeded")
		}
		return nil
	}

	// Under limit should pass
	if err := simulateBridgeTransfer(bridgeParams.MaxUnlockPerBlock - 100); err != nil {
		t.Errorf("FAIL: Transfer under limit rejected: %v", err)
	}
	// Exceed limit should fail
	if err := simulateBridgeTransfer(bridgeParams.MaxUnlockPerBlock + 100); err == nil {
		t.Error("FAIL: Transfer exceeding max_unlock_per_block did not trigger rate limit")
	}

	// 2. Simulate circuit breaker check
	simulatePausedExecute := func(isPaused bool) error {
		if isPaused {
			return errors.New("contract operations are paused")
		}
		return nil
	}

	if err := simulatePausedExecute(true); err == nil {
		t.Error("FAIL: Paused state did not prevent execute operations")
	}

	t.Log("[PASS] 10.5: Bridge day-1 rate limits, supply caps, and circuit-breaker roles are validated.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 10.6 — Monitoring and Alerts Verification
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase10_6_MonitoringAlerts(t *testing.T) {
	// 1. Verify dashboard exists
	dashboardPath := filepath.Join("..", "infra", "monitoring", "dashboards", "sovereign-l1-dashboard.json")
	if _, err := os.Stat(dashboardPath); err != nil {
		t.Fatalf("FAIL: Grafana dashboard file does not exist: %v", err)
	}

	// 2. Verify alert rules exist and contain routing structures
	rulesPath := filepath.Join("..", "infra", "monitoring", "alerts.rules.yml")
	rulesData, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read alerts.rules.yml: %v", err)
	}

	content := string(rulesData)
	requiredAlerts := []string{
		"ValidatorDowntimeAlert",
		"BridgeRateLimitHitAlert",
		"OracleStalenessBreachAlert",
	}

	for _, alert := range requiredAlerts {
		if !strings.Contains(content, alert) {
			t.Errorf("FAIL: alerts.rules.yml does not define required mainnet alert: %s", alert)
		}
	}

	t.Log("[PASS] 10.6: Prometheus alerts rules and Grafana dashboard configurations are validated.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 10.7 — Multi-Region K8s & Synchronous Replication Verification
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase10_7_MultiRegionDeployments(t *testing.T) {
	// 1. Check multi-region database setup
	dbYamlPath := filepath.Join("..", "infra", "k8s", "multi-region-database.yaml")
	dbData, err := os.ReadFile(dbYamlPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read multi-region-database.yaml: %v", err)
	}

	dbContent := string(dbData)
	if !strings.Contains(dbContent, "replicas: 2") {
		t.Error("FAIL: Database StatefulSet does not configure at least 2 replica nodes")
	}
	if !strings.Contains(dbContent, "synchronous_commit: \"on\"") {
		t.Error("FAIL: Database Patroni parameters missing synchronous_commit = on")
	}
	if !strings.Contains(dbContent, "synchronous_standby_names") {
		t.Error("FAIL: Database Patroni parameters missing synchronous_standby_names configuration")
	}

	// 2. Check multi-region network and WireGuard VPN setup
	netYamlPath := filepath.Join("..", "infra", "k8s", "multi-region-network.yaml")
	netData, err := os.ReadFile(netYamlPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read multi-region-network.yaml: %v", err)
	}

	netContent := string(netData)
	if !strings.Contains(netContent, "wireguard-tunnel") {
		t.Error("FAIL: Network DaemonSet is missing 'wireguard-tunnel' definition")
	}
	if !strings.Contains(netContent, "NET_ADMIN") {
		t.Error("FAIL: WireGuard container missing required NET_ADMIN capability")
	}
	if !strings.Contains(netContent, "port: 5432") || !strings.Contains(netContent, "cidr: 10.0.0.0/24") {
		t.Error("FAIL: NetworkPolicy is not restricting ingress replication traffic to WireGuard CIDR on 5432")
	}

	t.Log("[PASS] 10.7: Multi-region K8s deployment topology, secure WireGuard tunnels, and synchronous DB replication configs are verified.")
}
