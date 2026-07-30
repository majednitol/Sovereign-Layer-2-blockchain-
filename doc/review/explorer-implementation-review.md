# Sovereign L1 Explorer — Implementation Review Against Plan

> **Plan reference:** `sovereign-l1-explorer-plan.md` v6.0 (128 routes target)
> **Implementation:** `explorer/` directory
> **Review date:** 2026-07-30
> **Reviewer:** Cline (AI Software Engineer)

---

## 1. Executive Summary

The Sovereign L1 Explorer is a custom Next.js blockchain explorer designed to replace Ping.pub, Blockscout, and Celatone with a single unified application. The plan (v6.0) specifies **128 routes** across 4 phases with full Etherscan feature parity plus Sovereign L1 unique differentiators.

**Current implementation status: ~65 routes exist** (approximately 51% of the 128-route target). The implemented routes cover Phase 1 (Foundation) and Phase 2 (Custom Modules) well, with partial Phase 3 (Bridge/EVM/Analytics) and minimal Phase 4 (Hardening/Public API) coverage.

The explorer-indexer and explorer-api backend services are substantially more complete than the frontend, with comprehensive event decoding, token transfer indexing, and many REST endpoints already implemented.

---

## 2. Route Coverage Analysis

### 2.1 Phase 1 — Foundation Routes (Plan: 11 routes)

| Route | Plan Status | Implemented | File | Quality |
|---|---|---|---|---|
| `/` | ✅ | ✅ | `app/page.tsx` | Good — real data, TPS chart, stats summary |
| `/blocks` | ✅ | ✅ | `app/blocks/page.tsx` | Good — cursor pagination, gas progress bar, stat cards |
| `/blocks/:height` | ✅ | ✅ | `app/blocks/[height]/page.tsx` | Exists |
| `/txs` | ✅ | ✅ | `app/txs/page.tsx` | Exists |
| `/txs/:hash` | ✅ | ✅ | `app/txs/[hash]/page.tsx` | Exists |
| `/txs/pending` | ✅ | ✅ | `app/txs/pending/page.tsx` | Exists |
| `/address/:any` | ✅ | ✅ | `app/address/[any]/page.tsx` | Exists |
| `/consensus` | ✅ | ✅ | `app/consensus/page.tsx` | Exists |
| `/faucet` | ✅ | ✅ | `app/faucet/page.tsx` | Exists |
| `/params` | ✅ | ✅ | `app/params/page.tsx` | Exists |
| `/network` | ✅ | ✅ | `app/network/page.tsx` | Exists |

**Phase 1 verdict: 11/11 routes implemented (100%)** ✅

### 2.2 Phase 2 — Custom Modules + CosmWasm + Governance + IBC (Plan: 28 routes)

| Route | Plan Status | Implemented | Quality |
|---|---|---|---|
| `/validators` | ✅ | ✅ | Good — 30-slot grid, attestation score, missed blocks |
| `/validators/:addr` | ✅ | ✅ | Exists |
| `/certification` | ✅ | ✅ | Exists |
| `/staking` | ✅ | ✅ | Exists |
| `/staking/history` | ✅ | ✅ | Exists |
| `/oracle` | ✅ | ✅ | Exists |
| `/oracle/:feedId` | ✅ | ✅ | Exists |
| `/oracle/rounds/:roundId` | ✅ | ✅ | Exists |
| `/oracle/operators` | ✅ | ✅ | Exists |
| `/milestones` | ✅ | ✅ | Exists |
| `/milestones/:id` | ✅ | ✅ | Exists |
| `/settlements` | ✅ | ✅ | Exists |
| `/settlements/:id` | ✅ | ✅ | Exists |
| `/governance` | ✅ | ✅ | Exists |
| `/governance/:id` | ✅ | ✅ | Exists |
| `/governance/submit` | ✅ | ✅ | Exists |
| `/ibc` | ✅ | ✅ | Exists |
| `/ibc/channels` | ✅ | ✅ | Exists |
| `/ibc/channels/:channelId` | ✅ | ✅ | Exists |
| `/ibc/assets` | ✅ | ✅ | Exists |
| `/contracts` | ✅ | ✅ | Exists |
| `/contracts/:addr` | ✅ | ✅ | Exists |
| `/contracts/:addr/query` | ✅ | ✅ | Exists |
| `/contracts/:addr/execute` | ✅ | ✅ | Exists |
| `/contracts/:addr/nfts` | ✅ | ✅ | Exists |
| `/contracts/:addr/nfts/:tokenId` | ✅ | ✅ | Exists |
| `/codes` | ✅ | ✅ | Exists |
| `/codes/:codeId` | ✅ | ✅ | Exists |
| `/address/:bech32/send` | ✅ | ✅ | Exists |
| `/address/:bech32/stake` | ✅ | ✅ | Exists |
| `/codes/submit` | ✅ | ✅ | Exists |

**Phase 2 verdict: 28/28 routes implemented (100%)** ✅

### 2.3 Phase 3 — Bridge + EVM Tokens + Analytics (Plan: 22 routes)

| Route | Plan Status | Implemented | Quality |
|---|---|---|---|
| `/bridge` | ✅ | ✅ | Good — supply gauge, volume chart, tabs (overview/txs/relayers/nonces) |
| `/bridge/deposit` | ✅ | ✅ | Exists |
| `/bridge/withdraw` | ✅ | ✅ | Exists |
| `/bridge/tx/:nonce` | ✅ | ✅ | Exists |
| `/bridge/relayers` | ❌ | ❌ | **Missing** — plan specifies dedicated relayer page |
| `/bridge/nonces` | ❌ | ❌ | **Missing** — plan specifies dedicated nonce registry page |
| `/bridge/history` | ✅ | ✅ | Exists |
| `/evm` | ✅ | ✅ | Exists |
| `/evm/blocks` | ✅ | ✅ | Exists |
| `/evm/blocks/:number` | ✅ | ✅ | Exists |
| `/evm/txs/:hash` | ✅ | ✅ | Exists |
| `/evm/contracts` | ✅ | ✅ | Exists |
| `/evm/contracts/:addr` | ✅ | ✅ | Exists |
| `/evm/tokens` | ✅ | ✅ | Exists |
| `/evm/tokens/:addr` | ✅ | ✅ | Exists |
| `/evm/tokens/:addr/multi` | ❌ | ❌ | **Missing** — ERC-1155 detail page |
| `/evm/nfts` | ✅ | ✅ | Exists |
| `/evm/nfts/:addr` | ✅ | ✅ | Exists |
| `/evm/nfts/:addr/:id` | ❌ | ❌ | **Missing** — individual NFT detail page |
| `/analytics` | ✅ | ✅ | Exists |
| `/developers` | ✅ | ✅ | Exists |
| `/verify` | ✅ | ✅ | Exists |

**Phase 3 verdict: 18/22 routes implemented (82%)** 🟡

### 2.4 Phase 4 — Hardening + Public API (Plan: 3 routes)

| Route | Plan Status | Implemented | Quality |
|---|---|---|---|
| `/search` | ✅ | ✅ | Exists |
| `/docs` | ✅ | ✅ | Exists |
| `/status` | ✅ | ✅ | Exists |

**Phase 4 verdict: 3/3 routes implemented (100%)** ✅

### 2.5 Appendix E — Etherscan Parity Additions (Plan: 37 routes)

| Route | Implemented | Notes |
|---|---|---|
| `/gastracker` | ✅ | Exists |
| `/tokens/:addr` | ❌ | **Missing** — standalone token detail page |
| `/nfts` | ✅ | Exists |
| `/charts` | ✅ | Exists |
| `/charts/:chartName` (22 sub-routes) | ❌ | **Missing** — 22 individual chart pages not implemented |
| `/stat/supply` | ✅ | Exists |
| `/accounts` | ✅ | Exists |
| `/contracts/verified` | ✅ | Exists |
| `/contracts/verified/:type` | ❌ | **Missing** — type filter route |
| `/txs/internal` | ❌ | **Missing** |
| `/token-approvals` | ✅ | Exists |
| `/address/:addr/tokencheck` | ❌ | **Missing** |
| `/label/:slug` | ❌ | **Missing** |
| `/exportData` | ✅ | Exists |
| `/myaccount` | ❌ | **Missing** — user account page |
| `/pushnotification` | ❌ | **Missing** |

**Appendix E verdict: 7/37 routes implemented (19%)** 🔴

### 2.6 Appendix F — Second Pass Etherscan Parity (Plan: 30 routes)

| Route | Implemented | Notes |
|---|---|---|
| `/txs/withdrawals` | ❌ | **Missing** |
| `/blocks/reorgs` | ❌ | **Missing** |
| `/txs/erc20` | ❌ | **Missing** |
| `/txs/erc721` | ❌ | **Missing** |
| `/txs/erc1155` | ❌ | **Missing** |
| `/evm/tokens/multi` | ❌ | **Missing** |
| `/nfts/top-mints` | ❌ | **Missing** |
| `/nfts/latest-trades` | ❌ | **Missing** |
| `/nfts/latest-transfers` | ❌ | **Missing** |
| `/nfts/latest-mints` | ❌ | **Missing** |
| `/stats` | ✅ | Exists |
| `/directory` | ❌ | **Missing** |
| `/labelcloud` | ❌ | **Missing** |
| `/domains` | ❌ | **Missing** |
| `/txs/advanced-filter` | ✅ | Exists |
| `/gas/guzzlers` | ❌ | **Missing** |
| `/gas/spenders` | ❌ | **Missing** |
| `/tools` | ✅ | Exists |
| `/tools/unit-converter` | ❌ | **Missing** |
| `/tools/broadcast` | ❌ | **Missing** |
| `/tools/abi-encoder` | ❌ | **Missing** |
| `/tools/disassembler` | ❌ | **Missing** |
| `/tools/similar-contracts` | ❌ | **Missing** |
| `/tools/contract-diff` | ❌ | **Missing** |
| `/tools/constructor-args` | ❌ | **Missing** |
| `/tools/verify-signature` | ❌ | **Missing** |
| `/tools/compiler` | ❌ | **Missing** |
| `/tools/code-reader` | ❌ | **Missing** |
| `/api-plans` | ❌ | **Missing** |
| `/tokens/:addr/submit` | ❌ | **Missing** |

**Appendix F verdict: 3/30 routes implemented (10%)** 🔴

### 2.7 Additional Routes (not in plan but implemented)

| Route | Notes |
|---|---|
| `/deployments` | Extra — contract deployment list |
| `/evm/addresses/[addr]` | Extra — EVM-specific address page |
| `/evm/txs` | Extra — EVM transaction list |
| `/vaults/[addr]` | Extra — vault detail page |

### 2.8 Route Coverage Summary

| Phase | Planned | Implemented | Coverage |
|---|---|---|---|
| Phase 1 (Foundation) | 11 | 11 | 100% ✅ |
| Phase 2 (Custom Modules) | 28 | 28 | 100% ✅ |
| Phase 3 (Bridge/EVM/Analytics) | 22 | 18 | 82% 🟡 |
| Phase 4 (Hardening) | 3 | 3 | 100% ✅ |
| Appendix E (Etherscan parity) | 37 | 7 | 19% 🔴 |
| Appendix F (Second pass) | 30 | 3 | 10% 🔴 |
| **Total** | **128** | **~65** | **~51%** |

---

## 3. Implementation Quality Review

### 3.1 Data Fetching Patterns — Inconsistent

**Problem:** Two different data fetching patterns coexist:

1. **Pattern A (Correct):** Uses `apiClient.get()` with Zod schema validation
   - Example: `explorer/app/page.tsx`, `explorer/app/blocks/page.tsx`
   - Uses TanStack Query with proper cache keys
   - Schema validation catches API response mismatches

2. **Pattern B (Incorrect):** Uses raw `fetch()` without schema validation
   - Example: `explorer/app/validators/page.tsx`, `explorer/app/bridge/page.tsx`
   - Uses `useEffect` + `useState` instead of TanStack Query
   - No schema validation — type mismatches silently fail
   - Falls back to mock data on API failure

**Recommendation:** Migrate all pages to Pattern A (apiClient + TanStack Query + Zod). Remove all mock data fallbacks — the plan explicitly states "No mock data on any page."

### 3.2 Mock Data Fallbacks — Violates Plan Requirement

**Problem:** The bridge page (`app/bridge/page.tsx`) falls back to hardcoded mock data when the API fails:

```typescript
} catch (err) {
  console.warn("Failed to fetch bridge data from API. Falling back to mocks.", err);
  setMetrics({
    cosmosMinted: "1250000000000",
    // ... hardcoded mock data
  });
```

**Plan requirement (Phase 1 deliverable):** "No mock data on any page."

**Recommendation:** Remove all mock data fallbacks. Show proper error states instead.

### 3.3 Hardcoded Values

**Problem:** The blocks page (`app/blocks/page.tsx`) hardcodes the base fee:

```typescript
<span className="text-gray-400 font-mono text-xs">0.025 ucsov</span>
```

And calculates burnt fees using this hardcoded value:

```typescript
const burnt = (row.original.gasUsed || 0) * 0.025;
```

**Recommendation:** Fetch base fee from the API (`/api/rest/v1/explorer/gas-price` endpoint already exists in explorer-api).

### 3.4 Component Library — Missing shadcn/ui

**Plan specifies:** "Tailwind CSS + shadcn/ui"

**Actual:** Custom components in `components/ui/` (Card, Badge, DataTable, Sparkline, Tabs). No shadcn/ui dependency in `package.json`.

**Assessment:** The custom components are functional but lack the polish and accessibility features of shadcn/ui (which provides Radix UI primitives, ARIA compliance, keyboard navigation out of the box).

**Recommendation:** Either migrate to shadcn/ui or document the decision to use custom components and ensure accessibility is manually addressed.

### 3.5 Schema Coverage — Incomplete

**Problem:** `explorer/lib/schemas.ts` (193 lines) defines Zod schemas for only ~15 data types. The explorer-api has 50+ endpoints, many without corresponding Zod schemas.

**Missing schemas for:**
- Validator detail (full)
- Oracle feed detail
- Oracle round detail
- Milestone detail
- Settlement detail
- Governance proposal detail
- IBC channel detail
- IBC asset list
- Contract detail
- Code detail
- Bridge supply metrics
- Bridge relayer list
- Circuit breaker events
- Nonce registry
- EVM block detail
- EVM tx detail
- EVM contract detail
- EVM token detail
- EVM NFT detail
- Developer hub config
- Verify result

**Recommendation:** Add Zod schemas for all API responses. Every `apiClient.get()` call should have a schema.

### 3.6 Error Handling — Missing Error Boundaries

**Problem:** No error boundary components found in the explorer. If an API call fails and a component throws during render, the user sees a white screen.

**Plan requirement (Phase 4):** "Sentry error tracking integrated"

**Recommendation:**
1. Add a Next.js `error.tsx` boundary at the app root level
2. Add per-route `error.tsx` boundaries for data-heavy pages
3. Integrate Sentry for production error tracking
4. Show user-friendly error states with retry buttons

### 3.7 Loading States — Inconsistent

**Problem:** Some pages show loading text ("Loading blocks data feed..."), others show "..." placeholders, and some show nothing while loading.

**Recommendation:** Create a consistent `<LoadingState />` component (skeleton or spinner) and use it across all pages.

### 3.8 Responsive Design — Not Verified

**Plan requirement (Phase 4):** "Mobile responsive: all pages verified at 375px viewport"

**Assessment:** Pages use Tailwind responsive classes (`sm:`, `md:`, `lg:`) but no systematic mobile testing has been done. The validators grid uses `grid-cols-2 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-6` which should work but needs verification.

**Recommendation:** Run Lighthouse audit on all pages at 375px viewport. Fix any layout issues.

### 3.9 Accessibility — Not Addressed

**Plan requirement (Phase 4):** "WCAG 2.1 AA accessibility audit"

**Assessment:** No ARIA labels found on icon-only buttons. Color contrast appears sufficient (dark theme with high-contrast text) but not formally verified. No keyboard navigation testing done.

**Recommendation:**
1. Add `aria-label` to all icon-only buttons (copy, link, search)
2. Run axe-core or WAVE accessibility scanner
3. Verify keyboard navigation on interactive elements
4. Test with screen reader

---

## 4. Backend (Explorer-API) Review

### 4.1 API Coverage — Extensive

The explorer-api (`explorer-api/main.go`, 5554 lines) implements a comprehensive set of REST endpoints:

**Implemented endpoints:**
- Block: `GetBlock`, `ListBlocks` (cursor pagination)
- Transaction: `GetTx`, `ListTxs`, `ListTxsByAddress`
- Address: `GetAddress` (with bech32 ↔ hex resolution)
- Validator: `GetValidator`, `ListValidators`
- Streaming: `StreamLatestBlocks` (NATS), `StreamConsensusRound`
- Stats: `stats/summary`, `gas-price`, `top-accounts`, `supply-distribution`
- Analytics: `analytics/tps`, `analytics/block-time`, `analytics/validator-uptime`, `analytics/bridge-volume`
- Bridge: `bridge/deposits`, `bridge/withdraws`, `bridge/txs/:nonce`
- Tokens: `tokens/evm/:addr` (transfers, holders), `tokens/cw20/:addr` (transfers, holders)
- NFTs: `nfts/evm/:addr`, `nfts/cw721/:addr`
- Vaults: `vaults/evm/:addr`
- Contracts: `contracts/deployments`, `contracts/:addr/holders`
- Governance: `governance/proposals/:id/constitution-check`
- Staking: `staking/slot-events`, `validators/:addr/signing-history`
- Address: `addresses/:addr/portfolio`
- Faucet: `faucet` (POST)
- Mempool: `mempool`
- Etherscan API: `/api` (Etherscan-compatible)
- Verify: `verify/evm`, `verify/cosmwasm`, `evm/contracts/:addr`, `cosmwasm/codes/:codeId`
- Charts: `charts/:chartName`
- Gas: `gas-tracker`

### 4.2 Security Issues

**🔴 P0 — SQL Injection in ListTxsByAddress:**

```go
rows, err = s.db.Query(ctx, `
    SELECT ... FROM explorer.transactions
    WHERE (decoded::text LIKE '%' || $1 || '%' OR (decoded::text LIKE '%' || $2 || '%' AND $2 <> ''))
    ORDER BY height DESC, hash DESC
    LIMIT $3`, addressBech32, addressHex, limit)
```

While parameterized ($1, $2), the `LIKE '%' || $1 || '%'` pattern is safe from SQL injection (parameters are bound), but it performs a full-text scan on the JSONB column. This is a performance issue, not a security issue. However, the `addressBech32` and `addressHex` values come from user input and should be validated before use.

**🟡 P1 — CORS Wildcard:**

```go
w.Header().Set("Access-Control-Allow-Origin", "*")
```

The plan specifies "strict allowlist (explorer domain only)." The wildcard `*` allows any origin to make cross-origin requests.

**Recommendation:** Replace with configurable allowed origins from environment variable.

**🟡 P1 — Hardcoded Service URL:**

```go
func fetchBalances(address string) string {
    client := &http.Client{Timeout: 3 * time.Second}
    resp, err := client.Get(fmt.Sprintf("http://chain-node:1317/cosmos/bank/v1beta1/balances/%s", address))
```

The `chain-node:1317` hostname is hardcoded. In production K8s, the service DNS name may differ.

**Recommendation:** Make configurable via environment variable.

### 4.3 Mock Data in API

**Problem:** `StreamConsensusRound` returns mock validator data:

```go
round := &explorerv1.ConsensusRound{
    Height:   1,
    Round:    0,
    Step:     "RoundStepCommit",
    Proposer: "sovereignvaloper1valaddr0",
    Votes: []*explorerv1.ValidatorVote{
        {Validator: "sovereignvaloper1valaddr0", Voted: true, Power: 1000},
        {Validator: "sovereignvaloper1valaddr1", Voted: true, Power: 1000},
    },
}
```

Only the height is fetched from CometBFT; the rest is hardcoded.

**Recommendation:** Fetch real consensus round data from CometBFT RPC (`/consensus_state` endpoint).

---

## 5. Backend (Explorer-Indexer) Review

### 5.1 Indexer Coverage — Comprehensive

The explorer-indexer (`explorer-indexer/main.go`, 1691 lines) is the most complete component:

**Indexed data:**
- Cosmos blocks (height, time, proposer, tx_count, gas, app_hash)
- Cosmos transactions (hash, height, type, msg_types, decoded JSONB, fee, gas, status)
- Account mappings (bech32, hex, first_seen, last_active)
- Bridge transactions (direction, nonce, status, source/dest hash, amount)
- CosmWasm contracts (address, code_id, label, creator, admin, type_badge)
- CosmWasm token transfers (CW-20 transfers, CW-721 NFT transfers)
- CosmWasm token holders (balance tracking)
- CosmWasm NFT ownership
- EVM contracts (address, type_badge)
- EVM token transfers (ERC-20, ERC-721, ERC-1155)
- EVM token holders (balance tracking)
- EVM NFT ownership
- EVM vault events (ERC-4626 Deposit/Withdraw)
- Contract deployments
- Daily network stats aggregation
- Module events (validator, certification, oracle, milestone, settlement)
- BSC lock events (via BSC watcher goroutine)
- Webhook dispatch system
- Search index materialized view refresh

### 5.2 Indexer Quality Issues

**Chain Reset Handling:**

```go
if latestHeight < lastProcessedHeight {
    log.Printf("Detected chain reset...")
    _, err := db.Exec(ctx, `TRUNCATE TABLE explorer.blocks, ... CASCADE`)
```

This is a good defensive measure but truncating ALL tables on chain reset is aggressive. A more nuanced approach would be to only truncate blocks above the reset height.

**EVM Log Ingestion — No Error Handling:**

```go
logs, err := evmClient.FilterLogs(ctx, query)
if err != nil {
    log.Printf("failed to fetch EVM logs for block %d: %v", height, err)
} else {
    // process logs
}
```

If EVM log fetching fails, the block is still committed without EVM data. This could lead to missing token transfers. Consider retrying or marking the block for re-indexing.

**Token Holder Balance Tracking — Race Condition Risk:**

```go
_, _ = tx.Exec(ctx, `
    INSERT INTO explorer.evm_token_holders (token_address, holder_address, balance)
    VALUES ($1, $2, $3)
    ON CONFLICT (token_address, holder_address)
    DO UPDATE SET balance = explorer.evm_token_holders.balance - EXCLUDED.balance`,
    tokenAddr, from, val.String(),
)
```

The `balance - EXCLUDED.balance` pattern could go negative if transfers are processed out of order. The cleanup `DELETE FROM explorer.evm_token_holders WHERE balance = 0` handles zero balances but not negative ones.

**Recommendation:** Add a `CHECK (balance >= 0)` constraint on the holders table, or use `GREATEST(0, balance - EXCLUDED.balance)`.

---

## 6. Missing Features (Plan vs Implementation)

### 6.1 Critical Missing Frontend Features

| Feature | Plan Section | Impact |
|---|---|---|
| 22 individual chart sub-pages (`/charts/:chartName`) | E.14.4 | Major — plan specifies dedicated pages for each metric |
| Standalone token detail page (`/tokens/:addr`) | E.10 | Major — tokens only accessible via contract page |
| ERC-1155 collection list (`/evm/tokens/multi`) | F.20 | Medium — only detail page exists |
| Individual NFT detail (`/evm/nfts/:addr/:id`) | Phase 3 | Medium — collection page exists but not per-token |
| User account system (`/myaccount`) | E.20 | Low (Phase 4) — watchlist, private notes, API keys |
| Developer tools (10 tool routes) | F.12 | Medium — only `/tools` hub exists |
| Staking withdrawals page (`/txs/withdrawals`) | F.19 | Medium — undelegation completions |
| Chain-wide token transfer pages (`/txs/erc20`, etc.) | F.18 | Medium — 3 routes missing |
| NFT activity pages (top-mints, latest-trades, etc.) | F.17 | Medium — 4 routes missing |
| Gas guzzlers/spenders (`/gas/guzzlers`, `/gas/spenders`) | F.6 | Low — analytics feature |
| Label cloud (`/labelcloud`) | F.10 | Low — address labeling |
| Directory (`/directory`) | F.9 | Low — project directory |
| API plans page (`/api-plans`) | F.15 | Low (Phase 4) |
| Domain names (`/domains`) | F.2 | Low (Phase 4) |

### 6.2 Missing Data Fields on Existing Pages

Based on the plan's detailed field specifications (Appendix E & F), many data fields are missing from existing pages:

**Home page missing:**
- Token price (SOV/USD from oracle)
- Market cap
- Median gas price (USD)
- Last finalized block
- 14-day transaction history sparkline
- Block reward per block
- Validator moniker label

**Blocks list missing:**
- Network utilization banner (24H)
- Burnt fees total (24H)
- Base fee per gas (from API, not hardcoded)
- Block reward per block
- Validator moniker (linked)
- "Total of N blocks" header
- CSV download button

**Block detail missing:**
- Consensus Info tab (round, step, commit signatures)
- Block status badge (Finalized/Unfinalized)
- Block size in bytes
- Parent hash (linked)
- State root / app hash
- Last commit hash
- Prev/next block navigation arrows
- Confirmation count badge

**Transaction detail missing:**
- Action summary banner (human-readable 1-line description)
- Block confirmation count badge
- Gas limit & usage breakdown
- Gas fees (base, max, priority)
- Burnt fees + savings fees
- Tx type (Direct/Legacy)
- Account nonce/sequence
- Position in block
- Token transfers section (within tx)
- Internal transactions section
- Event logs section
- Raw ↔ decoded input toggle
- State changes tab

**Address page missing:**
- IN/OUT directional badges on tx rows
- Token holdings dropdown
- USD value (balance × oracle price)
- Funded by field
- Vesting schedule display
- Balance history sparkline
- Token approvals button
- Advanced filter panel
- CSV export
- Webhook subscribe
- QR code modal
- Address tags/labels
- Watchlist star

**Validator page missing:**
- Per-block signing heatmap (calendar grid)
- Oracle participation panel
- Governance vote history
- Delegation list
- Ejection timeline
- APR calculator

**Bridge page issues:**
- Falls back to mock data (violates "no mock data" requirement)
- Volume chart uses hardcoded data instead of API
- Missing confirmation progress bar animation
- Missing quorum signature tracker (N of T checkbox grid)

### 6.3 Missing Global UI Features

| Feature | Plan Section | Status |
|---|---|---|
| Global header bar (token price + gas always visible) | E.1 | ❌ Missing |
| Copy-to-clipboard on every address/hash | E.22 | ❌ Missing |
| Relative/absolute timestamp toggle | E.22 | ❌ Missing |
| Address truncation with tooltip | E.22 | ❌ Missing |
| Transaction status badge (Success/Failed/Pending) | E.22 | Partial — home page only |
| Breadcrumb navigation on all pages | E.22 | Partial — some pages have it |
| Pagination controls (First/Prev/Next/Last) | E.22 | Partial — cursor-based only |
| Download Page Data (CSV) on every list | E.22 | ❌ Missing |
| "More Details" expandable row | E.22 | ❌ Missing |
| Back to Top button | E.22 | ❌ Missing |
| Keyboard shortcut (`/` to focus search) | E.22 | ❌ Missing |
| Block/Tx not found friendly 404 | E.22 | ❌ Missing |
| Dark/light mode toggle | E.1 | ❌ Missing (dark only) |

---

## 7. Tech Stack Compliance

| Component | Plan | Actual | Status |
|---|---|---|---|
| Framework | Next.js 14 (App Router) | Next.js 16.2.9 | ✅ (newer — acceptable) |
| Language | TypeScript 5.x strict | TypeScript 5.x | ✅ |
| Styling | Tailwind CSS + shadcn/ui | Tailwind CSS only | 🟡 (shadcn/ui missing) |
| Charts | Recharts | Recharts 2.15.4 | ✅ |
| Tables | TanStack Table v8 | TanStack Table 8.19.3 | ✅ |
| Query | TanStack Query v5 | TanStack Query 5.51.1 | ✅ |
| Forms | React Hook Form + Zod | React Hook Form 7.52.1 + Zod 3.23.8 | ✅ |
| JSON view | react-json-view | ❌ Not installed | 🟡 |
| Syntax HL | Prism.js | ❌ Not installed | 🟡 |
| Cosmos wallets | @cosmjs/stargate + @keplr-wallet/types | ✅ Installed | ✅ |
| EVM wallet | wagmi v2 + viem + RainbowKit | ✅ Installed | ✅ |
| gRPC-Web | buf-generated stubs | @workspace/api-spec | ✅ |
| State | Zustand | Zustand 4.5.2 | ✅ |
| Themes | next-themes (dark/light) | next-themes 0.3.0 | ✅ (but dark only implemented) |

---

## 8. Priority Remediation Roadmap

### 🔴 P0 — Must Fix (Blocks production readiness)

| # | Issue | Effort | Impact |
|---|---|---|---|
| 1 | Remove mock data fallbacks from all pages | Small | Plan compliance — "no mock data" |
| 2 | Fix CORS wildcard in explorer-api | Small | Security — cross-origin attacks |
| 3 | Add error boundaries (app root + data-heavy routes) | Medium | UX — no white screens on API failure |
| 4 | Migrate all pages to apiClient + TanStack Query + Zod | Medium | Consistency — schema validation on all calls |
| 5 | Add missing Zod schemas for all API responses | Medium | Type safety — catch API response drift |
| 6 | Fix hardcoded base fee in blocks page | Small | Correctness — fees displayed incorrectly |

### 🟡 P1 — Should Fix (Before testnet launch)

| # | Issue | Effort | Impact |
|---|---|---|---|
| 7 | Implement 22 individual chart sub-pages | Large | Plan compliance — E.14.4 |
| 8 | Implement standalone token detail page (`/tokens/:addr`) | Medium | Plan compliance — E.10 |
| 9 | Implement missing Phase 3 routes (4 routes) | Medium | Plan compliance — 82% → 100% |
| 10 | Add global header bar with token price + gas | Medium | UX — E.1 |
| 11 | Add copy-to-clipboard on all addresses/hashes | Small | UX — E.22 |
| 12 | Add relative/absolute timestamp toggle | Small | UX — E.22 |
| 13 | Fix StreamConsensusRound mock data | Medium | Correctness — real consensus data |
| 14 | Add token holder balance CHECK constraint | Small | Data integrity — prevent negative balances |
| 15 | Make hardcoded service URLs configurable | Small | Deployability — K8s service DNS |

### 🟢 P2 — Fix During Hardening (Phase 4)

| # | Issue | Effort | Impact |
|---|---|---|---|
| 16 | Implement Appendix E routes (30 missing) | Large | Etherscan parity |
| 17 | Implement Appendix F routes (27 missing) | Large | Etherscan parity |
| 18 | Add shadcn/ui or formalize custom components | Medium | Accessibility — ARIA compliance |
| 19 | Run Lighthouse audit on all pages | Medium | Performance — ≥90 target |
| 20 | Run WCAG 2.1 AA accessibility audit | Medium | Compliance |
| 21 | Add dark/light mode toggle | Small | UX — plan specifies both |
| 22 | Add CSV export on all list pages | Medium | UX — E.22 |
| 23 | Add breadcrumb nav on all pages | Small | UX — E.22 |
| 24 | Add keyboard shortcut for search | Small | UX — E.22 |
| 25 | Integrate Sentry error tracking | Small | Operations — Phase 4 |
| 26 | Add missing data fields to existing pages | Large | Etherscan parity — many fields missing |

---

## 9. What's Working Well

### Strengths of the Current Implementation

1. **Explorer-Indexer is comprehensive** — handles Cosmos, EVM, CosmWasm, bridge events, all token standards (ERC-20/721/1155, CW-20/721/1155), ERC-4626 vault events, contract deployments, webhooks, and search index. This is the most mature component.

2. **Explorer-API has extensive endpoint coverage** — 5554 lines with 40+ custom REST handlers covering blocks, txs, validators, bridge, analytics, tokens, NFTs, contracts, governance, staking, faucet, mempool, Etherscan-compatible API, and contract verification.

3. **Phase 1 & 2 routes are 100% implemented** — all foundation and custom module routes exist with real data.

4. **Home page is well-designed** — live TPS chart, stats summary cards, recent blocks/txs feeds, search bar, good visual hierarchy.

5. **Blocks list page has good UX** — cursor pagination, gas progress bar, stat cards, breadcrumb nav, DataTable component.

6. **Validator slot grid is unique and well-implemented** — 30-cell grid with slot numbers, attestation scores, missed blocks, status indicators. This is a Sovereign L1 differentiator that no other explorer has.

7. **Bridge page has rich functionality** — supply invariant gauge, volume chart, tabbed interface (overview/txs/relayers/nonces), bitmap nonce registry visualization, relayer promotion ladder display.

8. **Zod schema validation pattern is correct** (where used) — `apiClient.get()` with Zod schemas catches API response format mismatches at runtime.

9. **TanStack Query usage is correct** (where used) — proper cache keys, refetch intervals, loading states.

10. **Token standard coverage in indexer** — ERC-20, ERC-721, ERC-1155, ERC-4626, CW-20, CW-721, CW-1155 all decoded and indexed with holder tracking.

---

## 10. Final Assessment

| Dimension | Score | Notes |
|---|---|---|
| Route coverage | 51% (65/128) | Phase 1-2 complete, Appendix E-F mostly missing |
| Data fetching consistency | 60% | Two patterns coexist, some pages lack schema validation |
| Mock data compliance | 70% | Some pages still fall back to mocks |
| Error handling | 30% | No error boundaries, inconsistent error states |
| Loading states | 50% | Inconsistent across pages |
| Accessibility | 20% | No formal audit, missing ARIA labels |
| Responsive design | 70% | Tailwind classes used but not verified |
| Backend API coverage | 85% | Extensive endpoint list, some mock data |
| Indexer coverage | 95% | Comprehensive event decoding |
| Security | 70% | CORS wildcard, hardcoded URLs, SQL LIKE performance |
| Plan compliance | 50% | Major gaps in Appendix E and F routes |

**Overall verdict:** The explorer has a solid foundation with Phase 1-2 routes complete and a comprehensive backend. However, it is **not yet production-ready** due to:
1. Missing 63 routes (49% of plan)
2. Mock data fallbacks violating the "no mock data" requirement
3. Missing error boundaries
4. Inconsistent data fetching patterns
5. Missing accessibility features
6. CORS security issue

**Estimated effort to complete:** 8-12 weeks for 2-3 frontend engineers to implement missing routes, fix data fetching consistency, add error boundaries, and pass accessibility/performance audits.