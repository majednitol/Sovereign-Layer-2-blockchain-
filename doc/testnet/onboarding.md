# Testnet Validator Onboarding Playbook (Milestone 6.6)

Welcome to the Sovereign L1 testnet onboarding playbook. 

To join the public testnet, operators must follow these steps. Please note that testnet uses different parameters (e.g., chain-id `sovereign-testnet-1` and EVM chain-id `7778`) to prevent any collision with mainnet key configurations.

---

## 1. Node Hardware Requirements

Testnet validators should run on configurations that closely match mainnet requirements to ensure performance metrics are realistic:
- **CPU**: 4 Cores minimum.
- **RAM**: 16 GB DDR4.
- **Storage**: 250 GB SSD (NVMe preferred).
- **OS**: Linux (Ubuntu 22.04 LTS recommended).

---

## 2. Onboarding Instructions

### Step 2.1: Key Generation
Generate your operator key on the testnet keychain:
```bash
chaind keys add <moniker>-test-operator --keyring-backend file
```

### Step 2.2: Node Initialization
Initialize the configuration files for the testnet environment:
```bash
chaind init <moniker> --chain-id sovereign-testnet-1
```

### Step 2.3: Request Faucet Funds
Since testnet requires staking tokens to propose blocks, request bootstrap tokens from the testnet faucet endpoint:
```bash
curl -X POST -d '{"address":"<your-cosmos-address>"}' https://api.testnet.sovereign.l1/faucet
```

### Step 2.4: Create Genesis Transaction (Gentx)
Create the validator genesis transaction:
```bash
chaind gentx <moniker>-test-operator 1000000000ucsov \
  --pubkey $(chaind tendermint show-validator) \
  --chain-id sovereign-testnet-1 \
  --moniker "<moniker>"
```

### Step 2.5: PR Submission
Create a pull request submitting the generated gentx file under the `infra/testnet/gentxs/` directory.

---

## 3. Peer Connectivity

Add testnet sentry node endpoints under `persistent_peers` in `config.toml`:
```toml
persistent_peers = "sentry-testnet-0@sentry-testnet-0.sovereign.l1:26656,sentry-testnet-1@sentry-testnet-1.sovereign.l1:26656"
```
