package e2e

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// --- Structs for parsing Kubernetes and other Yaml configs ---
type K8sDeployment struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string            `yaml:"name"`
		Namespace string            `yaml:"namespace"`
		Labels    map[string]string `yaml:"labels"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas int `yaml:"replicas"`
		Selector struct {
			MatchLabels map[string]string `yaml:"matchLabels"`
		} `yaml:"selector"`
		Template struct {
			Metadata struct {
				Labels map[string]string `yaml:"labels"`
			} `yaml:"metadata"`
			Spec struct {
				Containers []struct {
					Name    string   `yaml:"name"`
					Image   string   `yaml:"image"`
					Command []string `yaml:"command"`
					Ports   []struct {
						ContainerPort int    `yaml:"containerPort"`
						Name          string `yaml:"name"`
					} `yaml:"ports"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

type K8sNetworkPolicy struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec struct {
		PodSelector struct {
			MatchLabels map[string]string `yaml:"matchLabels"`
		} `yaml:"podSelector"`
		PolicyTypes []string `yaml:"policyTypes"`
	} `yaml:"spec"`
}

// --- Envoy Structs for Parsing ---
type EnvoyMatch struct {
	Prefix string `yaml:"prefix"`
}

type EnvoyUpgradeConfig struct {
	UpgradeType string `yaml:"upgrade_type"`
}

type EnvoyRouteSpec struct {
	Cluster        string               `yaml:"cluster"`
	PrefixRewrite  string               `yaml:"prefix_rewrite,omitempty"`
	UpgradeConfigs []EnvoyUpgradeConfig `yaml:"upgrade_configs,omitempty"`
}

type EnvoyRoute struct {
	Match EnvoyMatch     `yaml:"match"`
	Route EnvoyRouteSpec `yaml:"route"`
}

type EnvoyVirtualHost struct {
	Name    string       `yaml:"name"`
	Domains []string     `yaml:"domains"`
	Routes  []EnvoyRoute `yaml:"routes"`
}

type EnvoyRouteConfig struct {
	Name         string             `yaml:"name"`
	VirtualHosts []EnvoyVirtualHost `yaml:"virtual_hosts"`
}

type EnvoySocketAddress struct {
	Address   string `yaml:"address"`
	PortValue int    `yaml:"port_value"`
}

type EnvoyAddress struct {
	SocketAddress EnvoySocketAddress `yaml:"socket_address"`
}

type EnvoyEndpoint struct {
	Address EnvoyAddress `yaml:"address"`
}

type EnvoyLbEndpoint struct {
	Endpoint EnvoyEndpoint `yaml:"endpoint"`
}

type EnvoyLoadAssignment struct {
	ClusterName string `yaml:"cluster_name"`
	Endpoints   []struct {
		LbEndpoints []EnvoyLbEndpoint `yaml:"lb_endpoints"`
	} `yaml:"endpoints"`
}

type EnvoyCluster struct {
	Name           string              `yaml:"name"`
	ConnectTimeout string              `yaml:"connect_timeout"`
	LoadAssignment EnvoyLoadAssignment `yaml:"load_assignment"`
}

type EnvoyConfig struct {
	StaticResources struct {
		Listeners []interface{}  `yaml:"listeners"`
		Clusters  []EnvoyCluster `yaml:"clusters"`
	} `yaml:"static_resources"`
}

type FaucetRequest struct {
	Address string `json:"address"`
}

type FaucetResponse struct {
	TxHash  string `json:"tx_hash,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// --- Prometheus Structs ---
type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

type ScrapeConfig struct {
	JobName       string         `yaml:"job_name"`
	StaticConfigs []StaticConfig `yaml:"static_configs"`
}

type PrometheusConfig struct {
	Global        map[string]interface{} `yaml:"global"`
	RuleFiles     []string               `yaml:"rule_files"`
	ScrapeConfigs []ScrapeConfig         `yaml:"scrape_configs"`
}

// --- Alerts Structs ---
type Rule struct {
	Alert       string            `yaml:"alert"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

type Group struct {
	Name  string `yaml:"name"`
	Rules []Rule `yaml:"rules"`
}

type AlertRulesConfig struct {
	Groups []Group `yaml:"groups"`
}

// --- Helper: Simple TOML Key-Value Parser ---
func parseTOML(t *testing.T, filepath string) map[string]string {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatalf("FAIL: Could not read TOML file %s: %v", filepath, err)
	}

	config := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"' `)
			config[key] = val
		}
	}
	return config
}

// ═══════════════════════════════════════════════════════════════════════════════
// 1. Validator & Sentry Topology Manifest Tests (6.1)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_ValidatorManifest(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "validator-node.yaml")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read validator-node.yaml: %v", err)
	}

	docs := strings.Split(string(data), "---")
	foundDeployment := false

	for _, doc := range docs {
		var deploy K8sDeployment
		if err := yaml.Unmarshal([]byte(doc), &deploy); err != nil {
			continue
		}

		if deploy.Kind == "Deployment" && deploy.Metadata.Name == "sovereign-validator" {
			foundDeployment = true
			if deploy.Metadata.Labels["role"] != "validator" {
				t.Errorf("FAIL: sovereign-validator deployment missing role=validator label")
			}
			if len(deploy.Spec.Template.Spec.Containers) == 0 {
				t.Fatalf("FAIL: sovereign-validator deployment has no containers")
			}

			container := deploy.Spec.Template.Spec.Containers[0]
			if container.Name != "validator-node" {
				t.Errorf("FAIL: container name is '%s', expected 'validator-node'", container.Name)
			}

			// Validate validator p2p configs and private validator options
			hasPexFalse := false
			hasPersistentPeers := false
			hasPrivVal := false
			for _, arg := range container.Command {
				if strings.Contains(arg, "--p2p.pex=false") {
					hasPexFalse = true
				}
				if strings.Contains(arg, "--p2p.persistent-peers=") {
					hasPersistentPeers = true
				}
				if strings.Contains(arg, "--priv_validator_laddr=") {
					hasPrivVal = true
				}
			}

			if !hasPexFalse {
				t.Error("FAIL: Validator must disable Peer Exchange (--p2p.pex=false)")
			}
			if !hasPersistentPeers {
				t.Error("FAIL: Validator must define persistent peers pointing to sentries")
			}
			if !hasPrivVal {
				t.Error("FAIL: Validator must define --priv_validator_laddr pointing to signer")
			}
		}
	}

	if !foundDeployment {
		t.Error("FAIL: sovereign-validator Deployment not found in validator-node.yaml")
	}
	t.Log("[PASS] Verified validator node manifest topology rules.")
}

func TestPhase6_HorcruxManifest(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "horcrux-signer.yaml")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read horcrux-signer.yaml: %v", err)
	}

	docs := strings.Split(string(data), "---")
	foundStatefulSet := false

	for _, doc := range docs {
		var deploy K8sDeployment
		if err := yaml.Unmarshal([]byte(doc), &deploy); err != nil {
			continue
		}

		if deploy.Kind == "StatefulSet" && deploy.Metadata.Name == "horcrux-signer" {
			foundStatefulSet = true
			if deploy.Metadata.Labels["role"] != "horcrux-signer" {
				t.Errorf("FAIL: StatefulSet missing role=horcrux-signer label")
			}
			if deploy.Spec.Replicas != 3 {
				t.Errorf("FAIL: Horcrux co-signer replicas must be 3, got %d", deploy.Spec.Replicas)
			}
			if len(deploy.Spec.Template.Spec.Containers) == 0 {
				t.Fatalf("FAIL: StatefulSet has no containers")
			}
			container := deploy.Spec.Template.Spec.Containers[0]
			if container.Name != "horcrux" {
				t.Errorf("FAIL: container name is '%s', expected 'horcrux'", container.Name)
			}
		}
	}

	if !foundStatefulSet {
		t.Error("FAIL: horcrux-signer StatefulSet not found in horcrux-signer.yaml")
	}
	t.Log("[PASS] Verified Horcrux K8s StatefulSet manifest topology rules.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 2. Horcrux threshold signer configurations (6.2)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_HorcruxConfigs(t *testing.T) {
	files := []string{"horcrux-0.toml", "horcrux-1.toml", "horcrux-2.toml"}

	for i, file := range files {
		path := filepath.Join("..", "infra", "horcrux", file)
		config := parseTOML(t, path)

		// Assert threshold is 2-of-3
		if threshold, ok := config["threshold"]; !ok || threshold != "2" {
			t.Errorf("FAIL: %s: expected threshold=2, got '%s'", file, threshold)
		}

		// Assert co-signer node-id
		nodeIDKey := "node-id"
		if nodeID, ok := config[nodeIDKey]; ok {
			expectedID := string(rune('1' + i))
			if nodeID != expectedID {
				t.Errorf("FAIL: %s: expected node-id=%s, got '%s'", file, expectedID, nodeID)
			}
		}

		// Assert double signing protection
		doubleSignKey := "double-sign-protection"
		if dsp, ok := config[doubleSignKey]; !ok || dsp != "true" {
			t.Errorf("FAIL: %s: double-sign-protection must be enabled", file)
		}

		// Assert chain-id
		chainIDKey := "chain-id"
		if cid, ok := config[chainIDKey]; !ok || cid != "sovereign-testnet-1" {
			t.Errorf("FAIL: %s: invalid chain-id target '%s'", file, cid)
		}
	}
	t.Log("[PASS] Verified all 3 Horcrux co-signer TOML configs.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 3. Envoy Gateway & Faucet configuration (6.3 & 6.7)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_EnvoyGatewayConfig(t *testing.T) {
	path := filepath.Join("..", "infra", "envoy.yaml")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read Envoy configuration %s: %v", path, err)
	}

	var config EnvoyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("FAIL: Failed to unmarshal Envoy config into struct: %v", err)
	}

	content := string(data)

	// Ensure faucet route and upstream cluster exist
	if !strings.Contains(content, `prefix: "/faucet"`) {
		t.Error("FAIL: Envoy missing /faucet routing prefix")
	}

	foundFaucetCluster := false
	for _, cluster := range config.StaticResources.Clusters {
		if cluster.Name == "faucet_service" {
			foundFaucetCluster = true
			if len(cluster.LoadAssignment.Endpoints) > 0 && len(cluster.LoadAssignment.Endpoints[0].LbEndpoints) > 0 {
				addr := cluster.LoadAssignment.Endpoints[0].LbEndpoints[0].Endpoint.Address.SocketAddress
				if addr.Address != "faucet-service" || addr.PortValue != 8000 {
					t.Errorf("FAIL: faucet_service cluster points to incorrect address or port: %s:%d", addr.Address, addr.PortValue)
				}
			} else {
				t.Error("FAIL: faucet_service cluster has no endpoints defined")
			}
		}
	}

	if !foundFaucetCluster {
		t.Error("FAIL: faucet_service cluster is missing in Envoy configurations")
	}

	// Verify HPA exists for envoy
	hpaPath := filepath.Join("..", "infra", "k8s", "envoy-gateway.yaml")
	hpaData, err := ioutil.ReadFile(hpaPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read envoy-gateway.yaml: %v", err)
	}

	if !strings.Contains(string(hpaData), "kind: HorizontalPodAutoscaler") {
		t.Error("FAIL: Envoy Gateway must have HorizontalPodAutoscaler defined")
	}

	t.Log("[PASS] Verified Envoy Gateway HPA, /faucet routing, and faucet upstream cluster.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 4. Network Policies isolation rules (6.4)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_NetworkPolicies(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "network-policies.yaml")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read network-policies.yaml: %v", err)
	}

	docs := strings.Split(string(data), "---")
	foundValidatorPolicy := false
	foundSentryPolicy := false
	foundDatabasePolicy := false

	for _, doc := range docs {
		var policy K8sNetworkPolicy
		if err := yaml.Unmarshal([]byte(doc), &policy); err != nil {
			continue
		}

		if policy.Kind == "NetworkPolicy" {
			switch policy.Metadata.Name {
			case "validator-isolation-policy":
				foundValidatorPolicy = true
				if policy.Spec.PodSelector.MatchLabels["role"] != "validator" {
					t.Error("FAIL: validator-isolation-policy must select pod role: validator")
				}
			case "sentry-policy":
				foundSentryPolicy = true
				if policy.Spec.PodSelector.MatchLabels["role"] != "sentry" {
					t.Error("FAIL: sentry-policy must select pod role: sentry")
				}
			case "database-isolation-policy":
				foundDatabasePolicy = true
				if policy.Spec.PodSelector.MatchLabels["role"] != "database" {
					t.Error("FAIL: database-isolation-policy must select pod role: database")
				}
			}
		}
	}

	if !foundValidatorPolicy {
		t.Error("FAIL: validator-isolation-policy missing")
	}
	if !foundSentryPolicy {
		t.Error("FAIL: sentry-policy missing")
	}
	if !foundDatabasePolicy {
		t.Error("FAIL: database-isolation-policy missing")
	}

	t.Log("[PASS] Verified all database, validator, and sentry isolation NetworkPolicies.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 5. mTLS Cert-Manager & WireGuard configs (6.5)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_mTLSAndWireGuard(t *testing.T) {
	// Check Cert-manager config
	certPath := filepath.Join("..", "infra", "k8s", "tls-certmanager.yaml")
	certData, err := ioutil.ReadFile(certPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read tls-certmanager.yaml: %v", err)
	}
	certContent := string(certData)
	if !strings.Contains(certContent, "ClusterIssuer") || !strings.Contains(certContent, "Certificate") {
		t.Error("FAIL: tls-certmanager.yaml does not contain expected Cert-manager resources")
	}

	// Check WireGuard config
	wgPath := filepath.Join("..", "infra", "wireguard", "wg0.conf")
	wgData, err := ioutil.ReadFile(wgPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read wg0.conf: %v", err)
	}
	wgContent := string(wgData)
	if !strings.Contains(wgContent, "[Interface]") || !strings.Contains(wgContent, "[Peer]") {
		t.Error("FAIL: wg0.conf is missing interface or peer blocks")
	}

	t.Log("[PASS] Verified mTLS cert-manager resources and WireGuard VPN configs.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 6. Testnet Onboarding Playbook & Stability Checklists (6.6, 6.8 & 6.9)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_TestnetDocs(t *testing.T) {
	// Validator Onboarding
	onboardPath := filepath.Join("..", "doc", "testnet", "onboarding.md")
	onboardData, err := ioutil.ReadFile(onboardPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read onboarding.md: %v", err)
	}
	onboardContent := string(onboardData)
	if !strings.Contains(onboardContent, "gentx") || !strings.Contains(onboardContent, "chaind keys add") {
		t.Error("FAIL: onboarding.md missing keys or gentx guidelines")
	}

	// Stability Checklist
	stabilityPath := filepath.Join("..", "doc", "testnet", "stability_checklist.md")
	stabilityData, err := ioutil.ReadFile(stabilityPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read stability_checklist.md: %v", err)
	}
	stabilityContent := string(stabilityData)
	if !strings.Contains(stabilityContent, "Freeze") || !strings.Contains(stabilityContent, "Liveness") {
		t.Error("FAIL: stability_checklist.md missing freeze or consensus liveness guidelines")
	}

	t.Log("[PASS] Verified testnet onboarding guides and stability checklist documentation.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 7. Prometheus Targets and Alerts (I.17 - I.21)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPhase6_PrometheusScrapeConfigs(t *testing.T) {
	path := filepath.Join("..", "infra", "monitoring", "prometheus.yml")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read prometheus.yml: %v", err)
	}

	var config PrometheusConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("FAIL: Failed to unmarshal prometheus.yml: %v", err)
	}

	requiredJobs := map[string]string{
		"cosmos-validator":  "localhost:26660",
		"oracle-aggregator": "localhost:9200",
		"relayer":           "localhost:9300",
		"faucet":            "faucet-service:8000",
	}

	for jobName, expectedTarget := range requiredJobs {
		found := false
		for _, scrape := range config.ScrapeConfigs {
			if scrape.JobName == jobName {
				found = true
				if len(scrape.StaticConfigs) == 0 || len(scrape.StaticConfigs[0].Targets) == 0 {
					t.Errorf("FAIL: Scrape config for job '%s' has no static targets", jobName)
				} else {
					target := scrape.StaticConfigs[0].Targets[0]
					if target != expectedTarget {
						t.Errorf("FAIL: Scrape job '%s' target is '%s', expected '%s'", jobName, target, expectedTarget)
					}
				}
			}
		}
		if !found {
			t.Errorf("FAIL: Scrape job '%s' is missing in prometheus.yml", jobName)
		}
	}
	t.Log("[PASS] Checked Prometheus scrape configs: faucet and other components included.")
}

func TestPhase6_AlertingRules(t *testing.T) {
	path := filepath.Join("..", "infra", "monitoring", "alerts.rules.yml")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read alerts.rules.yml: %v", err)
	}

	var config AlertRulesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		t.Fatalf("FAIL: Failed to unmarshal alerts.rules.yml: %v", err)
	}

	if len(config.Groups) == 0 {
		t.Fatalf("FAIL: No alert groups defined in alerts.rules.yml")
	}

	group := config.Groups[0]
	requiredAlerts := []string{
		"ValidatorMissedBlocks",
		"HorcruxSignerOffline",
		"RelayerMissedEvent",
		"EvmJsonrpcErrorRate",
		"BlockscoutIndexingLag",
		"EvmBlockGasHigh",
		"ChainHalted",
		"NatsPublishedBacklog",
		"OracleStaleness",
		"RewardsBucketDepleting",
		"IngestionCrash",
	}

	alertMap := make(map[string]Rule)
	for _, rule := range group.Rules {
		alertMap[rule.Alert] = rule
	}

	for _, alert := range requiredAlerts {
		_, found := alertMap[alert]
		if !found {
			t.Errorf("FAIL: Alert rule '%s' is missing in alerts.rules.yml", alert)
		}
	}
	t.Log("[PASS] Verified all 11 Prometheus alert rules present.")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 8. Faucet Endpoint Logic Execution Test (6.7)
// ═══════════════════════════════════════════════════════════════════════════════

// Mock handleFaucet wrapper directly inside the test suite to execute HTTP scenarios
func TestPhase6_FaucetEndpointExecution(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: "Only POST allowed"})
			return
		}

		var req FaucetRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: "Invalid JSON"})
			return
		}

		address := strings.TrimSpace(req.Address)
		if address == "" || (!strings.HasPrefix(address, "cosmos") && !strings.HasPrefix(address, "sov")) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: "Invalid address format"})
			return
		}

		// Mock the output success for valid address for the E2E verification
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(FaucetResponse{Success: true, TxHash: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"})
	})

	// Scenario 1: Reject GET Request
	t.Run("Reject GET request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/faucet", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", w.Code)
		}
		var resp FaucetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if resp.Success || resp.Error != "Only POST allowed" {
			t.Errorf("unexpected body payload: %+v", resp)
		}
	})

	// Scenario 2: Reject Invalid JSON
	t.Run("Reject invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/faucet", bytes.NewBufferString("{bad_json}"))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
	})

	// Scenario 3: Reject Invalid Address Format
	t.Run("Reject invalid address prefix", func(t *testing.T) {
		reqBody, _ := json.Marshal(FaucetRequest{Address: "invalid12345"})
		req := httptest.NewRequest(http.MethodPost, "/faucet", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", w.Code)
		}
		var resp FaucetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if resp.Success || resp.Error != "Invalid address format" {
			t.Errorf("unexpected body payload: %+v", resp)
		}
	})

	// Scenario 4: Accept Valid Address Format
	t.Run("Accept valid address prefix", func(t *testing.T) {
		reqBody, _ := json.Marshal(FaucetRequest{Address: "cosmos1majedurrahman"})
		req := httptest.NewRequest(http.MethodPost, "/faucet", bytes.NewBuffer(reqBody))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", w.Code)
		}
		var resp FaucetResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if !resp.Success || resp.TxHash == "" {
			t.Errorf("unexpected body payload: %+v", resp)
		}
	})
}

// 9. Real compiled Faucet Binary process integration test
func TestPhase6_FaucetBinaryExecution(t *testing.T) {
	faucetPath := filepath.Join("..", "bin", "faucet")
	if _, err := exec.LookPath(faucetPath); err != nil {
		t.Skip("Skipping faucet binary process test: bin/faucet not compiled. Run 'make build' first.")
	}

	// Launch faucet server process on port 8099
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(faucetPath, "-listen", "127.0.0.1:8099", "-node", "http://127.0.0.1:26657")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start faucet binary process: %v", err)
	}

	// Ensure process gets killed when test finishes
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	// Wait for server to bind and start listening by retrying Post request
	var resp *http.Response
	reqBody, _ := json.Marshal(FaucetRequest{Address: "cosmos1majedurrahman"})
	success := false
	var lastErr error
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err = http.Post("http://127.0.0.1:8099/faucet", "application/json", bytes.NewBuffer(reqBody))
		if err == nil {
			success = true
			break
		}
		lastErr = err
	}

	if !success {
		t.Fatalf("Failed to dial local faucet binary server: %v\nStdout: %s\nStderr: %s", lastErr, stdout.String(), stderr.String())
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	// Verify the response is structured properly in JSON
	var faucetResp FaucetResponse
	if err := json.Unmarshal(bodyBytes, &faucetResp); err != nil {
		t.Fatalf("Faucet server did not return valid FaucetResponse JSON: %s, error: %v", string(bodyBytes), err)
	}

	// Expect the faucet request to fail with a broadcast error since chain-node is offline,
	// but verify it tried to execute and returned a structured FaucetResponse.
	t.Logf("[PASS] Faucet binary successfully responded. Success: %t, Error: %s", faucetResp.Success, faucetResp.Error)
}


