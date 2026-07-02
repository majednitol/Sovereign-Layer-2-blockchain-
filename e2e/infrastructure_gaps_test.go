package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Infrastructure Gaps Verification Test Suite
// Covers: Validator Node (I.1), Backend Node (I.2), NATS JetStream (I.3),
//         Database StatefulSets (I.4), Horcrux (I.5), gRPC-Gateway (I.6),
//         Helm charts (I.7), Terraform configs (I.8), Grafana provisioning (I.12),
//         GitHub Actions CI/CD (I.13), WAL / PITR (I.14)
// ═══════════════════════════════════════════════════════════════════════════════

func TestI1_ValidatorNodeManifest(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "validator-node.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read validator-node.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "role: validator") {
		t.Error("FAIL: validator-node.yaml missing role: validator label")
	}
	if !strings.Contains(content, "--p2p.pex=false") {
		t.Error("FAIL: validator-node.yaml does not disable peer exchange")
	}
	if !strings.Contains(content, "image: sovereign-l1/chain-node") {
		t.Error("FAIL: validator-node.yaml does not use sovereign-l1/chain-node image")
	}

	t.Log("[PASS] I.1: Validator Node deployment manifest is fully compliant.")
}

func TestI2_BackendDeploymentManifest(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "backend-deployment.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read backend-deployment.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "name: backend-ingestion") || !strings.Contains(content, "replicas: 1") {
		t.Error("FAIL: Ingestion deployment missing or does not enforce 1 replica (singleton)")
	}
	if !strings.Contains(content, "name: backend-projection") || !strings.Contains(content, "replicas: 2") {
		t.Error("FAIL: Projection deployment missing or does not define at least 2 replicas")
	}
	if !strings.Contains(content, "name: backend-api") || !strings.Contains(content, "replicas: 3") {
		t.Error("FAIL: API gateway deployment missing or does not define at least 3 replicas")
	}

	t.Log("[PASS] I.2: Backend services (ingestion, projection, api) K8s manifests verified successfully.")
}

func TestI3_NatsJetStreamStatefulSet(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "nats-statefulset.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read nats-statefulset.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "replicas: 3") {
		t.Error("FAIL: NATS StatefulSet does not configure 3 replicas")
	}
	if !strings.Contains(content, "-js") || !strings.Contains(content, "-sd") {
		t.Error("FAIL: NATS startup args missing JetStream (-js) or storage directory (-sd) parameters")
	}
	if !strings.Contains(content, "podAntiAffinity") {
		t.Error("FAIL: NATS StatefulSet does not declare podAntiAffinity scheduling rules")
	}

	t.Log("[PASS] I.3: NATS JetStream StatefulSet manifest conforms to HA layout requirements.")
}

func TestI4_DatabaseStatefulSet(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "database.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read database.yaml: %v", err)
	}

	content := string(data)
	requiredComponents := []string{
		"name: database-write",
		"name: database-read",
		"name: database-relayer",
		"volumeClaimTemplates",
		"storage: 100Gi",
		"storage: 50Gi",
	}

	for _, comp := range requiredComponents {
		if !strings.Contains(content, comp) {
			t.Errorf("FAIL: database.yaml missing required component/param: %s", comp)
		}
	}

	t.Log("[PASS] I.4: Write, Read, and Relayer databases are configured as StatefulSets with storage PVCs.")
}

func TestI5_HorcruxSignerStatefulSet(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "horcrux-signer.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read horcrux-signer.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "replicas: 3") {
		t.Error("FAIL: Horcrux signer StatefulSet does not configure 3 replicas")
	}
	if !strings.Contains(content, "name: secrets-volume") || !strings.Contains(content, "secretName: horcrux-shares") {
		t.Error("FAIL: Horcrux container does not mount secrets-volume sharing consensus keys")
	}

	t.Log("[PASS] I.5: Horcrux threshold signer K8s deployment topology is verified.")
}

func TestI6_GrpcGatewaySeparateDeployment(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "grpc-gateway.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read grpc-gateway.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "kind: HorizontalPodAutoscaler") {
		t.Error("FAIL: grpc-gateway.yaml is missing HPA (HorizontalPodAutoscaler) definition")
	}
	if !strings.Contains(content, "minReplicas: 2") || !strings.Contains(content, "maxReplicas: 10") {
		t.Error("FAIL: grpc-gateway HPA does not define minReplicas=2 or maxReplicas=10 bounds")
	}
	if !strings.Contains(content, "averageUtilization: 80") {
		t.Error("FAIL: grpc-gateway HPA target utilization is not set to 80%")
	}

	t.Log("[PASS] I.6: Separate gRPC-Gateway deployment and HorizontalPodAutoscaler verified.")
}

func TestI7_HelmChartVerification(t *testing.T) {
	chartPath := filepath.Join("..", "infra", "helm", "sovereign-chain", "Chart.yaml")
	chartData, err := os.ReadFile(chartPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read Chart.yaml: %v", err)
	}

	if !strings.Contains(string(chartData), "name: sovereign-chain") {
		t.Error("FAIL: Chart.yaml does not declare name: sovereign-chain")
	}

	valuesPath := filepath.Join("..", "infra", "helm", "sovereign-chain", "values.yaml")
	valuesData, err := os.ReadFile(valuesPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read values.yaml: %v", err)
	}

	valuesContent := string(valuesData)
	if !strings.Contains(valuesContent, "validator:") || !strings.Contains(valuesContent, "database:") {
		t.Error("FAIL: values.yaml missing required validator or database value configurations")
	}

	t.Log("[PASS] I.7: Helm Chart metadata and values structure validated.")
}

func TestI8_TerraformAWSConfig(t *testing.T) {
	mainPath := filepath.Join("..", "infra", "terraform", "main.tf")
	mainData, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read main.tf: %v", err)
	}

	mainContent := string(mainData)
	requiredResources := []string{
		"resource \"aws_vpc\" \"main\"",
		"resource \"aws_eks_cluster\" \"eks\"",
		"resource \"aws_db_instance\" \"rds\"",
		"resource \"aws_s3_bucket\" \"backups\"",
	}

	for _, res := range requiredResources {
		if !strings.Contains(mainContent, res) {
			t.Errorf("FAIL: main.tf does not declare required AWS resource: %s", res)
		}
	}

	// Verify outputs.tf
	outputPath := filepath.Join("..", "infra", "terraform", "outputs.tf")
	outputData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read outputs.tf: %v", err)
	}

	outputContent := string(outputData)
	requiredOutputs := []string{
		"output \"eks_endpoint\"",
		"output \"rds_host\"",
		"output \"s3_backup_bucket\"",
	}

	for _, out := range requiredOutputs {
		if !strings.Contains(outputContent, out) {
			t.Errorf("FAIL: outputs.tf does not export required value: %s", out)
		}
	}

	t.Log("[PASS] I.8: Terraform AWS resources provisioning configurations verified successfully.")
}

func TestI12_GrafanaProvisioningConfig(t *testing.T) {
	dsPath := filepath.Join("..", "infra", "monitoring", "grafana-provisioning", "datasources", "datasources.yaml")
	dsData, err := os.ReadFile(dsPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read datasources.yaml: %v", err)
	}

	if !strings.Contains(string(dsData), "name: Prometheus") || !strings.Contains(string(dsData), "name: TimescaleDB") {
		t.Error("FAIL: datasources.yaml missing Prometheus or TimescaleDB datasource definitions")
	}

	dashPath := filepath.Join("..", "infra", "monitoring", "grafana-provisioning", "dashboards", "dashboards.yaml")
	dashData, err := os.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("FAIL: Could not read dashboards.yaml: %v", err)
	}

	if !strings.Contains(string(dashData), "providers:") || !strings.Contains(string(dashData), "path: /etc/grafana/provisioning/dashboards") {
		t.Error("FAIL: dashboards.yaml does not declare standard dashboards provisioning provider paths")
	}

	t.Log("[PASS] I.12: Grafana datasource and dashboard provisioning rules are verified.")
}

func TestI14_PostgresWALArchiving(t *testing.T) {
	path := filepath.Join("..", "scripts", "pg_wal_archive.sh")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("FAIL: pg_wal_archive.sh does not exist: %v", err)
	}

	// Verify executable permission
	mode := info.Mode()
	if mode&0111 == 0 {
		t.Error("FAIL: pg_wal_archive.sh is not executable")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read pg_wal_archive.sh: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "WAL_PATH=") || !strings.Contains(content, "WAL_FILE=") {
		t.Error("FAIL: pg_wal_archive.sh does not bind WAL parameters")
	}
	if !strings.Contains(content, "db_backups/s3_mock/wal") && !strings.Contains(content, "aws s3 cp") {
		t.Error("FAIL: pg_wal_archive.sh does not configure backup S3 destination copy paths")
	}

	t.Log("[PASS] I.14: Database WAL Point-in-Time Recovery archive script is configured and executable.")
}

func TestI13_CiPipelineCdEnhancement(t *testing.T) {
	path := filepath.Join("..", ".github", "workflows", "ci.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read ci.yml: %v", err)
	}

	content := string(data)
	requiredChecks := []string{
		"helm lint",
		"terraform validate",
		"docker build",
		"docker push",
	}

	for _, check := range requiredChecks {
		if !strings.Contains(content, check) {
			t.Errorf("FAIL: ci.yml does not define required CI/CD checks step: %s", check)
		}
	}

	t.Log("[PASS] I.13: GitHub Actions pipeline successfully enhanced for Helm lint, Terraform validation, and CD Docker build simulation.")
}

func TestI9_WireGuardVPN(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "multi-region-network.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read multi-region-network.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "wireguard-tunnel") {
		t.Error("FAIL: Network DaemonSet is missing 'wireguard-tunnel' definition")
	}
	if !strings.Contains(content, "NET_ADMIN") {
		t.Error("FAIL: WireGuard container missing required NET_ADMIN capability")
	}
	if !strings.Contains(content, "port: 5432") || !strings.Contains(content, "cidr: 10.0.0.0/24") {
		t.Error("FAIL: Network Policy missing cross-region Postgres port or WireGuard subnet block")
	}

	t.Log("[PASS] I.9: WireGuard VPN cross-cluster network security and tunnel configuration verified.")
}

func TestI10_CertManagerMTLS(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "tls-certmanager.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read tls-certmanager.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "kind: ClusterIssuer") || !strings.Contains(content, "name: sovereign-ca-issuer") {
		t.Error("FAIL: tls-certmanager.yaml missing CA ClusterIssuer definition")
	}
	if !strings.Contains(content, "name: validator-node-cert") || !strings.Contains(content, "name: sentry-node-cert") {
		t.Error("FAIL: tls-certmanager.yaml missing validator-node or sentry-node Certificate specifications")
	}

	t.Log("[PASS] I.10: cert-manager mTLS configuration with local self-signed CA verified.")
}

func TestI11_GrafanaDashboardJSON(t *testing.T) {
	path := filepath.Join("..", "infra", "monitoring", "dashboards", "sovereign-l1-dashboard.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read sovereign-l1-dashboard.json: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Chain TPS and Block Interval") {
		t.Error("FAIL: Dashboard missing TPS panel")
	}
	if !strings.Contains(content, "Bridge Volume Inbound/Outbound") {
		t.Error("FAIL: Dashboard missing Bridge Volume panel")
	}
	if !strings.Contains(content, "Oracle Asset Price updates") {
		t.Error("FAIL: Dashboard missing Oracle Price panel")
	}
	if !strings.Contains(content, "Validator Uptime and Missed Blocks") {
		t.Error("FAIL: Dashboard missing Validator Uptime panel")
	}

	t.Log("[PASS] I.11: Grafana dashboard panels for TPS, Bridge, Oracle, and Validator metrics verified.")
}

func TestI15_PostgresFailover(t *testing.T) {
	path := filepath.Join("..", "infra", "k8s", "multi-region-database.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("FAIL: Could not read multi-region-database.yaml: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "PATRONI_NAME") || !strings.Contains(content, "PATRONI_BOOTSTRAP_DCS") {
		t.Error("FAIL: Patroni environment variables not configured in database StatefulSet")
	}
	if !strings.Contains(content, "synchronous_commit: \"on\"") {
		t.Error("FAIL: Patroni synchronous_commit is not forced to 'on'")
	}
	if !strings.Contains(content, "synchronous_standby_names") {
		t.Error("FAIL: Patroni synchronous_standby_names not configured")
	}

	t.Log("[PASS] I.15: Patroni automatic failover and synchronous replication verified in database StatefulSet.")
}

