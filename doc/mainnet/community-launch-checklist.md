# Community & Marketing Launch Readiness Checklist

This document acts as the master checklist to align technical milestones with community/marketing activities prior to and during the Sovereign L1 mainnet launch.

---

## 1. Pre-Launch Warmup Phase

- [ ] **Release Official Documentation:**
  - Publicly publish the [Wallet Setup Guide](file:///Users/majedurrahman/Sovereign/doc/mainnet/wallet-setup-guide.md).
  - Distribute validator onboarding materials.
- [ ] **Bug Bounty Launch:**
  - Deploy the updated [Bug Bounty Policy](file:///Users/majedurrahman/Sovereign/doc/ops/bug-bounty-policy.md) on the website.
- [ ] **Community Announcement:**
  - Draft announcement post for Twitter/X and Discord detailing the migration path.
- [ ] **Etherscan/BscScan Verification:**
  - Verify `LockBox.sol` source code on BscScan.
  - Upload token logo and description to BSC Scan.

---

## 2. Launch Day Operations

- [ ] **Genesis Ceremony Verification:**
  - Run `verify-genesis` on validator gentxs and announce the official SHA-256 genesis hash.
- [ ] **Launch Chain & Relayer:**
  - Boot initial primary validators and ensure the bridge relayer begins monitoring blocks.
- [ ] **Add Liquidity:**
  - Follow the [Liquidity Provision Guide](file:///Users/majedurrahman/Sovereign/doc/mainnet/liquidity-provision-guide.md) to add WSOV/BNB pool on PancakeSwap.
  - Execute LP Token lock.
- [ ] **Submit Listings:**
  - Submit applications to CoinMarketCap and CoinGecko using the supply API routes:
    - `/api/total-supply`
    - `/api/circulating-supply`
- [ ] **Verify End-to-End Bridge Flow:**
  - Perform one standard test transaction to bridge WSOV from BSC to Sovereign L1.

---

## 3. Post-Launch & Governance

- [ ] **First Governance Proposal:**
  - Draft and submit the proposal to adjust EVM parameters using [First Governance Proposal](file:///Users/majedurrahman/Sovereign/doc/governance/first-governance-proposal.md).
  - Coordinate active validator voting to reach consensus.
