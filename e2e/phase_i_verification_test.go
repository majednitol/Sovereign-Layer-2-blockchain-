package e2e

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Phase I Launch Readiness Verification Test Suite
// Covers: Liquidity provision, wallet setup, bridge UI hardening, listing applications,
//         monitoring rules, governance proposal, bug bounty, community launch.
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhaseI_1_DocumentationExistAndNotStubbed(t *testing.T) {
	docs := []string{
		"doc/mainnet/liquidity-provision-guide.md",
		"doc/mainnet/wallet-setup-guide.md",
		"doc/mainnet/listing-application.md",
		"doc/mainnet/community-launch-checklist.md",
		"doc/ops/bug-bounty-policy.md",
		"doc/governance/first-governance-proposal.md",
	}

	for _, relPath := range docs {
		absPath := filepath.Join("..", relPath)
		content, err := ioutil.ReadFile(absPath)
		if err != nil {
			t.Fatalf("FAIL: Could not read document %s: %v", relPath, err)
		}

		strContent := string(content)
		if len(strContent) < 100 {
			t.Errorf("FAIL: Document %s appears to be too short or empty", relPath)
		}

		// Ensure no placeholder or owner action keys remain
		if strings.Contains(strContent, "🔑 OWNER ACTION") || strings.Contains(strContent, "TODO") {
			t.Errorf("FAIL: Document %s contains unresolved '🔑 OWNER ACTION' or 'TODO' stubs", relPath)
		}
	}
	t.Log("[PASS] All required launch readiness documents exist and contain no placeholders.")
}

func TestPhaseI_2_WalletConfigurationValid(t *testing.T) {
	configPath := filepath.Join("..", "frontend/config/wallets.json")
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read wallets.json: %v", err)
	}

	var wallets map[string]interface{}
	if err := json.Unmarshal(content, &wallets); err != nil {
		t.Fatalf("FAIL: wallets.json is not valid JSON: %v", err)
	}

	bscMainnet, exists := wallets["metamaskBscMainnet"].(map[string]interface{})
	if !exists {
		t.Fatal("FAIL: metamaskBscMainnet configuration does not exist in wallets.json")
	}

	chainId, ok := bscMainnet["chainId"].(string)
	if !ok || chainId != "0x38" {
		t.Errorf("FAIL: metamaskBscMainnet chainId is '%s', expected '0x38'", chainId)
	}

	walletConnect, exists := wallets["walletConnect"].(map[string]interface{})
	if !exists {
		t.Fatal("FAIL: walletConnect configuration does not exist in wallets.json")
	}

	projectId, ok := walletConnect["projectId"].(string)
	if !ok || !strings.Contains(projectId, "OWNER_ACTION_REQUIRED") {
		t.Errorf("FAIL: walletConnect projectId is '%s', expected it to contain 'OWNER_ACTION_REQUIRED' flag", projectId)
	}

	t.Log("[PASS] Wallets config has correct mainnet chain IDs and WalletConnect flag.")
}

func TestPhaseI_3_AlertRulesAndDashboardValid(t *testing.T) {
	// 1. Verify Alerts
	alertsPath := filepath.Join("..", "infra/monitoring/alerts.rules.yml")
	content, err := ioutil.ReadFile(alertsPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read alerts.rules.yml: %v", err)
	}

	var alerts yaml.Node
	if err := yaml.Unmarshal(content, &alerts); err != nil {
		t.Fatalf("FAIL: alerts.rules.yml is not valid YAML: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "sovereign-mainnet-alerts") {
		t.Error("FAIL: alerts.rules.yml does not contain mainnet alert group 'sovereign-mainnet-alerts'")
	}
	if !strings.Contains(strContent, "LaunchBlockProductionStalled") {
		t.Error("FAIL: alerts.rules.yml is missing LaunchBlockProductionStalled alert rule")
	}
	if !strings.Contains(strContent, "LaunchValidatorMissing") {
		t.Error("FAIL: alerts.rules.yml is missing LaunchValidatorMissing alert rule")
	}
	if !strings.Contains(strContent, "LaunchBridgeInvariantDrift") {
		t.Error("FAIL: alerts.rules.yml is missing LaunchBridgeInvariantDrift alert rule")
	}

	// 2. Verify Dashboard
	dashPath := filepath.Join("..", "infra/monitoring/dashboards/phase-i-launch-dashboard.json")
	dashContent, err := ioutil.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read phase-i-launch-dashboard.json: %v", err)
	}

	var dashboard map[string]interface{}
	if err := json.Unmarshal(dashContent, &dashboard); err != nil {
		t.Fatalf("FAIL: dashboard.json is not valid JSON: %v", err)
	}

	title, ok := dashboard["title"].(string)
	if !ok || !strings.Contains(title, "Phase I Launch") {
		t.Errorf("FAIL: Dashboard title is '%s', expected to contain 'Phase I Launch'", title)
	}

	panels, ok := dashboard["panels"].([]interface{})
	if !ok || len(panels) == 0 {
		t.Errorf("FAIL: Dashboard has no panels or invalid panels list structure")
	}

	t.Log("[PASS] Alert rules and Grafana launch dashboard verified successfully.")
}

func TestPhaseI_4_ScriptExecutability(t *testing.T) {
	scriptPath := filepath.Join("..", "scripts/verify-lp-lock.sh")
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("FAIL: verify-lp-lock.sh does not exist: %v", err)
	}

	mode := info.Mode()
	if mode&0111 == 0 {
		t.Error("FAIL: verify-lp-lock.sh is not executable")
	}

	t.Log("[PASS] verify-lp-lock.sh script exists and has executable permissions set.")
}
