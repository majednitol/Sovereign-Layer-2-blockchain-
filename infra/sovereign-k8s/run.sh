#!/usr/bin/env bash

# Exit immediately if a command exits with a non-zero status
set -e

# Load configurations
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/config.env" ]; then
    echo "Loading config.env..."
    source "$SCRIPT_DIR/config.env"
else
    echo "Error: config.env not found!"
    exit 1
fi

echo "============================================="
echo "Starting Sovereign K8s Ordered Deployment"
echo "Namespace: $NAMESPACE"
echo "Registry:  $DOCKER_REGISTRY"
echo "============================================="

# Create Namespace
echo "Checking namespace: $NAMESPACE"
kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || kubectl create namespace "$NAMESPACE"

# Helper function to wait for deployment rollout
wait_for_deployment() {
    local name=$1
    echo "Waiting for deployment/$name to be ready..."
    kubectl rollout status deployment/"$name" -n "$NAMESPACE" --timeout=120s
}

# Helper function to wait for statefulset rollout
wait_for_statefulset() {
    local name=$1
    echo "Waiting for statefulset/$name to be ready..."
    kubectl rollout status statefulset/"$name" -n "$NAMESPACE" --timeout=120s
}

# 1. Defer NFS storage setup
echo "Deploying [1.nfs] Persistent Volumes..."
kubectl apply -f "$SCRIPT_DIR/1.nfs/" -n "$NAMESPACE"
sleep 2

# 2. Vault Secret Management
echo "Deploying [2.vault] Secret Vault..."
kubectl apply -f "$SCRIPT_DIR/2.vault/" -n "$NAMESPACE"
wait_for_deployment "vault"

# 3. Databases
echo "Deploying [3.databases] DBs & PgBouncer..."
kubectl apply -f "$SCRIPT_DIR/3.databases/" -n "$NAMESPACE"
wait_for_statefulset "db-write"
wait_for_statefulset "db-read"
wait_for_statefulset "db-relayer"
wait_for_deployment "pgbouncer-read"

# 4. NATS JetStream Cluster
echo "Deploying [4.nats] NATS cluster..."
kubectl apply -f "$SCRIPT_DIR/4.nats/" -n "$NAMESPACE"
wait_for_statefulset "nats"

# 5. Chain Node (Validator, Sentry, Horcrux, Network policies)
echo "Generating dynamically genesis-config from chain/genesis.json..."
kubectl create configmap genesis-config --from-file=genesis.json="$SCRIPT_DIR/../../chain/genesis.json" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo "Deploying [5.chain-node] Chain components..."
kubectl apply -f "$SCRIPT_DIR/5.chain-node/" -n "$NAMESPACE"
wait_for_deployment "sovereign-sentry"
wait_for_deployment "sovereign-validator"
wait_for_statefulset "horcrux-signer"

# 6. Oracle
echo "Deploying [6.oracle] Oracle Aggregator..."
kubectl apply -f "$SCRIPT_DIR/6.oracle/oracle.yaml" -n "$NAMESPACE"
wait_for_deployment "oracle"

# 7. Relayer
echo "Deploying [7.relayer] BSC Relayer..."
kubectl apply -f "$SCRIPT_DIR/7.relayer/relayer.yaml" -n "$NAMESPACE"
wait_for_deployment "relayer"

# 8. Backend Ingestion
echo "Deploying [8.backend-ingestion] Ingestion Singleton..."
kubectl apply -f "$SCRIPT_DIR/8.backend-ingestion/" -n "$NAMESPACE"
wait_for_deployment "backend-ingestion"

# 9. Backend Projection
echo "Deploying [9.backend-projection] Projection Service..."
kubectl apply -f "$SCRIPT_DIR/9.backend-projection/" -n "$NAMESPACE"
wait_for_deployment "backend-projection"

# 10. Backend API & REST Gateway
echo "Deploying [10.backend-api] API Service..."
kubectl apply -f "$SCRIPT_DIR/10.backend-api/" -n "$NAMESPACE"
wait_for_deployment "backend-api"
wait_for_deployment "grpc-gateway"

# 11. Faucet
echo "Deploying [11.faucet] Token Faucet..."
kubectl apply -f "$SCRIPT_DIR/11.faucet/" -n "$NAMESPACE"
wait_for_deployment "faucet-service"

# 12. Explorers (Unified Custom Explorer)
echo "Deploying [12.explorers] block explorers..."
kubectl apply -f "$SCRIPT_DIR/12.explorers/" -n "$NAMESPACE"
wait_for_statefulset "explorer-redis"
wait_for_statefulset "explorer-indexer"
wait_for_deployment "explorer-api"
wait_for_deployment "explorer-frontend"

# 13. Envoy Gateway
echo "Generating dynamically envoy-config from infra/envoy.yaml..."
kubectl create configmap envoy-config --from-file=envoy.yaml="$SCRIPT_DIR/../envoy.yaml" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo "Deploying [13.envoy-gateway] API Envoy Gateway..."
kubectl apply -f "$SCRIPT_DIR/13.envoy-gateway/" -n "$NAMESPACE"
wait_for_deployment "envoy-gateway"

# 14. Monitoring
echo "Generating dynamically monitoring-config for Prometheus..."
kubectl create configmap monitoring-config --from-file=prometheus.yml="$SCRIPT_DIR/../monitoring/prometheus.yml" --from-file=alerts.rules.yml="$SCRIPT_DIR/../monitoring/alerts.rules.yml" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo "Generating dynamically Grafana datasources, provider and dashboards JSON configmaps..."
kubectl create configmap grafana-datasources-config --from-file=datasources.yaml="$SCRIPT_DIR/../monitoring/grafana-provisioning/datasources/datasources.yaml" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl create configmap grafana-dashboards-provider-config --from-file=dashboards.yaml="$SCRIPT_DIR/../monitoring/grafana-provisioning/dashboards/dashboards.yaml" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
kubectl create configmap grafana-dashboards-json-config --from-file="$SCRIPT_DIR/../monitoring/dashboards/" -n "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

echo "Deploying [14.monitoring] Grafana & Prometheus..."
kubectl apply -f "$SCRIPT_DIR/14.monitoring/" -n "$NAMESPACE"
wait_for_deployment "prometheus"
wait_for_deployment "grafana"

# 15. Ingress
echo "Deploying [15.ingress] Ingress Routing..."
kubectl apply -f "$SCRIPT_DIR/15.ingress/" -n "$NAMESPACE"

# 16. Frontend
echo "Deploying [16.frontend] Next.js frontend..."
kubectl apply -f "$SCRIPT_DIR/16.frontend/frontend.yaml" -n "$NAMESPACE"
wait_for_deployment "frontend"

echo "============================================="
echo "Sovereign K8s Setup Successfully Deployed!"
echo "============================================="
