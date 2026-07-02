# ADR 013: Decoupled gRPC-Gateway Deployment Topology

## Context & Problem Statement
Sovereign L1 exposes both a RESTful JSON API and a gRPC API to clients. The Cosmos SDK utilizes `grpc-gateway` to translate REST requests (HTTP/1.1 JSON) to internal gRPC calls (HTTP/2). Running `grpc-gateway` in-process or as a sidecar inside the main gRPC server pod poses risks: high REST traffic spikes can exhaust the pod's memory/CPU resources, causing the gRPC server itself to crash. We need to isolate this boundary.

## Decision & Design

1. **Independent K8s Deployment**: `grpc-gateway` must be packaged and deployed as a **separate Kubernetes Deployment** with its own Horizontal Pod Autoscaler (HPA).
2. **Gateway-to-Server Routing**:
   - Envoy Gateway proxies all public incoming REST routes (`/api/rest/*`) to the `grpc-gateway` Service.
   - The decoupled `grpc-gateway` instance forwards requests to the `module/api` gRPC server Service (`backend_grpc` upstream cluster in Envoy).
3. **Scaling Independence**:
   - The REST translator (`grpc-gateway`) can scale dynamically based on HTTP request counts without impacting the core gRPC server.
   - The core `module/api` gRPC server scales based on active streaming connections and CPU utilization.
