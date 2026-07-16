# Mainnet Cutover Launch Day Runbook (Phase H)

This document maps out the timeline, operations, and checklists for the Genesis Ceremony and L1 launch day.

---

## 1. Timeline and Checklist

```
  T-72h              T-24h                  T-0                    T+48h
  ┌──────────────────┼──────────────────────┼──────────────────────┐
  │ Pre-Ceremony     │ Genesis Ceremony     │ Block 1 Starts       │ Post-Launch Audit
  │ - Collect gentxs │ - Run ceremony.sh    │ - Monitor finality   │ - Deploy DEX pools
  │ - Replace addrs  │ - Distribute genesis │ - Check oracles      │ - Submit CMC/CG
  │ - Horcrux verify │ - Verify SHA-256     │ - Bridge init        │ - Run low-stakes gov
  └──────────────────┴──────────────────────┴──────────────────────┘
```

### T-72h: Pre-Ceremony Preparation

- [ ] **Replace OWNER_ACTION_REQUIRED markers** in `chain/genesis.prod.json`:
  - `circuit_breaker_address` → real Cosmos address of emergency pause key
  - `gnosis_safe_address` → real Cosmos address of Gnosis Safe operator
  - `lockbox_address` → real BSC address of deployed LockBox contract
- [ ] **Verify BSC-side infrastructure**: Confirm LockBox + Gnosis Safe are deployed on BSC mainnet per [bsc-bridge-checklist.md](./bsc-bridge-checklist.md).
- [ ] **Collect all validator gentxs**: Minimum 3 validators must submit `gentx-<moniker>.json` to `infra/mainnet/gentxs/`. See [validator-onboarding.md](./validator-onboarding.md).
- [ ] **Run Horcrux config check**: `./scripts/horcrux_ceremony_check.sh`
- [ ] **Cold Multisig custodians confirmed**: Ensure at least 5 custodians are active and reachable for the next 72 hours.

### T-24h: Genesis Ceremony & Verification

- [ ] **Run the ceremony script** (includes 7 automated safety checks):
  ```bash
  ./scripts/genesis-ceremony.sh
  ```
  The script will automatically verify:
  1. No OWNER_ACTION_REQUIRED placeholders remain
  2. No suspicious test/placeholder addresses present
  3. uwsov bridge supply starts at 0
  4. Minimum 3 validator gentxs collected
  5. Horcrux 2-of-3 threshold configs are valid
  6. chain_id is `sovereign-1`
  7. Compiled genesis passes `chaind validate-genesis`

- [ ] **Confirm SHA-256 checksum** matches across ALL validator operators.
- [ ] **Distribute the final genesis.json** securely to all validators.
- [ ] **Each validator independently verifies**:
  ```bash
  shasum -a 256 $HOME/.chain/config/genesis.json
  ```

### T-0: Chain Start & Genesis Block 1

- [ ] **Boot Sentry & Validator Nodes**: All operators launch the validator client with the verified genesis.
- [ ] **Liveness check**: Confirm the chain starts producing blocks and height increments.
- [ ] **Validator slot power check**: Verify consensus voting power distribution is as expected.
- [ ] **Oracle daemon startup**: Confirm oracle operators start publishing price commits (verify no `FATAL` log from HSM resolution — see `oracle/main.go`).
- [ ] **Bridge initialization**: Confirm LockBox on BSC is deployed and parameters (Safe address, relayer set) match genesis params exactly.

### T+48h: Post-Launch Stability

- [ ] **Verify no jailing events**: Verify validator set is stable and not jailing operators within the first 10,000-block window.
- [ ] **Bridge volume and invariant check**: Check that `uwsov` total supply remains 0 (since no BSC locks have occurred yet).
- [ ] **Grafana/Prometheus review**: Confirm consensus finality rate is 100% and block time is ≤ 5.0 seconds.
- [ ] **First governance proposal**: Execute a low-stakes governance proposal to prove the mechanism works publicly.

---

## 2. Emergency Pause & Rollback Operations

In the event of a critical contract bug or security breach during launch:

1. **Emergency Pause (CosmWasm)**: Trigger `emergency_pause` on Constitution contract to freeze WASM operations.
   - Requires cold multisig quorum (minimum 5-of-7 signers)
   - Constitution contract address is set at genesis

2. **Bridge Pause (BSC)**: Call `pause()` on BSC LockBox contract via Gnosis Safe multisig.
   - Requires `quorum_threshold: 3` signatures from registered relayers
   - This halts all BSC→Cosmos bridge minting

3. **Chain Halt**: If both pauses are insufficient:
   - Coordinate validator operators to stop node services simultaneously
   - Prepare upgrade plan with fix before restarting

4. **Communication**: Notify all channels (Discord, Telegram, Twitter) immediately upon any emergency action.

---

## 3. Key Files & Scripts Reference

| Purpose | File |
|---------|------|
| Genesis ceremony script | `scripts/genesis-ceremony.sh` |
| Horcrux config checker | `scripts/horcrux_ceremony_check.sh` |
| Production genesis template | `chain/genesis.prod.json` |
| Validator gentx submissions | `infra/mainnet/gentxs/` |
| Validator onboarding guide | `doc/mainnet/validator-onboarding.md` |
| BSC bridge checklist | `doc/mainnet/bsc-bridge-checklist.md` |
| ADR-007 operational security | `doc/adr/adr-007-operational-security.md` |
