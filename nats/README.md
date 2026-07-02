# NATS JetStream System Configuration

This directory is reserved for holding NATS configurations, credentials, and access control files.

## NATS Configuration Overview
* **Local Devnet**: Set up as a 3-node cluster using the service configurations defined in the root [docker-compose.yml](file:///Users/majedurrahman/Sovereign/docker-compose.yml).
* **Production**: Deployed via Helm values and Kubernetes configurations defined inside `/infra/` using TLS, credential isolation, and NKey authentication.
* **Stream Namespaces**:
  - `bridge`: Bridge activity events (Solidity events monitored by watcher).
  - `chain`: Block production and custom module execution results.
  - `stream`: Final processed client projections.
