# Testnet Onboarding Playbook (Milestone 6.6)

Welcome to the Sovereign L1 testnet onboarding guide. External validators must perform the following actions:

## 1. Key Generation
Run the following to initialize your validator keys:
```bash
chaind keys add <operator-key-name>
```

## 2. Validator Node Initialization
Initialize the configuration directory:
```bash
chaind init <moniker> --chain-id sovereign-testnet-1
```

## 3. Generate Genesis Transaction (Gentx)
Create the validator genesis transaction:
```bash
chaind gentx <operator-key-name> 100000000000usov \
  --pubkey $(chaind tendermint show-validator) \
  --chain-id sovereign-testnet-1 \
  --moniker "<moniker>"
```

## 4. Onboarding Submission
Create a pull request adding the generated gentx file under the `infra/testnet/gentxs/` directory.

## 5. Peer Configuration
Add sentry node endpoints under `persistent_peers` in `config.toml`:
`persistent_peers = "<sentry-id>@sentry-ip:26656"`
