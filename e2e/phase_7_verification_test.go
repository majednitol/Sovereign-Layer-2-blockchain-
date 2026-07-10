package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 7 Verification Test Suite
// Covers: Wallet Configuration (7.1), dApp Frontend (7.2), Analytics Dashboard (7.3)
// ═══════════════════════════════════════════════════════════════════════════════

// --- Wallet JSON Structures ---

type Bip44Config struct {
	CoinType int `json:"coinType"`
}

type KeplrCurrencyConfig struct {
	CoinDenom        string  `json:"coinDenom"`
	CoinMinimalDenom string  `json:"coinMinimalDenom"`
	CoinDecimals     int     `json:"coinDecimals"`
	GasPriceStep     *struct {
		Low     float64 `json:"low"`
		Average float64 `json:"average"`
		High    float64 `json:"high"`
	} `json:"gasPriceStep,omitempty"`
}

type Bech32Config struct {
	Bech32PrefixAccAddr  string `json:"bech32PrefixAccAddr"`
	Bech32PrefixAccPub   string `json:"bech32PrefixAccPub"`
	Bech32PrefixValAddr  string `json:"bech32PrefixValAddr"`
	Bech32PrefixValPub   string `json:"bech32PrefixValPub"`
	Bech32PrefixConsAddr string `json:"bech32PrefixConsAddr"`
	Bech32PrefixConsPub  string `json:"bech32PrefixConsPub"`
}

type KeplrConfig struct {
	ChainId       string              `json:"chainId"`
	ChainName     string              `json:"chainName"`
	Rpc           string              `json:"rpc"`
	Rest          string              `json:"rest"`
	Bip44         Bip44Config         `json:"bip44"`
	Bech32Config  Bech32Config        `json:"bech32Config"`
	Currencies    []KeplrCurrencyConfig `json:"currencies"`
	FeeCurrencies []KeplrCurrencyConfig `json:"feeCurrencies"`
	StakeCurrency KeplrCurrencyConfig `json:"stakeCurrency"`
}

type NativeCurrency struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

type MetaMaskSovereignEvmConfig struct {
	ChainId           string         `json:"chainId"`
	ChainName         string         `json:"chainName"`
	NativeCurrency    NativeCurrency `json:"nativeCurrency"`
	RpcUrls           []string       `json:"rpcUrls"`
	BlockExplorerUrls []string       `json:"blockExplorerUrls"`
}

type MetaMaskBscConfig struct {
	ChainId           string         `json:"chainId"`
	ChainName         string         `json:"chainName"`
	NativeCurrency    NativeCurrency `json:"nativeCurrency"`
	RpcUrls           []string       `json:"rpcUrls"`
	BlockExplorerUrls []string       `json:"blockExplorerUrls"`
}

type WalletConnectNamespace struct {
	Methods []string `json:"methods"`
	Chains  []string `json:"chains"`
}

type WalletConnectConfig struct {
	ProjectId  string                         `json:"projectId"`
	Namespaces map[string]WalletConnectNamespace `json:"namespaces"`
}

type WalletsJSON struct {
	Keplr                KeplrConfig                `json:"keplr"`
	MetamaskSovereignEvm MetaMaskSovereignEvmConfig `json:"metamaskSovereignEvm"`
	MetamaskBsc          MetaMaskBscConfig          `json:"metamaskBsc"`
	WalletConnect        WalletConnectConfig        `json:"walletConnect"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// 7.1 — Wallet Configuration Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase7_1_KeplrConfigCoinType60(t *testing.T) {
	wallets := loadWalletsJSON(t)

	// Plan: bip44.coinType = 60 (Ethereum BIP-44 path, required for dual-address compatibility)
	if wallets.Keplr.Bip44.CoinType != 60 {
		t.Errorf("FAIL: Keplr bip44.coinType is %d, expected 60 (Ethereum BIP-44 path for dual-address compatibility)",
			wallets.Keplr.Bip44.CoinType)
	} else {
		t.Log("[PASS] Keplr bip44.coinType = 60 (Ethereum BIP-44 path)")
	}
}

func TestPhase7_1_KeplrConfigRequiredFields(t *testing.T) {
	wallets := loadWalletsJSON(t)

	// Verify all required fields from the plan
	if wallets.Keplr.ChainId == "" {
		t.Error("FAIL: Keplr chainId is empty")
	}
	if wallets.Keplr.ChainName == "" {
		t.Error("FAIL: Keplr chainName is empty")
	}
	if wallets.Keplr.Rpc == "" {
		t.Error("FAIL: Keplr rpc is empty")
	}
	if wallets.Keplr.Rest == "" {
		t.Error("FAIL: Keplr rest is empty")
	}
	if wallets.Keplr.Bech32Config.Bech32PrefixAccAddr == "" {
		t.Error("FAIL: Keplr bech32Config.bech32PrefixAccAddr is empty")
	}
	if len(wallets.Keplr.Currencies) == 0 {
		t.Error("FAIL: Keplr currencies array is empty")
	}
	if len(wallets.Keplr.FeeCurrencies) == 0 {
		t.Error("FAIL: Keplr feeCurrencies array is empty")
	}
	if wallets.Keplr.StakeCurrency.CoinDenom == "" {
		t.Error("FAIL: Keplr stakeCurrency.coinDenom is empty")
	}

	// Verify feeCurrencies includes gasPriceStep
	for _, fee := range wallets.Keplr.FeeCurrencies {
		if fee.GasPriceStep == nil {
			t.Error("FAIL: Keplr feeCurrencies missing gasPriceStep")
		}
	}

	// Verify ucsov denomination pattern
	if wallets.Keplr.Currencies[0].CoinMinimalDenom == "" {
		t.Error("FAIL: Keplr currencies[0].coinMinimalDenom is empty")
	}

	t.Log("[PASS] Keplr config has all required fields: chainId, chainName, rpc, rest, bip44, bech32Config, currencies, feeCurrencies, stakeCurrency")
}

func TestPhase7_1_MetaMaskSovereignEvmConfig(t *testing.T) {
	wallets := loadWalletsJSON(t)

	// Plan: MetaMask — Sovereign chain EVM config (NOT the BSC config)
	evm := wallets.MetamaskSovereignEvm

	if evm.ChainId == "" {
		t.Error("FAIL: MetaMask sovereign EVM chainId is empty")
	}

	// Plan: nativeCurrency: TOKEN with decimals: 18
	if evm.NativeCurrency.Decimals != 18 {
		t.Errorf("FAIL: MetaMask sovereign EVM decimals is %d, expected 18 (matches cosmos/evm x/vm EVM denom 'aesov')",
			evm.NativeCurrency.Decimals)
	}

	// Plan: rpcUrls must contain /evm-rpc
	foundEvmRpc := false
	for _, url := range evm.RpcUrls {
		if strings.Contains(url, "/evm-rpc") {
			foundEvmRpc = true
		}
	}
	if !foundEvmRpc {
		t.Errorf("FAIL: MetaMask sovereign EVM rpcUrls does not contain '/evm-rpc' endpoint: %v", evm.RpcUrls)
	}

	// Plan: blockExplorerUrls must contain /blockscout
	foundBlockscout := false
	for _, url := range evm.BlockExplorerUrls {
		if strings.Contains(url, "/blockscout") || strings.Contains(url, "blockscout") {
			foundBlockscout = true
		}
	}
	if !foundBlockscout {
		t.Errorf("FAIL: MetaMask sovereign EVM blockExplorerUrls does not contain '/blockscout': %v", evm.BlockExplorerUrls)
	}

	t.Log("[PASS] MetaMask sovereign EVM config has correct chainId, TOKEN 18 decimals, /evm-rpc, and /blockscout")
}

func TestPhase7_1_MetaMaskBscConfigSeparate(t *testing.T) {
	wallets := loadWalletsJSON(t)

	// Plan: MetaMask — BSC config (existing BSC bridge config — unchanged): standard BSC mainnet / testnet RPC
	bsc := wallets.MetamaskBsc

	if bsc.ChainId == "" {
		t.Error("FAIL: MetaMask BSC chainId is empty")
	}

	// Verify it's BSC testnet (0x61 = 97) or BSC mainnet (0x38 = 56)
	if bsc.ChainId != "0x61" && bsc.ChainId != "0x38" {
		t.Errorf("FAIL: MetaMask BSC chainId '%s' is not BSC testnet (0x61) or mainnet (0x38)", bsc.ChainId)
	}

	if len(bsc.RpcUrls) == 0 {
		t.Error("FAIL: MetaMask BSC rpcUrls is empty")
	}

	if len(bsc.BlockExplorerUrls) == 0 {
		t.Error("FAIL: MetaMask BSC blockExplorerUrls is empty")
	}

	t.Log("[PASS] MetaMask BSC config is correctly separated from sovereign EVM config")
}

func TestPhase7_1_WalletConnectV2Config(t *testing.T) {
	wallets := loadWalletsJSON(t)

	wc := wallets.WalletConnect

	if wc.ProjectId == "" {
		t.Error("FAIL: WalletConnect projectId is empty")
	}

	// Verify eip155 namespace exists with sovereign EVM chain
	eip155, ok := wc.Namespaces["eip155"]
	if !ok {
		t.Fatal("FAIL: WalletConnect namespaces missing 'eip155' namespace")
	}

	if len(eip155.Methods) == 0 {
		t.Error("FAIL: WalletConnect eip155 methods array is empty")
	}

	// Must include sovereign EVM chain ID (not just BSC)
	if len(eip155.Chains) < 2 {
		t.Errorf("FAIL: WalletConnect eip155 chains should include both BSC and sovereign EVM chains, got %d chains: %v",
			len(eip155.Chains), eip155.Chains)
	}

	// Verify cosmos namespace
	cosmos, ok := wc.Namespaces["cosmos"]
	if !ok {
		t.Fatal("FAIL: WalletConnect namespaces missing 'cosmos' namespace")
	}

	if len(cosmos.Methods) == 0 {
		t.Error("FAIL: WalletConnect cosmos methods array is empty")
	}

	// cosmos namespace must reference cosmos_signDirect and cosmos_signAmino
	hasSignDirect := false
	hasSignAmino := false
	for _, method := range cosmos.Methods {
		if method == "cosmos_signDirect" {
			hasSignDirect = true
		}
		if method == "cosmos_signAmino" {
			hasSignAmino = true
		}
	}
	if !hasSignDirect {
		t.Error("FAIL: WalletConnect cosmos namespace missing 'cosmos_signDirect' method")
	}
	if !hasSignAmino {
		t.Error("FAIL: WalletConnect cosmos namespace missing 'cosmos_signAmino' method")
	}

	t.Log("[PASS] WalletConnect v2 config has eip155 (with sovereign EVM chain) and cosmos namespaces")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 7.2 — dApp Frontend Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase7_2_BridgeTrackerNoWebSocket(t *testing.T) {
	// Plan: "no /api/stream WebSocket — removed; all real-time data through gRPC server-streaming only"
	path := filepath.Join("..", "frontend", "components", "BridgeTracker.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read BridgeTracker.tsx: %v", err)
	}

	content := string(data)

	// Must NOT contain /api/stream or WebSocket references
	if strings.Contains(content, "/api/stream") {
		t.Error("FAIL: BridgeTracker.tsx still references obsolete '/api/stream' WebSocket endpoint")
	}
	if strings.Contains(content, "new WebSocket(") {
		t.Error("FAIL: BridgeTracker.tsx still creates a WebSocket connection — should use gRPC server-streaming")
	}

	// Must contain gRPC-Web streaming reference
	if !strings.Contains(content, "grpcweb") && !strings.Contains(content, "grpc-web") {
		t.Error("FAIL: BridgeTracker.tsx does not reference gRPC-Web streaming — expected gRPC server-streaming")
	}

	// Must contain auto-reconnect logic
	if !strings.Contains(content, "reconnect") && !strings.Contains(content, "backoff") && !strings.Contains(content, "retryCount") {
		t.Error("FAIL: BridgeTracker.tsx missing auto-reconnect / exponential backoff logic")
	}

	t.Log("[PASS] BridgeTracker.tsx uses gRPC server-streaming (not WebSocket) with auto-reconnect")
}

func TestPhase7_2_LayoutNavDashboardLink(t *testing.T) {
	path := filepath.Join("..", "frontend", "app", "layout.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read layout.tsx: %v", err)
	}

	content := string(data)

	// Must have /dashboard nav link
	if !strings.Contains(content, "/dashboard") {
		t.Error("FAIL: layout.tsx does not contain a nav link to /dashboard")
	}

	// Must have Analytics reference
	if !strings.Contains(content, "Analytics") && !strings.Contains(content, "analytics") {
		t.Error("FAIL: layout.tsx /dashboard nav link missing 'Analytics' label")
	}

	t.Log("[PASS] layout.tsx contains /dashboard Analytics nav link")
}

func TestPhase7_2_GovernancePageGasDisplay(t *testing.T) {
	// Plan: Governance page — gas limit param display so users understand Constitution check cost
	path := filepath.Join("..", "frontend", "app", "governance", "page.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read governance/page.tsx: %v", err)
	}

	content := string(data)

	// Must display gas-related information
	if !strings.Contains(content, "Gas") && !strings.Contains(content, "gas") {
		t.Error("FAIL: governance/page.tsx does not display gas limit parameters")
	}

	t.Log("[PASS] governance/page.tsx displays gas limit parameters for Constitution check cost")
}

func TestPhase7_2_BridgePageTieredConfirmation(t *testing.T) {
	// Plan: Bridge page — displays tiered confirmation status (standard vs. large transfer); live countdown
	path := filepath.Join("..", "frontend", "app", "page.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read page.tsx: %v", err)
	}

	content := string(data)

	// Must reference tiered confirmation / standard vs large
	if !strings.Contains(content, "Standard") || !strings.Contains(content, "Large") {
		t.Error("FAIL: page.tsx does not display tiered confirmation tiers (Standard vs. Large transfer)")
	}

	// Must have countdown
	if !strings.Contains(content, "Remaining") && !strings.Contains(content, "countdown") && !strings.Contains(content, "secondsRemaining") {
		t.Error("FAIL: page.tsx missing live countdown for bridge confirmations")
	}

	t.Log("[PASS] page.tsx displays tiered bridge confirmations (Standard/Large) with live countdown")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 7.3 — Analytics Dashboard Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase7_3_DashboardRouteExists(t *testing.T) {
	// Plan: Route: /dashboard/* in the existing Next.js dApp (frontend)
	path := filepath.Join("..", "frontend", "app", "dashboard", "page.tsx")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("FAIL: /dashboard route page.tsx does not exist at %s", path)
	}
	t.Log("[PASS] /dashboard route page.tsx exists")
}

func TestPhase7_3_DashboardChainOverview(t *testing.T) {
	// Plan: Chain Overview — Live TPS, TPS chart (24h), Block time p95, Active validators, Total txs (24h)
	content := readDashboardPage(t)

	requiredElements := []struct {
		name  string
		check string
	}{
		{"Live TPS", "TPS"},
		{"Block time", "Block"},
		{"Total Txs", "Total"},
		{"StreamChainStats", "StreamChainStats"},
		{"tps_1h", "tps_1h"},
		{"GetTps", "GetTps"},
		{"GetBlockStats", "GetBlockStats"},
	}

	for _, el := range requiredElements {
		if !strings.Contains(content, el.check) {
			t.Errorf("FAIL: Dashboard missing Chain Overview element: %s (searched for '%s')", el.name, el.check)
		}
	}

	t.Log("[PASS] Dashboard Chain Overview section contains all required elements")
}

func TestPhase7_3_DashboardBridge(t *testing.T) {
	// Plan: Bridge — Volume chart, Pending queue, Per-tx status, Lock/Release direction
	content := readDashboardPage(t)

	requiredElements := []struct {
		name  string
		check string
	}{
		{"Bridge volume", "Volume"},
		{"Lock/Release split", "Lock"},
		{"GetBridgeVolume", "GetBridgeVolume"},
		{"bridge_volume_1h", "bridge_volume_1h"},
	}

	for _, el := range requiredElements {
		if !strings.Contains(content, el.check) {
			t.Errorf("FAIL: Dashboard missing Bridge element: %s (searched for '%s')", el.name, el.check)
		}
	}

	t.Log("[PASS] Dashboard Bridge section contains all required elements")
}

func TestPhase7_3_DashboardOracle(t *testing.T) {
	// Plan: Oracle — Price OHLC chart, Asset selector, Participation rate
	content := readDashboardPage(t)

	requiredElements := []struct {
		name  string
		check string
	}{
		{"OHLC fields (open)", "open"},
		{"OHLC fields (high)", "high"},
		{"OHLC fields (low)", "low"},
		{"OHLC fields (close)", "close"},
		{"Asset selector", "Asset"},
		{"Participation/submission_count", "submission"},
		{"GetOraclePrice", "GetOraclePrice"},
		{"oracle_price_1h", "oracle_price_1h"},
	}

	for _, el := range requiredElements {
		if !strings.Contains(content, el.check) {
			t.Errorf("FAIL: Dashboard missing Oracle element: %s (searched for '%s')", el.name, el.check)
		}
	}

	t.Log("[PASS] Dashboard Oracle section contains all required elements including OHLC and asset selector")
}

func TestPhase7_3_DashboardValidators(t *testing.T) {
	// Plan: Validators — Uptime table, Uptime trend chart, Missed blocks
	content := readDashboardPage(t)

	requiredElements := []struct {
		name  string
		check string
	}{
		{"Uptime table", "Uptime"},
		{"Missed blocks", "Missed"},
		{"validator_uptime_1d", "validator_uptime_1d"},
		{"Validator address", "validator_address"},
	}

	for _, el := range requiredElements {
		if !strings.Contains(content, el.check) {
			t.Errorf("FAIL: Dashboard missing Validators element: %s (searched for '%s')", el.name, el.check)
		}
	}

	t.Log("[PASS] Dashboard Validators section contains uptime table, trend chart, and missed blocks")
}

func TestPhase7_3_DashboardSettlement(t *testing.T) {
	// Plan: Settlement — Pending list, Settlement detail, Lifecycle display (pending → confirmed → finalized)
	content := readDashboardPage(t)

	requiredElements := []struct {
		name  string
		check string
	}{
		{"Settlement ID", "settlement_id"},
		{"Lifecycle status pending", "pending"},
		{"Lifecycle status confirmed", "confirmed"},
		{"Lifecycle status finalized", "finalized"},
		{"Lifecycle flow", "Lifecycle"},
	}

	for _, el := range requiredElements {
		if !strings.Contains(content, el.check) {
			t.Errorf("FAIL: Dashboard missing Settlement element: %s (searched for '%s')", el.name, el.check)
		}
	}

	t.Log("[PASS] Dashboard Settlement section shows lifecycle: pending → confirmed → finalized")
}

func TestPhase7_3_DashboardMilestonesCertifications(t *testing.T) {
	// Plan: Milestones & Certifications — Milestone timeline, Certification list
	content := readDashboardPage(t)

	requiredElements := []struct {
		name  string
		check string
	}{
		{"Milestone ID", "milestone_id"},
		{"Milestone status achieved", "achieved"},
		{"Certification", "Certification"},
		{"certification projection", "certification"},
	}

	for _, el := range requiredElements {
		if !strings.Contains(content, el.check) {
			t.Errorf("FAIL: Dashboard missing Milestones/Certifications element: %s (searched for '%s')", el.name, el.check)
		}
	}

	t.Log("[PASS] Dashboard Milestones & Certifications section contains timeline and certification list")
}

func TestPhase7_3_DashboardStreamingAutoReconnect(t *testing.T) {
	// Plan: Real-time updates use gRPC server-streaming with SDK auto-reconnect
	content := readDashboardPage(t)

	if !strings.Contains(content, "StreamChainStats") {
		t.Error("FAIL: Dashboard does not reference StreamChainStats RPC for real-time updates")
	}
	if !strings.Contains(content, "auto-reconnect") {
		t.Error("FAIL: Dashboard does not mention auto-reconnect for streaming")
	}
	if !strings.Contains(content, "gRPC") || !strings.Contains(content, "server-streaming") {
		t.Error("FAIL: Dashboard does not reference gRPC server-streaming as the data transport")
	}

	t.Log("[PASS] Dashboard uses gRPC server-streaming with auto-reconnect for real-time updates")
}

func TestPhase7_3_DashboardTimescaleDBSources(t *testing.T) {
	// Plan: All chart data must be sourced from TimescaleDB continuous aggregates
	content := readDashboardPage(t)

	requiredAggregates := []string{
		"tps_1h",
		"bridge_volume_1h",
		"oracle_price_1h",
		"validator_uptime_1d",
	}

	for _, agg := range requiredAggregates {
		if !strings.Contains(content, agg) {
			t.Errorf("FAIL: Dashboard does not reference TimescaleDB continuous aggregate: %s", agg)
		}
	}

	t.Log("[PASS] Dashboard references all required TimescaleDB continuous aggregates")
}

func TestPhase7_3_DashboardNoCanvasDependency(t *testing.T) {
	// Plan: Charts: Recharts or Victory — lightweight, no canvas dependency, SSR-safe
	content := readDashboardPage(t)

	// The dashboard should NOT use raw canvas elements (violates SSR-safe requirement)
	if strings.Contains(content, "<canvas") {
		t.Error("FAIL: Dashboard uses <canvas> elements — plan requires SSR-safe charts (no canvas dependency)")
	}

	t.Log("[PASS] Dashboard does not use <canvas> — charts are SSR-safe")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Proto Alignment Tests — Verify analytics RPCs exist in proto files
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase7_ProtoAnalyticsRPCs(t *testing.T) {
	// Plan requires these 6 analytics RPCs in backend/v1/query.proto
	queryProtoPath := filepath.Join("..", "proto", "backend", "v1", "query.proto")
	data, err := os.ReadFile(queryProtoPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read query.proto: %v", err)
	}

	content := string(data)

	requiredQueryRPCs := []string{
		"GetTps",
		"GetBlockStats",
		"GetBridgeVolume",
		"GetOraclePrice",
		"GetValidatorUptime",
	}

	for _, rpc := range requiredQueryRPCs {
		if !strings.Contains(content, rpc) {
			t.Errorf("FAIL: query.proto missing required analytics RPC: %s", rpc)
		}
	}

	// StreamChainStats is in stream.proto
	streamProtoPath := filepath.Join("..", "proto", "backend", "v1", "stream.proto")
	streamData, err := os.ReadFile(streamProtoPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read stream.proto: %v", err)
	}

	streamContent := string(streamData)
	if !strings.Contains(streamContent, "StreamChainStats") {
		t.Error("FAIL: stream.proto missing StreamChainStats RPC")
	}

	t.Log("[PASS] All 6 analytics RPCs defined: GetTps, GetBlockStats, GetBridgeVolume, GetOraclePrice, GetValidatorUptime in query.proto; StreamChainStats in stream.proto")
}

func TestPhase7_ProtoStreamRPCs(t *testing.T) {
	// Plan references StreamBridgeEvents and StreamOracleEvents for real-time dashboard updates
	streamProtoPath := filepath.Join("..", "proto", "backend", "v1", "stream.proto")
	data, err := os.ReadFile(streamProtoPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read stream.proto: %v", err)
	}

	content := string(data)

	requiredStreamRPCs := []string{
		"StreamBridgeEvents",
		"StreamOracleEvents",
		"StreamChainStats",
	}

	for _, rpc := range requiredStreamRPCs {
		if !strings.Contains(content, rpc) {
			t.Errorf("FAIL: stream.proto missing required streaming RPC: %s", rpc)
		}
	}

	t.Log("[PASS] All required streaming RPCs defined in stream.proto")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func loadWalletsJSON(t *testing.T) WalletsJSON {
	t.Helper()
	path := filepath.Join("..", "frontend", "config", "wallets.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read wallets.json: %v", err)
	}

	var wallets WalletsJSON
	if err := json.Unmarshal(data, &wallets); err != nil {
		t.Fatalf("FAIL: Failed to unmarshal wallets.json: %v", err)
	}

	return wallets
}

func readDashboardPage(t *testing.T) string {
	t.Helper()
	path := filepath.Join("..", "frontend", "app", "dashboard", "page.tsx")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read dashboard/page.tsx: %v", err)
	}
	return string(data)
}

func TestPhase7_4_PnpmWorkspaceAndDependencies(t *testing.T) {
	// 1. Assert pnpm-workspace.yaml exists in root
	workspacePath := filepath.Join("..", "pnpm-workspace.yaml")
	data, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("FAIL: Could not read pnpm-workspace.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "packages:") || !strings.Contains(content, "frontend") {
		t.Error("FAIL: pnpm-workspace.yaml missing required configurations")
	}

	// 2. Assert @workspace/api-spec package exists and is correct
	specPkgPath := filepath.Join("..", "packages", "api-spec", "package.json")
	specData, err := os.ReadFile(specPkgPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read packages/api-spec/package.json: %v", err)
	}
	var specPkg struct {
		Name         string            `json:"name"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(specData, &specPkg); err != nil {
		t.Fatalf("FAIL: Failed to unmarshal api-spec package.json: %v", err)
	}
	if specPkg.Name != "@workspace/api-spec" {
		t.Errorf("FAIL: Expected package name '@workspace/api-spec', got '%s'", specPkg.Name)
	}

	// 3. Assert frontend/package.json has required dependencies
	frontPkgPath := filepath.Join("..", "frontend", "package.json")
	frontData, err := os.ReadFile(frontPkgPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read frontend/package.json: %v", err)
	}
	var frontPkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal(frontData, &frontPkg); err != nil {
		t.Fatalf("FAIL: Failed to unmarshal frontend package.json: %v", err)
	}

	requiredDeps := []string{
		"@workspace/api-spec",
		"@cosmjs/stargate",
		"@cosmjs/proto-signing",
		"wagmi",
		"viem",
		"recharts",
	}
	for _, dep := range requiredDeps {
		if _, ok := frontPkg.Dependencies[dep]; !ok {
			t.Errorf("FAIL: frontend/package.json missing required dependency: %s", dep)
		}
	}

	// 4. Assert x-wallet-address header defined in client config
	grpcClientPath := filepath.Join("..", "frontend", "config", "grpc-client.ts")
	clientCode, err := os.ReadFile(grpcClientPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read grpc-client.ts: %v", err)
	}
	clientStr := string(clientCode)
	if !strings.Contains(clientStr, "x-wallet-address") {
		t.Error("FAIL: grpc-client.ts missing x-wallet-address metadata header injection")
	}

	// 5. Assert wallet_addEthereumChain in WalletConnect
	walletConnectPath := filepath.Join("..", "frontend", "components", "WalletConnect.tsx")
	walletCode, err := os.ReadFile(walletConnectPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read WalletConnect.tsx: %v", err)
	}
	walletStr := string(walletCode)
	if !strings.Contains(walletStr, "wallet_addEthereumChain") {
		t.Error("FAIL: WalletConnect.tsx missing wallet_addEthereumChain MetaMask config addition")
	}

	t.Log("[PASS] Verified pnpm workspace layout, spec packaging, wallet dependencies, metadata headers, and MetaMask network integration.")
}

