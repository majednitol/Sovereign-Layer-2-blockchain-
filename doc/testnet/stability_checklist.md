# Testnet Stability Window & Freeze Checklist (Milestones 6.8 & 6.9)

> ❌ **CURRENT STATUS**: NOT YET STARTED — No public testnet stability run has occurred.

This document outlines the target consensus metrics, Prometheus query definitions, Grafana configuration details, freeze policies, and validation scenarios for verifying network stability before mainnet launch.

---

## 1. Consensus & Liveness Metrics

| Metric | Target | Prometheus Query | Alert Threshold |
|:---|:---|:---|:---|
| **Consensus Liveness** | 100% blocks finalized | `rate(tendermint_consensus_height[5m]) > 0` | Finality rate drops to 0 for > 15s |
| **Block Finality Latency** | Avg Block Time ≤ 5.0s | `histogram_quantile(0.99, sum(rate(tendermint_consensus_block_gossip_parts_latency_bucket[5m])) by (le))` | P99 latency > 6.0s |
| **Missed Block Rate** | 0 misses per validator | `sum(increase(tendermint_consensus_missing_validators[5m])) by (validator)` | > 10 missed blocks in 5 minutes |
| **Network Uptime** | 100% uptime (4 weeks) | `up{job="sovereign-node"}` | Any node goes down |

---

## 2. Grafana Dashboard Panels

To monitor the stability window, a dedicated Grafana dashboard (`Sovereign L1 Stability Dashboard`) must be configured with:
- **Consensus State Panel**: Panel visualizing active heights and active validator consensus voting power fractions.
- **Peer Connectivity Graph**: Visual mapping of persistent peer nodes and sentry latency.
- **ABCI++ Latency Histogram**: Quantile analysis of vote extensions process duration.

---

## 3. Code Freeze Policy

- **Freeze Deadline**: Post-Week 17.
- **Rule**: Zero protocol-breaking code modifications are allowed after this date.
- **Exceptions**: High or critical security patches (CVEs) or bugs resulting in consensus halts. All exceptions require a 5-of-7 cold multisig execution on the Constitution contract to approve code updates.

---

## 4. Phase G Exercise Scenarios

Before declaring the testnet run complete, the following scenarios must be executed and logged:

### 4.1 Core Operations
- [ ] **Bridge lock/unlock round-trips**: Transfer WSOV back and forth between BSC testnet and Sovereign L1. Verify the balance changes on both sides.
- [ ] **Oracle commit-reveal**: Confirm price reports for BTC_USD, ETH_USD, and BNB_USD progress without gap rounds.

### 4.2 Adversarial & Stress Testing
- [ ] **Dropped-reveal scenario**: Stop one oracle operator during the reveal window and assert that slashing/jailing behaves correctly without blocking price resolution.
- [ ] **Stale-feed scenario**: Simulate price feed source outage and verify the staleness state machine pauses the milestone clock.
- [ ] **Governance proposal lifecycle**: Submit, vote, and execute a proposal end-to-end to verify key authorization checks.
- [ ] **Treasury concurrent load**: Trigger multiple simultaneous withdrawals to check the reentrancy locks.

---

## 5. External Tester Invitation & Validation

External testers are encouraged to interact with the testnet to stress-test client-facing components.
- **Faucet Endpoint**: `/faucet` via Envoy Gateway (users request test CSOV).
- **Wallet Connection**: Test wallet onboarding with Keplr and MetaMask using testnet chain IDs.
- **Feedback Logs**: Issues reported by the community must be logged in `doc/testnet/testnet-bug-log.md`.

---

## 6. Daily Stability Report Template

Every 24 hours during the stability run, the operator must record:
```markdown
### Testnet Status Report - [Date]
- Active block height:
- Average block time (24h):
- Active validator count:
- Failed/Slashing events:
- Open bugs tracked:
- Bridge invariant verified (Yes/No):
```
