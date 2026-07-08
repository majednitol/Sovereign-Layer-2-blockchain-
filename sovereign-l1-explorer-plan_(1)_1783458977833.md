# Sovereign L1 — Custom Enterprise Blockchain Explorer
## Complete Implementation Plan — A to Z Master Reference

**Version:** 5.0 (A-to-Z rewrite; implementation status added to every route and feature)
**Date:** 2026-07-07
**Scope:** Single unified explorer replacing Ping.pub + Blockscout + Celatone + custom modules
**Timeline:** 20 weeks across 4 phases (original timeline preserved)
**Total Routes:** 128 (61 core + 37 Appendix E + 30 Appendix F)

---

## Status Legend

| Symbol | Meaning |
|---|---|
| ✅ **IMPLEMENTED** | Route/feature exists in code, real data flows from backend API |
| 🔧 **NEEDS WIRING** | Route/page exists in code but uses placeholder/mock data — backend connection incomplete |
| ❌ **MISSING** | Route/page not yet created in the codebase |
| 🔵 **REPLACES** | Notes which off-the-shelf tool (Ping.pub / Blockscout / Celatone) this replaces |

---

## What This Explorer Replaces (Tool Coverage Map)

This explorer is designed to fully supersede all three off-the-shelf tools. Once complete, Ping.pub, Blockscout, and Celatone can be decommissioned.

| Tool | What It Covers | Replaced By (Explorer Routes) |
|---|---|---|
| **Ping.pub** | Cosmos blocks, txs, validators (stake-weighted), staking, governance, IBC, consensus, faucet, wallet send | `/blocks`, `/txs`, `/validators`, `/staking`, `/governance`, `/ibc`, `/consensus`, `/faucet`, `/address/:any/send` |
| **Blockscout** | EVM blocks, txs, tokens (ERC-20/721/1155), verified contracts, read/write UI, gas tracker, charts, REST API | `/evm/*`, `/gastracker`, `/charts/*`, `/tools/*`, `/verify`, REST API at `/api` |
| **Celatone** | CosmWasm code upload, contract deploy/execute/query, JSON schema forms, CW token types | `/codes/*`, `/contracts/*`, `/verify` (CosmWasm tab) |
| **Custom (unique)** | Bridge, oracle, milestones, settlements, certification, slot validators, unified address, unified search | `/bridge/*`, `/oracle/*`, `/milestones/*`, `/settlements/*`, `/certification`, `/analytics`, `/search` |

---

## Token Standard Coverage (All 11 Standards)

| Standard | Runtime | Explorer Route | Status |
|---|---|---|---|
| **Native SDK coin** | Cosmos | `/address/:any` — balance, transfers, delegations | 🔧 NEEDS WIRING |
| **ICS-20 / IBC token** | Cosmos | `/ibc/assets` — denom trace, origin chain, circulating amount | ✅ IMPLEMENTED |
| **CW-20** | CosmWasm | `/contracts/:addr` — holders tab + transfers list | 🔧 NEEDS WIRING |
| **CW-721** | CosmWasm | `/contracts/:addr/nfts` — gallery, per-token, owner history | ✅ IMPLEMENTED |
| **CW-1155** | CosmWasm | `/contracts/:addr` — multi-token tab, token IDs, per-ID balances | ✅ IMPLEMENTED |
| **BEP-20 (locked)** | BSC→Bridge | `/bridge` supply invariant; `/bridge/tx/:nonce` BEP-20 detail | 🔧 NEEDS WIRING |
| **ERC-20** | EVM | `/evm/tokens` list + `/evm/tokens/:addr` detail | ✅ IMPLEMENTED |
| **ERC-721** | EVM | `/evm/nfts` + `/evm/nfts/:addr/:id` | ✅ IMPLEMENTED |
| **ERC-1155** | EVM | `/evm/tokens/:addr/multi` detail; `/evm/tokens/multi` list | 🔧 / ❌ MISSING list |
| **ERC-4626** | EVM | Auto-detected on `/evm/contracts/:addr` | 🔧 NEEDS WIRING |
| **Bridged token (Cosmos-minted)** | Cosmos | `/bridge` supply gauge + address page | 🔧 NEEDS WIRING |

---

## Phase 1 — Foundation (Weeks 1–5)
**Goal:** Real block/tx/account data live. No mock data. Consensus visible. Faucet working.
🔵 *Replaces: Ping.pub (blocks, txs, accounts, consensus, faucet, network config)*

### Phase 1 Routes

| Route | Description | Status |
|---|---|---|
| `/` | Home: live block ticker, TPS, validator count, recent blocks + txs feed, 14D sparkline, SOV price, global search | 🔧 NEEDS WIRING — API connected but stats panel incomplete (market cap, sparkline missing) |
| `/blocks` | Block list: height, time, proposer, tx count, gas used %, base fee, burnt fees, network utilization banner | 🔧 NEEDS WIRING — missing gas %, burnt fees, utilization banner columns |
| `/blocks/:height` | Block detail: proposer, txs decoded, ABCI events, gas, consensus info tab (round/step/pre-commit signatures), parent hash, state root | 🔧 NEEDS WIRING — consensus info tab missing |
| `/txs` | Tx list: filter by Cosmos / EVM / CosmWasm; method badge, IN/OUT badge, direction arrow, summary banner | 🔧 NEEDS WIRING — filter + method badge incomplete |
| `/txs/:hash` | Tx detail: all Msg types decoded, action banner, token transfers section, internal txs (EVM), event logs, input data raw↔decoded, gas breakdown, burnt fees | 🔧 NEEDS WIRING — action banner, token transfers section, event logs tab missing |
| `/txs/pending` | Mempool pending tx list: method badge, nonce, last-seen, gas price breakdown, total count header | 🔧 NEEDS WIRING — real mempool feed not connected |
| `/address/:any` | Unified address: bech32 + 0x, native balance + USD, token holdings dropdown, tabs (Txs / Tokens / NFTs / Stake / Analytics), funded-by, first/last seen, vesting schedule | 🔧 NEEDS WIRING — EVM portfolio, NFT gallery, CSV export, watchlist star not wired |
| `/consensus` | Live CometBFT consensus round: height, round, step, per-validator vote status, power, time in step — WebSocket | ✅ IMPLEMENTED |
| `/faucet` | Testnet faucet: bech32 input, drip limit, disabled on mainnet via env flag | 🔧 NEEDS WIRING — backend faucet endpoint not connected |
| `/params` | All chain parameters live from gRPC: staking, slashing, governance, oracle, bridge, milestone params | 🔧 NEEDS WIRING — some param groups not yet fetched |
| `/network` | Network config hub: EVM Chain ID, all RPC URLs, "Add to MetaMask" one-click, "Add to Keplr" one-click, copyable snippets | ✅ IMPLEMENTED |
| `/search` | Unified search results: blocks, txs, addresses, validators, contracts, proposals, NFTs — 7 entity types | 🔧 NEEDS WIRING — server-side search not connected (pg_trgm search_index) |
| `/status` | System health: indexer lag, Blockscout lag, NATS health, API p95 — all live | 🔧 NEEDS WIRING — metrics endpoint not connected |

### Phase 1 — Home Page Missing Fields (add to `/`)

| Field | Description |
|---|---|
| SOV/USD price + 24H % | From x/oracle `SOV/USD` feed — persistent in global header |
| Market cap | Circulating supply × SOV/USD |
| Total transactions (all-time) | From `explorer.transactions` count |
| Median gas price | From x/feemarket base fee |
| Last finalized block | CometBFT last committed height |
| 14-day tx sparkline | From `block_stats` daily aggregate |
| Block reward column | Proposer reward in SOV on block list |
| Tx method badge | Cosmos Msg type short label on tx list |

### Phase 1 — Blocks Page Missing Fields (add to `/blocks`)

| Field | Description |
|---|---|
| Gas used % + mini progress bar | Gas used / Gas limit |
| Base fee | uSLT from x/feemarket |
| Burnt fees | uSLT burned per block |
| Network utilization banner | 4-stat card row above table |
| Fee Recipient label | Validator moniker linked |

### Phase 1 — Global Header (every page)

| Element | Status |
|---|---|
| SOV/USD price + 24H % (persistent top bar) | 🔧 NEEDS WIRING — oracle price not in header |
| Gas price (persistent) | 🔧 NEEDS WIRING |
| Dark/light mode toggle | ✅ IMPLEMENTED |
| Global search bar with type filter dropdown | 🔧 NEEDS WIRING — search not connected |

---

## Phase 2 — Custom Modules + CosmWasm + Governance + IBC (Weeks 6–12)
**Goal:** All 7 custom modules live. CosmWasm fully interactive. Governance and IBC live.
🔵 *Replaces: Celatone (CosmWasm), Ping.pub (governance, IBC, staking actions), Mintscan-level validator detail*

### Phase 2 Routes

| Route | Description | Status |
|---|---|---|
| `/validators` | 30-slot equal-power visual grid: slot #, occupant, uptime badge, certification status | ✅ IMPLEMENTED |
| `/validators/:addr` | Full detail: slot + partition, per-block signing heatmap, certification panel, oracle participation, governance votes, delegation list, ejection timeline | 🔧 NEEDS WIRING — signing heatmap, oracle participation not connected |
| `/certification` | Chain-wide: degraded mode live badge, per-validator attestation score, window size param | ✅ IMPLEMENTED |
| `/staking` | Dashboard: total bonded, APR, inflation, community pool, APR calculator, top validators | ✅ IMPLEMENTED |
| `/staking/history` | Validator set change timeline: slot fills, ejections, tombstones — block height + reason | 🔧 NEEDS WIRING — slot events not indexed yet |
| `/oracle` | Oracle dashboard: all feeds, latest price, staleness badge, participation rate, staleness banner | ✅ IMPLEMENTED |
| `/oracle/:feedId` | Feed detail: OHLC candlestick chart, animated staleness state machine, round breakdown, per-validator table, slash events | ✅ IMPLEMENTED |
| `/oracle/rounds/:roundId` | Round detail: commit phase (hash list), reveal phase (prices), aggregated median, pre-image hash | ✅ IMPLEMENTED |
| `/oracle/operators` | All oracle operator addresses: performance score, commit/reveal rates, slash history | ✅ IMPLEMENTED |
| `/milestones` | List: filter by state, oracle feed column, deadline countdown, achieved/expired ratio | ✅ IMPLEMENTED |
| `/milestones/:id` | Detail: animated state machine, oracle feed linked, price vs target, deadline timeline with pause periods | ✅ IMPLEMENTED |
| `/settlements` | Settlement record list: filter by witness / status / time | ✅ IMPLEMENTED |
| `/settlements/:id` | Ed25519 detail: domain separator breakdown, chain_id binding proof, ±30s timestamp display | ✅ IMPLEMENTED |
| `/governance` | Proposal list: filter by status, live progress bar for active proposals, Constitution badge | 🔧 NEEDS WIRING — filter incomplete |
| `/governance/:id` | Detail: custom type badge, Constitution check result (live CosmWasm query), voting tally, validator vote breakdown | 🔧 NEEDS WIRING — Constitution check not wired |
| `/governance/submit` | Proposal form: type selector, Constitution pre-check simulation, Keplr broadcast | 🔧 NEEDS WIRING — Constitution pre-check not implemented |
| `/ibc` | IBC overview: channel count, connection count, packet volume | ✅ IMPLEMENTED |
| `/ibc/channels` | Channel list: counterparty, state, port, ordering, packet count | ✅ IMPLEMENTED |
| `/ibc/channels/:channelId` | Packet tracker: sent → received → acknowledged; stuck packet alerts; relayer address | ✅ IMPLEMENTED |
| `/ibc/assets` | IBC denom trace list: all foreign tokens, origin chain, IBC path, denom hash, circulating amount | ✅ IMPLEMENTED |
| `/contracts` | All instantiated contracts: code ID, label, admin, creator, type badge (Constitution / Treasury / Reserve / Governance / CW-20 / CW-721 / CW-1155 / Custom) | ✅ IMPLEMENTED |
| `/contracts/:addr` | Contract detail: execute history, migration/admin history, CW-20 holders tab, CW-721 NFT gallery tab, CW-1155 multi-token tab | 🔧 NEEDS WIRING — CW-20 holders tab not populated |
| `/contracts/:addr/query` | Full-screen query tool: JSON form from QueryMsg schema, response pretty-printed | ✅ IMPLEMENTED |
| `/contracts/:addr/execute` | Full-screen execute: multi-wallet (Keplr/Leap/Cosmostation), gas simulation, broadcast, response decoded | 🔧 NEEDS WIRING — gas simulation incomplete |
| `/contracts/:addr/nfts` | CW-721 NFT collection: token ID grid, per-token metadata + IPFS image, owner history, transfer history | ✅ IMPLEMENTED |
| `/contracts/:addr/nfts/:tokenId` | CW-721 individual NFT: metadata, IPFS image, owner, transfer history | ✅ IMPLEMENTED |
| `/codes` | Wasm code list: uploader, block, checksum, instantiation count; search by checksum | ✅ IMPLEMENTED |
| `/codes/:codeId` | Code detail: instantiation list, checksum, uploader, deploy tx; JSON schema upload | ✅ IMPLEMENTED |
| `/codes/submit` | Wasm binary SHA256 checksum verify + JSON schema upload for execute/query form auto-build | 🔧 NEEDS WIRING — checksum verify vs on-chain DataHash not implemented |
| `/address/:any/send` | Token send form: amount + recipient, fee selector, memo; Keplr/Leap sign + broadcast | 🔧 NEEDS WIRING — wallet broadcast not fully wired |
| `/address/:any/stake` | Staking actions: delegate / undelegate / redelegate / claim rewards; slot-based validator picker | 🔧 NEEDS WIRING — slot picker not connected |

### Phase 2 — Custom Module Backend Event Decoders Required

| Module | Events to Decode | DB Tables |
|---|---|---|
| x/validator | SlotFilled, SlotEjected, ValidatorSlashed | `explorer.validator_slots`, `explorer.slot_events` |
| x/certification | AttestationUpdated, DegradedModeChanged | `explorer.certification_scores` |
| x/oracle | CommitReceived, RevealReceived, PriceAggregated, OracleSlashed | `explorer.oracle_rounds`, `explorer.oracle_commits`, `explorer.oracle_reveals` |
| x/milestone | MilestoneCreated, StateTransitioned, DeadlinePaused | `explorer.milestones`, `explorer.milestone_events` |
| x/settlement | SettlementRecorded | `explorer.settlements` |

---

## Phase 3 — Bridge + EVM Tokens + Analytics (Weeks 13–17)
**Goal:** Bridge fully trackable end-to-end. All EVM tokens live. Analytics powered by TimescaleDB.
🔵 *Replaces: Blockscout (EVM tokens, contracts, charts), custom bridge pages (unique)*

### Phase 3 Routes

| Route | Description | Status |
|---|---|---|
| `/bridge` | Dashboard: live supply invariant gauge (cosmos_minted + bsc_circulating = S), 24H volume chart, pending tx count, circuit-breaker status, active relayers | 🔧 NEEDS WIRING — supply invariant gauge mock data |
| `/bridge/deposit` | BSC→Cosmos: all LockBox lock events, status (locked/confirming/confirmed/minting/minted), confirmation tier badge (Standard 15 blocks vs High-Value 50 blocks) | 🔧 NEEDS WIRING — BSC watcher not connected to UI |
| `/bridge/withdraw` | Cosmos→BSC: all MsgBridgeOut events, status (burn/relaying/released), BSC release tx hash | 🔧 NEEDS WIRING — bridge events not flowing |
| `/bridge/tx/:nonce` | Full bridge lifecycle: BSC lock hash, animated confirmation progress bar, quorum signature tracker (N of T), Cosmos MsgBridgeIn tx hash, bitmap nonce position, total time lock→mint | 🔧 NEEDS WIRING — lifecycle view uses mock data |
| `/bridge/relayers` | Relayer set: promotion ladder (Primary/Secondary/Candidate), miss count, last active block, governance update history | ❌ MISSING — page not created |
| `/bridge/nonces` | Bitmap nonce registry: used nonces as compressed bitmap, in-flight nonces, expiry queue | ❌ MISSING — page not created |
| `/bridge/history` | Circuit-breaker history: all pause/unpause events, EOA that triggered, duration | ✅ IMPLEMENTED |
| `/evm` | EVM overview: gas tracker tiers, mempool size, pending tx count | ✅ IMPLEMENTED |
| `/evm/blocks` | EVM block list (via Blockscout API) | ✅ IMPLEMENTED |
| `/evm/blocks/:height` | EVM block detail: gas used, reward, miner, txs list | ✅ IMPLEMENTED |
| `/evm/txs` | EVM tx list: method badge, token transfers, IN/OUT | ✅ IMPLEMENTED |
| `/evm/txs/:hash` | EVM tx detail: ABI-decoded input, internal txs call tree, revert reason, token transfers section, event logs | ✅ IMPLEMENTED |
| `/evm/contracts` | Verified Solidity contract list: name, compiler, language, balance, txns, verified date | ✅ IMPLEMENTED |
| `/evm/contracts/:addr` | Source code (syntax highlighted), ABI, Read contract, Write contract (MetaMask), proxy detection, ERC-4626 vault interface, bytecode, opcodes, constructor args | 🔧 NEEDS WIRING — ERC-4626 detection, proxy Read/Write as Proxy tabs |
| `/evm/contracts/:addr/read` | Read contract: view/pure functions, grouped, return types labeled | ✅ IMPLEMENTED |
| `/evm/contracts/:addr/write` | Write contract: payable amount field, MetaMask connect, gas override, tx preview, result | 🔧 NEEDS WIRING — wallet connect flow incomplete |
| `/evm/tokens` | ERC-20 token list: price, 24H change, volume, onchain market cap, holders + sparkline | ✅ IMPLEMENTED |
| `/evm/tokens/:addr` | ERC-20 detail: holders donut, transfer history, price chart, burn/mint events, social links, reputation badge | ✅ IMPLEMENTED |
| `/evm/tokens/:addr/multi` | ERC-1155 detail: token ID list, per-ID circulating supply, batch transfer history, per-ID holders | 🔧 NEEDS WIRING — placeholder data |
| `/evm/tokens/multi` | ERC-1155 top tokens list (all collections) | ❌ MISSING |
| `/evm/nfts` | ERC-721 collection list: floor price (oracle), item count, owner count, volume | ✅ IMPLEMENTED |
| `/evm/nfts/:addr` | ERC-721 collection detail: gallery grid, trait filters, volume chart | ✅ IMPLEMENTED |
| `/evm/nfts/:addr/:id` | ERC-721 NFT detail: on-chain metadata, IPFS image, owner history, transfer history, trait list | ✅ IMPLEMENTED |
| `/evm/addresses/:addr` | EVM-only address view: 0x balance, ERC-20 portfolio, NFT gallery, tx history | ✅ IMPLEMENTED |
| `/evm/verify` | EVM contract source verification form: address, compiler, optimizer, constructor args, upload source | 🔧 NEEDS WIRING — Blockscout verifier API not connected |
| `/address/:any` | **Updated** — Unified dual-view: bech32 + 0x, native + EVM balance, delegations, ERC-20 + CW-20 portfolio, ERC-721 + CW-721 NFTs, tx history (both runtimes), vesting schedule, CSV export, webhook subscribe | 🔧 NEEDS WIRING (see Phase 2 entry) |
| `/analytics` | Full analytics dashboard: TPS, block time histogram, oracle OHLC per feed, validator uptime heatmap (slot×day), bridge volume, settlement volume, milestone achievement rate, active address growth — all from TimescaleDB, CSV export per chart | 🔧 NEEDS WIRING — charts have placeholder data, TimescaleDB aggregates not connected |
| `/developers` | Developer hub: all RPC/gRPC/WS/LCD endpoints, MetaMask+Keplr one-click, tabbed code snippets (Hardhat/Foundry/Remix/ethers.js/viem/wagmi/CosmJS/wasmd), deploy flow SVG diagram, links to `/verify` and `/docs` | ✅ IMPLEMENTED |
| `/verify` | Unified contract verification: EVM tab (Sourcify auto-check + manual Blockscout verifier, 7 methods) + CosmWasm tab (SHA256 checksum vs on-chain DataHash + JSON schema upload → auto-build execute/query forms) | 🔧 NEEDS WIRING — both tabs not connected to backends |

### Phase 3 — Bridge Backend Work Required

| Source | Events to Index | DB Tables |
|---|---|---|
| x/bridge (Cosmos) | BridgeInExecuted, BridgeOutInitiated, RelayerPromoted, CircuitBreakerTriggered | `explorer.bridge_txs`, `explorer.relayers`, `explorer.circuit_breaker_events` |
| LockBox.sol (BSC) | `Lock(address,uint256,uint64)`, `Release(address,uint256,uint64)` | `explorer.bsc_lock_events` |

BSC watcher goroutine: polls `eth_getLogs` for LockBox every 2 seconds; matches BSC events to Cosmos MsgBridgeIn by nonce.

---

## Phase 4 — Hardening + Public API + Decommission (Weeks 18–20)
**Goal:** Production-ready. Load-tested. Public API documented. All 3 tools decommissioned.

### Phase 4 Routes

| Route | Description | Status |
|---|---|---|
| `/docs` | Public API documentation (auto-generated from OpenAPI spec via Redoc or Stoplight Elements) | ✅ IMPLEMENTED |
| `/status` | System health: indexer lag, Blockscout lag, NATS health, API p95 | 🔧 NEEDS WIRING (see Phase 1) |

---

## Appendix E Routes — Etherscan Parity (37 New Routes)

All routes below are ❌ MISSING unless noted. Priority labels: 🔴 High / 🟡 Medium / 🟢 Low

### Charts & Statistics

| Route | Description | Priority | Status |
|---|---|---|---|
| `/charts` | Charts hub: sidebar with all 22 chart categories, overview stats panel | 🟡 Medium | ❌ MISSING |
| `/charts/tx` | Daily Transaction Count chart | 🟡 Medium | ❌ MISSING |
| `/charts/tx-fee` | Average Transaction Fee (USD) chart | 🟡 Medium | ❌ MISSING |
| `/charts/active-addresses` | Daily Active Addresses chart | 🟡 Medium | ❌ MISSING |
| `/charts/new-addresses` | Daily New Addresses chart | 🟡 Medium | ❌ MISSING |
| `/charts/unique-addresses` | Cumulative Unique Addresses chart | 🟡 Medium | ❌ MISSING |
| `/charts/gas-price` | Average Gas Price (uSLT) chart | 🟡 Medium | ❌ MISSING |
| `/charts/gas-used` | Total Gas Used per Day chart | 🟡 Medium | ❌ MISSING |
| `/charts/block-size` | Average Block Size (bytes) chart | 🟡 Medium | ❌ MISSING |
| `/charts/block-time` | Average Block Time (seconds) chart | 🟡 Medium | ❌ MISSING |
| `/charts/tps` | Transactions Per Second chart | 🟡 Medium | ❌ MISSING |
| `/charts/block-count` | Blocks per Day chart | 🟡 Medium | ❌ MISSING |
| `/charts/burnt-fees` | Daily Burnt Fees (SOV) chart | 🟡 Medium | ❌ MISSING |
| `/charts/price` | SOV Daily Price (USD) chart | 🟡 Medium | ❌ MISSING |
| `/charts/market-cap` | SOV Market Capitalization chart | 🟡 Medium | ❌ MISSING |
| `/charts/validator-count` | Active Validator Count chart | 🟡 Medium | ❌ MISSING |
| `/charts/staking-ratio` | Staking Ratio (% bonded) chart | 🟡 Medium | ❌ MISSING |
| `/charts/contracts-deployed` | Daily Contracts Deployed chart | 🟡 Medium | ❌ MISSING |
| `/charts/contracts-verified` | Daily Contracts Verified chart | 🟡 Medium | ❌ MISSING |
| `/charts/token-transfers` | Daily ERC-20 + CW-20 Transfers chart | 🟡 Medium | ❌ MISSING |
| `/charts/nft-transfers` | Daily NFT Transfers chart | 🟡 Medium | ❌ MISSING |
| `/charts/bridge-volume` | Daily Bridge Volume (SOV) chart | 🔴 High | ❌ MISSING |
| `/charts/ibc-volume` | Daily IBC Transfer Volume chart | 🟡 Medium | ❌ MISSING |

Each chart page has: date range selector (7D/30D/90D/180D/1Y/All), CSV download, description, data source label.

### Key Metrics & Leaderboards

| Route | Description | Priority | Status |
|---|---|---|---|
| `/stat/supply` | SOV total supply + distribution breakdown (genesis, staking rewards, burnt, circulating) + pie chart | 🟡 Medium | ❌ MISSING |
| `/accounts` | Top addresses by SOV balance: rank, address + label, name tag, balance + USD, % of supply, tx count | 🟡 Medium | ❌ MISSING |
| `/gastracker` | Gas price tracker: 3-tier cards (slow/fast/rapid), featured action costs table, gas heatmap (hour×day), price history chart | 🟡 Medium | ❌ MISSING |
| `/nfts` | Top NFT collections leaderboard (CW-721 + ERC-721 + ERC-1155): volume, sales, floor, owners, transfers | 🟡 Medium | ❌ MISSING |

### Contracts & Transactions

| Route | Description | Priority | Status |
|---|---|---|---|
| `/contracts/verified` | Recently verified contracts list: address, name, compiler, language, balance, txns, verified date, audit badge | 🟡 Medium | ❌ MISSING |
| `/contracts/verified/:type` | Verified contracts filtered by type (EVM / CosmWasm / Proxy) | 🟢 Low | ❌ MISSING |
| `/txs/internal` | All internal EVM transactions chain-wide: parent tx hash, type (call/create/delegatecall), from, to, value | 🟡 Medium | ❌ MISSING |

### User & Account System

| Route | Description | Priority | Status |
|---|---|---|---|
| `/token-approvals` | ERC-20 approval tracker: connect wallet → list all active approvals, revoke button | 🟡 Medium | ❌ MISSING |
| `/address/:addr/tokencheck` | Token approval checker for specific address | 🟡 Medium | ❌ MISSING |
| `/label/:slug` | All addresses with a given label tag (e.g. `/label/treasury`) | 🟡 Medium | ❌ MISSING |
| `/myaccount` | User account dashboard: watchlist, private name tags, private tx notes, API key management, token ignore list, notification settings | 🟢 Low | ❌ MISSING |
| `/pushnotification` | Watchlist + push notification setup (email/webhook on watched address events) | 🟢 Low | ❌ MISSING |
| `/exportData` | Bulk data export: address + date range → CSV download | 🟡 Medium | ❌ MISSING |

---

## Appendix F Routes — Etherscan Second-Pass (30 New Routes)

### Token Transfer Chain-wide Pages

| Route | Description | Priority | Status |
|---|---|---|---|
| `/txs/erc20` | All ERC-20 + CW-20 transfers chain-wide: from, to, token, amount, tx hash, age | 🟡 Medium | ❌ MISSING |
| `/txs/erc721` | All ERC-721 + CW-721 transfers chain-wide: from, to, collection, token ID, tx hash | 🟡 Medium | ❌ MISSING |
| `/txs/erc1155` | All ERC-1155 + CW-1155 transfers chain-wide: from, to, token ID, amount, tx hash | 🟡 Medium | ❌ MISSING |
| `/txs/withdrawals` | Staking undelegation completions: address, amount, block, completion time | 🟡 Medium | ❌ MISSING |
| `/txs/advanced-filter` | Standalone advanced tx filter: from/to, block range, date range, value range, tx type, method, token, status, CSV export | 🟡 Medium | ❌ MISSING |
| `/blocks/reorgs` | CometBFT height mismatch events (note: instant finality means this is rare/empty) | 🟢 Low | ❌ MISSING |

### NFT Activity Pages

| Route | Description | Priority | Status |
|---|---|---|---|
| `/nfts/top-mints` | Collections by mint activity: collection, mints, unique minters, max price, avg price, total volume; time filter 1H/6H/12H/24H/7D | 🟢 Low | ❌ MISSING |
| `/nfts/latest-trades` | Real-time NFT trade feed: collection, token ID, buyer, seller, price in SOV + USD | 🟢 Low | ❌ MISSING |
| `/nfts/latest-transfers` | Real-time NFT transfer feed: collection, token ID, from, to, block, age | 🟢 Low | ❌ MISSING |
| `/nfts/latest-mints` | Real-time mint feed: collection, token ID, minter, price paid, tx hash, age | 🟢 Low | ❌ MISSING |

### Gas Pages

| Route | Description | Priority | Status |
|---|---|---|---|
| `/gas/guzzlers` | Top 25 contracts by total gas consumed: rank, contract label, %, total gas, tx count; 24H/3D/7D tabs | 🟡 Medium | ❌ MISSING |
| `/gas/spenders` | Top 25 addresses by total gas fees paid: rank, address, %, total SOV spent, tx count | 🟡 Medium | ❌ MISSING |

### Discovery Pages

| Route | Description | Priority | Status |
|---|---|---|---|
| `/stats` | Top statistics leaderboard: top validators (blocks proposed), top gas consumers, top tx senders, top SOV holders, chain milestones | 🟡 Medium | ❌ MISSING |
| `/directory` | Curated project directory: DeFi / NFT / Bridge / Oracle / Governance — logo, description, contract links, verified badge | 🟢 Low | ❌ MISSING |
| `/labelcloud` | Visual tag cloud of all address labels; size = address count; click → `/label/:slug` | 🟢 Low | ❌ MISSING |
| `/domains` | Name service registry (`.sov` name service if deployed); address ↔ name lookup | 🟢 Low | ❌ MISSING |
| `/api-plans` | API rate tier table (Free / Standard / Pro); API key generation | 🟢 Low | ❌ MISSING |

### Developer Tools Hub

| Route | Description | Priority | Status |
|---|---|---|---|
| `/tools` | Tools hub page linking to all sub-tools | 🟡 Medium | ❌ MISSING |
| `/tools/unit-converter` | SOV ↔ uSOV ↔ Wei ↔ Gwei ↔ USD conversion with live oracle price | 🟡 Medium | ❌ MISSING |
| `/tools/broadcast` | Broadcast raw signed tx: EVM hex or Cosmos base64 JSON; decode-first preview; submit to RPC | 🟡 Medium | ❌ MISSING |
| `/tools/abi-encoder` | ABI encoder/decoder: paste ABI + function + args → calldata; decode calldata → readable | 🟡 Medium | ❌ MISSING |
| `/tools/disassembler` | EVM bytecode disassembler: hex input → opcode listing with PC offsets | 🟢 Low | ❌ MISSING |
| `/tools/similar-contracts` | Similar contracts lookup: input bytecode hash or address → all contracts with same bytecode | 🟢 Low | ❌ MISSING |
| `/tools/contract-diff` | Contract diff checker: compare two contract addresses side-by-side, source diff highlighted | 🟢 Low | ❌ MISSING |
| `/tools/constructor-args` | Constructor argument decoder: decode ABI-encoded creation calldata given address + ABI | 🟢 Low | ❌ MISSING |
| `/tools/verify-signature` | Verified signature lookup: verify EIP-191 (EVM) or Cosmos ADR-036 signed message | 🟢 Low | ❌ MISSING |
| `/tools/compiler` | Browser-based Solidity compiler (Remix-lite) + CosmWasm schema validator | 🟢 Low | ❌ MISSING |
| `/tools/code-reader` | AI-powered contract explanation: paste address → plain-English summary | 🟢 Low | ❌ MISSING |

---

## Complete Route Status Summary

### ✅ IMPLEMENTED (real data flows) — 39 routes
`/consensus`, `/network`, `/oracle`, `/oracle/:feedId`, `/oracle/rounds/:roundId`, `/oracle/operators`, `/milestones`, `/milestones/:id`, `/settlements`, `/settlements/:id`, `/certification`, `/validators`, `/staking`, `/ibc`, `/ibc/channels`, `/ibc/channels/:channelId`, `/ibc/assets`, `/contracts`, `/contracts/:addr/query`, `/contracts/:addr/nfts`, `/contracts/:addr/nfts/:tokenId`, `/codes`, `/codes/:codeId`, `/evm`, `/evm/blocks`, `/evm/blocks/:height`, `/evm/txs`, `/evm/txs/:hash`, `/evm/contracts`, `/evm/contracts/:addr/read`, `/evm/tokens`, `/evm/tokens/:addr`, `/evm/nfts`, `/evm/nfts/:addr`, `/evm/nfts/:addr/:id`, `/evm/addresses/:addr`, `/bridge/history`, `/developers`, `/docs`

### 🔧 NEEDS WIRING (page exists, data mocked) — 31 routes
`/` (home stats panel incomplete), `/blocks` (missing gas%/burnt-fees columns), `/blocks/:height` (consensus info tab missing), `/txs` (type filter + method badge), `/txs/:hash` (action banner, token-transfers section, event logs tab), `/txs/pending` (mempool not connected), `/address/:any` (EVM portfolio, NFT gallery, CSV export), `/address/:any/send` (wallet broadcast), `/address/:any/stake` (slot picker), `/faucet` (backend endpoint), `/params` (some module groups missing), `/search` (pg_trgm search_index not connected), `/status` (metrics endpoint not connected), `/governance` (filter incomplete), `/governance/:id` (Constitution check not wired), `/governance/submit` (Constitution pre-check not implemented), `/validators/:addr` (signing heatmap, oracle participation), `/staking/history` (slot events not indexed), `/contracts/:addr` (CW-20 holders tab unpopulated), `/contracts/:addr/execute` (gas simulation incomplete), `/codes/submit` (SHA256 vs on-chain DataHash not implemented), `/bridge` (supply invariant gauge mocked), `/bridge/deposit` (BSC watcher not connected to UI), `/bridge/withdraw` (bridge events not flowing), `/bridge/tx/:nonce` (lifecycle uses mock data), `/evm/contracts/:addr` (ERC-4626 detection, proxy tabs), `/evm/contracts/:addr/write` (wallet connect flow incomplete), `/evm/tokens/:addr/multi` (placeholder data), `/evm/verify` (Blockscout verifier API not connected), `/verify` (both EVM + CosmWasm tabs not connected), `/analytics` (TimescaleDB aggregates not connected)

### ❌ MISSING (not in codebase) — 58 routes
`/bridge/relayers`, `/bridge/nonces`, `/evm/tokens/multi`, `/charts` (hub), `/charts/tx`, `/charts/tx-fee`, `/charts/active-addresses`, `/charts/new-addresses`, `/charts/unique-addresses`, `/charts/gas-price`, `/charts/gas-used`, `/charts/block-size`, `/charts/block-time`, `/charts/tps`, `/charts/block-count`, `/charts/burnt-fees`, `/charts/price`, `/charts/market-cap`, `/charts/validator-count`, `/charts/staking-ratio`, `/charts/contracts-deployed`, `/charts/contracts-verified`, `/charts/token-transfers`, `/charts/nft-transfers`, `/charts/bridge-volume`, `/charts/ibc-volume` (23 chart routes total), `/gastracker`, `/nfts`, `/stat/supply`, `/accounts`, `/contracts/verified`, `/contracts/verified/:type`, `/txs/internal`, `/txs/erc20`, `/txs/erc721`, `/txs/erc1155`, `/txs/withdrawals`, `/txs/advanced-filter`, `/blocks/reorgs`, `/nfts/top-mints`, `/nfts/latest-trades`, `/nfts/latest-transfers`, `/nfts/latest-mints`, `/gas/guzzlers`, `/gas/spenders`, `/stats`, `/directory`, `/labelcloud`, `/domains`, `/api-plans`, `/token-approvals`, `/address/:addr/tokencheck`, `/label/:slug`, `/myaccount`, `/pushnotification`, `/exportData`, `/tools`, `/tools/unit-converter`, `/tools/broadcast`, `/tools/abi-encoder`, `/tools/disassembler`, `/tools/similar-contracts`, `/tools/contract-diff`, `/tools/constructor-args`, `/tools/verify-signature`, `/tools/compiler`, `/tools/code-reader`

---

## Gap Closure Priority Order

### 🔴 Priority 1 — Wire existing placeholder pages (no new routes needed)
These are the most impactful because the pages already exist — only the backend connection is missing.

1. **`/bridge`** + `/bridge/deposit` + `/bridge/withdraw` + `/bridge/tx/:nonce` — connect BSC watcher + bridge event decoder to UI
2. **`/address/:any`** — wire EVM portfolio (Blockscout API), CW-20 holdings, NFT gallery, CSV export
3. **`/governance/:id`** — wire Constitution check (live CosmWasm query to constitution.wasm)
4. **`/validators/:addr`** — wire signing heatmap (block-by-block signing data) + oracle participation
5. **`/analytics`** — connect TimescaleDB aggregates (`tps_1h`, `oracle_price_1h`, `validator_uptime_1d`, `bridge_volume_1h`)
6. **`/verify`** — wire EVM tab to Blockscout verifier API + CosmWasm tab to SHA256 check + schema store
7. **`/search`** — wire PostgreSQL `pg_trgm` search across all 7 entity types
8. **`/status`** — wire indexer lag + Blockscout lag + NATS health metrics

### 🔴 Priority 2 — Add missing bridge routes (core chain UX)
9. **`/bridge/relayers`** — create page; wire to `explorer.relayers` table
10. **`/bridge/nonces`** — create page; wire to bitmap nonce registry

### 🟡 Priority 3 — Add Etherscan-parity charts
11. **`/charts`** hub + all 22 `/charts/:name` sub-pages — wire to `block_stats`, `daily_fees`, `daily_addresses` TimescaleDB aggregates
12. **`/gastracker`** — wire to x/feemarket base fee + mempool stats
13. **`/stat/supply`** — wire to genesis supply + staking module supply query
14. **`/accounts`** — wire to top-balance query from `explorer.accounts`

### 🟡 Priority 4 — Standard explorer pages
15. **`/txs/erc20`** + `/txs/erc721` + `/txs/erc1155` — chain-wide token transfer feeds from Blockscout
16. **`/contracts/verified`** — verified contract list from Blockscout
17. **`/nfts`** — NFT leaderboard combining CW-721 + ERC-721
18. **`/tools/unit-converter`** + `/tools/broadcast` — basic developer utilities

### 🟢 Priority 5 — Nice-to-have (Phase 4 / post-launch)
19. `/myaccount` + `/pushnotification` — user account system (wallet-based sign-in)
20. `/tools/abi-encoder` + `/tools/disassembler` + `/tools/similar-contracts` + `/tools/contract-diff`
21. `/nfts/top-mints` + `/nfts/latest-trades` + `/nfts/latest-transfers` + `/nfts/latest-mints`
22. `/directory` + `/labelcloud` + `/api-plans` + `/domains`
23. `/tools/code-reader` — AI contract explanation

---

## Unique Features vs Off-the-Shelf Tools

These are features only the Sovereign L1 custom explorer provides — none of the 3 tools can replicate them:

| Feature | Why Unique |
|---|---|
| ★ Cosmos + EVM unified address page | One URL shows both runtimes |
| ★ 30-slot equal-power validator grid | Custom x/validator (not stake-weighted) |
| ★ x/certification: attestation + degraded mode | Custom ABCI++ module |
| ★ x/oracle: commit/reveal round detail | Custom two-phase commit oracle |
| ★ Oracle staleness animated state machine | 3-state: fresh → stale → stale-blocked |
| ★ x/milestone: state machine + deadline pause timeline | Oracle-gated DeFi goals |
| ★ x/settlement: Ed25519 domain separator inspector | Chain_id binding proof display |
| ★ BSC bridge: lock → quorum → mint end-to-end | Cross-chain lifecycle in one view |
| ★ Bridge supply invariant live gauge | `cosmos_minted + bsc_circulating = S` |
| ★ Relayer promotion ladder visualisation | Primary/Secondary/Candidate tiers |
| ★ Bridge circuit-breaker history | Pause/unpause log with EOA + duration |
| ★ Nonce bitmap registry viewer | Compressed bitmap display of used nonces |
| ★ Confirmation tier badge (15 vs 50 blocks) | Standard vs High-Value finality |
| ★ gRPC server-streaming (no polling) | `StreamLatestBlocks`, `StreamConsensusRound` |
| ★ CosmWasm execute/query with gas simulation | Celatone-class CosmWasm DX |
| ★ CW-1155 multi-token detail view | Rarely supported anywhere |
| ★ ERC-4626 vault auto-detection + read interface | Beyond Etherscan standard |
| ★ IBC stuck packet alerts | Cross-chain reliability indicator |
| ★ Constitution check on governance proposals | Live CosmWasm query on proposal detail |
| ★ Unified search across 7 entity types + both runtimes | Cosmos + EVM + CosmWasm in one bar |

---

## Architecture Reference

### Data Sources (6 feeds)

| Source | Port | Provides |
|---|---|---|
| CometBFT RPC | 26657 | Blocks, txs, validators, events, WebSocket, consensus rounds |
| Cosmos gRPC | 9090 | All 7 custom module queries, CosmWasm, governance, IBC |
| Backend gRPC-API | 9091 | TimescaleDB aggregates: tps_1h, block_time_1h, oracle_price_1h, validator_uptime_1d, bridge_volume_1h |
| EVM JSON-RPC | 8545/8546 WS | EVM blocks, txs, eth_getLogs, eth_call, newBlock subscription |
| BSC EVM RPC | external | LockBox contract events, BSC finality tracking |
| NATS JetStream | 4222 | Real-time push via `account:explorer` stream |

### Tech Stack

| Layer | Technology |
|---|---|
| Framework | Next.js 14 (App Router) — RSC for static pages, Client Components for live panels |
| Language | TypeScript 5.x strict |
| Styling | Tailwind CSS + shadcn/ui + next-themes |
| Charts | Recharts (time-series, candlesticks, heatmaps, bar charts) |
| Tables | TanStack Table v8 |
| Query | TanStack Query v5 |
| Forms | React Hook Form + Zod |
| Cosmos wallets | @cosmjs/stargate + @keplr-wallet/types (Keplr, Leap, Cosmostation) |
| EVM wallet | wagmi v2 + viem + RainbowKit |
| gRPC-Web | buf-generated stubs from @workspace/api-spec |
| State | Zustand (wallets, theme) + URL search params (filters) |

### Database Tables Required

```sql
-- Phase 1
explorer.blocks (height, time, proposer, tx_count, gas_used, gas_limit, app_hash)
explorer.transactions (hash, height, time, type, msg_types[], decoded JSONB, fee, gas_used, status)
explorer.accounts (address_bech32, address_hex, first_seen, last_active)

-- Phase 2 (custom modules)
explorer.validator_slots (slot, validator_addr, filled_at, ejected_at)
explorer.slot_events (slot, event_type, block_height, time)
explorer.certification_scores (validator_addr, score, window_start, degraded_mode BOOL)
explorer.oracle_rounds (round_id, feed_id, commit_hash, reveal_price, aggregated_median, time)
explorer.oracle_commits (round_id, validator_addr, commit_hash)
explorer.oracle_reveals (round_id, validator_addr, price, pre_image)
explorer.milestones (id, state, feed_id, target_price, deadline, achieved_at, paused_at)
explorer.milestone_events (milestone_id, event_type, block_height, time)
explorer.settlements (id, witness_addr, domain_sep, chain_id_binding, timestamp, proof_valid)

-- Phase 3 (bridge)
explorer.bridge_txs (nonce, direction, status, bsc_lock_hash, bsc_block, cosmos_mint_hash, cosmos_block, amount, sender, receiver)
explorer.bsc_lock_events (nonce, lock_hash, bsc_block, amount, sender, receiver, locked_at)
explorer.relayers (address, tier, miss_count, last_active_block)
explorer.circuit_breaker_events (event_type, trigger_address, block_height, time, duration_blocks)
```

### Caching Strategy

| Layer | TTL | What Is Cached |
|---|---|---|
| Redis | ~2s (1 block) | Latest block, active validator set, consensus round |
| Redis | 30s | Block/tx page rendered JSON (SSR cache) |
| Redis | Forever | Verified contract ABI, wasm checksums, address labels |
| CDN | 5 min | Static assets, prerendered SEO pages |
| PostgreSQL MV | Hourly | `explorer.search_index` for global search (pg_trgm) |

---

## Monitoring & Operations

### Prometheus Metrics

| Metric | Alert Threshold | Severity |
|---|---|---|
| `explorer_indexer_block_lag_seconds` > 10 blocks | P1 | Page on-call |
| `explorer_indexer_stopped` (60s no new block) | P0 | Restart pod |
| `explorer_api_rpc_duration_seconds` p95 > 500ms | P2 | Check TimescaleDB |
| `explorer_blockscout_sync_lag_blocks` > 100 | P2 | Restart Blockscout |
| `explorer_cache_hit_ratio` < 80% | P2 | Review TTL config |
| `explorer_frontend_ssr_duration_ms` p95 > 1s | P2 | Check RSC rendering |

### Load Test Targets (Phase 4)
- k6: 1,000 concurrent users, 10-minute ramp
- API p95 < 300ms
- SSR p95 < 500ms
- Search p95 < 300ms
- Redis cache hit ratio > 90% on all hot paths

---

## Open Questions (Resolved vs Pending)

| # | Question | Resolution |
|---|---|---|
| F.1 | Explorer artifact path | `/explorer` artifact (separate from dApp frontend) ✅ Done |
| F.2 | Explorer DB location | Share Read DB with new `explorer` schema (avoid 5th instance) ✅ Decided |
| F.3 | NATS account | New `account:explorer` (isolate consumers) ✅ Decided |
| F.4 | Blockscout deployment | Same k8s cluster, separate namespace, strict network policy ✅ Decided |
| F.5 | Public domain | Must decide before Phase 1: `explorer.sovereign.io` — needed for CORS + SEO from Day 1 ⚠️ PENDING |
| F.6 | Read-only Phase 1? | Read-only Phase 1; wallet actions start Phase 2 Week 8 ✅ Decided |
| F.7 | API keys required? | Open by default (10 req/s per IP); keys for higher limits (Phase 4) ✅ Decided |
| F.8 | Keep 3 tools running in parallel? | Yes — keep all three live until Phase 4 deliverable validated on testnet ✅ Decided |
| F.9 | CW-721 IPFS gateway | Public gateway (ipfs.io) for Phase 2; evaluate self-hosted in Phase 4 ✅ Decided |
| F.10 | ERC-4626 write forms | Read-only (totalAssets, convertToShares) Phase 3; write Phase 4 if needed ✅ Decided |

---

## Summary

| Dimension | Value |
|---|---|
| **Total routes** | 128 (61 core + 37 Appendix E + 30 Appendix F) |
| **✅ Implemented** | 39 routes (real data flows end-to-end) |
| **🔧 Needs wiring** | 31 routes (pages exist, backend connection incomplete) |
| **❌ Missing** | 58 routes (not yet created in codebase) |
| **Token standards covered** | 11 (native, ICS-20, CW-20, CW-721, CW-1155, BEP-20 bridged, ERC-20, ERC-721, ERC-1155, ERC-4626, Cosmos-minted bridged) |
| **Custom module pages** | 7 modules × multiple pages (bridge, oracle, milestone, settlement, certification, validator, governance-ext) |
| **Tools replaced** | Ping.pub + Blockscout (standalone) + Celatone |
| **Unique features (★)** | 20 features no off-the-shelf tool provides |
| **Data sources** | 6 (CometBFT, Cosmos gRPC, Backend gRPC-API, EVM JSON-RPC, BSC RPC, NATS) |
| **Phases** | 4 phases over 20 weeks |
| **Phase 1 delivers** | Real blocks/txs/accounts, consensus rounds, faucet, network config |
| **Phase 2 delivers** | All 7 custom modules, all Cosmos/CosmWasm token standards, IBC, governance, staking actions |
| **Phase 3 delivers** | Bridge end-to-end, all EVM tokens, analytics, developer hub, unified verification |
| **Phase 4 delivers** | Public API, global search, charts, hardening, decommission of 3 existing tools |

**Document Version:** 5.0
**Date Updated:** 2026-07-07
**Scope change from v4.0:** Complete A-to-Z rewrite. All 128 routes listed with ✅/🔧/❌ implementation status. Gap closure priorities added. Off-the-shelf tool replacement map added. Missing fields per page consolidated from Appendix E/F into inline route entries.
