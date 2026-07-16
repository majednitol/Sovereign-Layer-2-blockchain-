# Public Bug Bounty Policy — Sovereign L1

This document outlines the scope, rules, and reward structure for the Sovereign L1 Public Bug Bounty Program.

---

## 1. Scope

### In-Scope
The following components are subject to bounty rewards:
- **Smart Contracts (`/contracts/`)**: Rust/CosmWasm code for Constitution, Governance, Treasury, and Reserve Fund.
- **BSC Bridge Contracts (`/bridge/src/`)**: `LockBox.sol` Solidity contract.
- **Custom Go Modules (`/chain/x/`)**: `validator`, `certification`, `oracle`, `milestone`, `settlement`, `bridge`, `governance-ext`.
- **Relayer Engine (`/relayer/`)**: Relayer submission logic and signature aggregation.

### Out-of-Scope
- Frontend/dApp interface.
- CQRS API and indexing backend (except where SQL injection/DB exploitation can occur).
- Standard Cosmos SDK / Tendermint core libraries (unless a zero-day is found).
- Third-party dependencies.

---

## 2. Severity and Reward Structure

Rewards are determined based on the impact of the reported vulnerability, using the Immunefi Vulnerability Severity Classification System.

| Severity | Impact Scenario | Target Reward (USD) |
|:---|:---|:---|
| **Critical** | Direct theft of funds (Treasury, Reserve Fund, LockBox), permanent block halt, unauthorized rewrite of the Constitution. | Up to $50,000 (paid in CSOV) |
| **High** | Temporary freeze of funds, unauthorized contract migration, oracle price manipulation. | Up to $15,000 (paid in CSOV) |
| **Medium** | Jailing of validators without consensus violation, spamming state storage, temporary relayer halt. | Up to $5,000 (paid in CSOV) |
| **Low** | Contract queries failing, UI/UX mismatches, minor event-logging omissions. | Up to $1,000 (paid in CSOV) |

---

## 3. Submission Rules & Disclosures

- Submit findings to the designated secure communication channel: `security@sovereign.l1`.
- Do not disclose the vulnerability to the public until a fix has been deployed and verified.
- Do not exploit the vulnerability beyond what is required to demonstrate a Proof-of-Concept (PoC).
- Do not attempt physical attacks, social engineering, or DDoS.

---

## 4. Post-Launch Scope Expansion (Phase I)

Following the Mainnet launch, the bounty scope is expanded to cover live operational vulnerabilities on:
- PancakeSwap DEX WSOV/BNB pool manipulation and LP token locking/burning mechanisms.
- Production-grade bridge transactions against the live BSC Mainnet lockbox/Gnosis Safe.

---

## 5. Response SLA

We commit to the following response timeline for all valid submissions:
- **Acknowledgment:** Within 48 hours of submission.
- **Triage:** Within 7 business days.
- **Resolution & Reward:** Within 30 days of validation.

---

## 6. Continuous Bounty & Acknowledgments

The Sovereign L1 Public Bug Bounty Program operates continuously. Security researchers who successfully report verified vulnerabilities will be acknowledged in our Hall of Fame.
