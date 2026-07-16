# Mainnet Validator Onboarding Playbook

Welcome to the Sovereign L1 mainnet validator onboarding playbook. Before participating in the network, all validators must satisfy the system criteria and follow these secure coordination steps.

## 1. Node Hardware Requirements

| Specification | Minimum Requirement | Recommended Specification |
|:---|:---|:---|
| **CPU** | 4 Cores (x86_64 or ARM64) | 8 Cores (AMD EPYC/Intel Xeon) |
| **RAM** | 16 GB DDR4 | 32 GB DDR4/DDR5 |
| **Storage** | 500 GB NVMe SSD | 1 TB NVMe SSD (High IOPS) |
| **Network** | 100 Mbps symmetrical | 1 Gbps symmetrical |
| **OS** | Linux (Ubuntu 22.04 LTS / Debian 12) | Linux (Ubuntu 22.04 LTS) |

## 2. Validator Liveness SLA & Missed-Block Penalties

- **Target block finality**: ≤ 5.0 seconds.
- **Uptime SLA**: ≥ 99.9%.
- **Downtime penalty (Jailing)**: Jail window is 10,000 blocks. If a validator misses more than 50% of blocks in any window (5,000 blocks), they are jailed and their delegation is slashed by 0.01%.
- **Double-signing penalty**: Immediate tombstoning and 5.00% delegation slashing.

## 3. Remote Signer Infrastructure (Horcrux 2-of-3 Threshold)

To prevent single points of failure and double-signing events, validators MUST set up a threshold signature consensus cluster using Horcrux.
- Do not run the validator key directly on a single node.
- Split the private key into 3 shards and distribute them across 3 cosigner nodes.
- Each cosigner node must run behind a WireGuard VPN tunnel with no direct public SSH exposure.
- Configure `double-sign-protection = true` in all TOML configs.

## 4. Onboarding Instructions

### Step 4.1: Key Generation
Initialize your validator ledger/local key:
```bash
chaind keys add <moniker>-operator --keyring-backend file
```

### Step 4.2: Node Initialization
Initialize config files with the mainnet chain ID (`sovereign-1`):
```bash
chaind init <moniker> --chain-id sovereign-1
```

### Step 4.3: Create Genesis Transaction (Gentx)
Generate your genesis staking transaction. You must stake a minimum of `1,000,000,000 ucsov` (1,000 CSOV) to be eligible:
```bash
chaind gentx <moniker>-operator 1000000000ucsov \
  --pubkey $(chaind tendermint show-validator) \
  --chain-id sovereign-1 \
  --moniker "<moniker>"
```

### Step 4.4: PR Submission
Submit your gentx file to the Sovereign Foundation registry under `infra/mainnet/gentxs/` for final genesis integration.

## 5. Monitoring & Metrics

Validators must expose their Prometheus metrics endpoint for cluster health checks.
- Expose Tendermint metrics on `localhost:26660/metrics`.
- Expose app-specific telemetry on `localhost:1317/metrics`.
- Configure alerts for memory consumption, block latency, and missed signatures.
