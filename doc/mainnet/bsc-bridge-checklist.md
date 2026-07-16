# BSC Bridge Infrastructure Launch Checklist

This checklist tracks the deployment and verification of the BSC-side bridge smart contracts and coordination infrastructure before mainnet launch.

## 1. Solidity Contracts Deployment (BSC Mainnet)

- [ ] **Deploy Gnosis Safe Multi-Sig**
  - [ ] Set threshold to 3-of-5.
  - [ ] Roster of signers:
    - [ ] Signer 1: `[PENDING]`
    - [ ] Signer 2: `[PENDING]`
    - [ ] Signer 3: `[PENDING]`
    - [ ] Signer 4: `[PENDING]`
    - [ ] Signer 5: `[PENDING]`
  - [ ] Record deployed Gnosis Safe address: `[PENDING]`
- [ ] **Deploy LockBox.sol contract**
  - [ ] Owner: Gnosis Safe address above.
  - [ ] Rate limits configured: `maxUnlockPerBlock` set to `100,000 WSOV` equivalent.
  - [ ] Rate limit duration and recovery tested on testnet.
  - [ ] Record deployed LockBox address: `[PENDING]`
  - [ ] Verify contract source code on BscScan.

## 2. Invariant & Parameter Verification

- [ ] **Verify Genesis Bridge Parameters**
  - [ ] `gnosis_safe_address` in `genesis.prod.json` matches deployed safe.
  - [ ] `lockbox_address` in `genesis.prod.json` matches deployed LockBox.
  - [ ] `circuit_breaker_address` matches operational multi-sig/EOA.
- [ ] **Supply Cap Alignment**
  - [ ] Cosmos-side supply cap for `uwsov` is set to `1,000,000,000 WSOV`.
  - [ ] LockBox contract locked balance matches bridge minted supply (initially 0).

## 3. Relayer Infrastructure Setup

- [ ] **Relayer Node Deployment**
  - [ ] Deploy relayer instance 1 (Region A).
  - [ ] Deploy relayer instance 2 (Region B).
  - [ ] Configure NATS authentication and account isolation.
- [ ] **Keys & DB Security**
  - [ ] Verify relayer keys are stored in encrypted vaults.
  - [ ] Verify Relayer DB PostgreSQL database users use minimal permissions.

## 4. Test Lock/Unlock Execution Drill

- [ ] **Test Transaction (BSC Testnet to Sovereign Devnet)**
  - [ ] Perform a deposit of `10 WSOV` equivalent on BSC.
  - [ ] Verify relayer detects deposit and submits `MsgBridgeIn` to Sovereign.
  - [ ] Verify receiver address balance matches on Sovereign chain.
  - [ ] Perform withdrawal (bridge out) on Sovereign.
  - [ ] Verify relayer processes withdrawal and unlocks tokens on BSC.
  - [ ] Verify bridge supply invariants hold.
