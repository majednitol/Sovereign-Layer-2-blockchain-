# Sovereign L1 — Custom Enterprise Blockchain Explorer
## Complete Implementation Plan — Phase-First Structure

**Version:** 4.0 (developer & deployer experience added; RPC/verification/network-config coverage)  
**Date:** 2026-06-21  
**Scope:** Single unified explorer replacing Ping.pub + Blockscout + Celatone  
**Timeline:** 20 weeks across 4 phases  
**Pages:** 61 routes (58 from v3.0 + 3 new developer routes)

---

## Token Standard Coverage

The explorer covers **every token standard** on both runtimes, plus bridge tokens and CosmWasm tokens.

| Standard | Runtime | What It Is | Explorer Coverage | Phase |
|---|---|---|---|---|
| **Native SDK coin** | Cosmos | Chain's native token (staking, fees, bridge) | Balance, transfers, delegations, bridge history | 1 |
| **ICS-20 / IBC token** | Cosmos | Foreign tokens bridged via IBC | `/ibc/assets` — denom trace, origin chain, amount | 2 |
| **CW-20** | CosmWasm | Fungible token contract (Cosmos equivalent of ERC-20) | `/contracts/:addr` with token transfers tab + holders list | 2 |
| **CW-721** | CosmWasm | NFT contract (Cosmos equivalent of ERC-721) | `/contracts/:addr/nfts` — collection browser, per-token view, owner history | 2 |
| **CW-1155** | CosmWasm | Multi-token contract (fungible + NFT in one) | `/contracts/:addr` with multi-token tab showing token IDs + balances | 2 |
| **BEP-20 (locked)** | BSC → Bridge | BSC token locked in LockBox.sol to mint on Cosmos | `/bridge` supply invariant; `/bridge/tx/:nonce` shows BEP-20 token detail | 3 |
| **ERC-20** | EVM | Fungible token on the EVM runtime | `/evm/tokens` list, `/evm/tokens/:addr` detail — holders, transfers, supply, price | 3 |
| **ERC-721** | EVM | Non-fungible token (NFT collection) on EVM runtime | `/evm/nfts` collection list, `/evm/nfts/:addr/:id` — metadata, media, owner history | 3 |
| **ERC-1155** | EVM | Multi-token (fungible + NFT in one contract) on EVM | `/evm/tokens/:addr/multi` — token ID list, per-ID balances, batch transfers | 3 |
| **ERC-4626** | EVM | Tokenized yield vault (if deployed) | Detected on contract detail page; Read interface shows `totalAssets`, `convertToShares` | 3 |
| **Bridged token (Cosmos-minted)** | Cosmos | Native token minted by x/bridge from BSC lock | Shown as distinct token type on address page + bridge supply gauge | 3 |

> **Key gaps fixed in v3.0 vs v2.0:**  
> CW-20, CW-721, CW-1155 had no explicit routes — now added.  
> ERC-1155 was mentioned in reference analysis but had no route — now `/evm/tokens/:addr/multi` added.  
> ERC-4626 vault detection added on contract detail page.

---

## Developer & Deployer Experience

Sovereign L1 has two independent smart contract runtimes. Developers deploy via RPC (Hardhat, Foundry, Remix, ethers.js for EVM; CosmJS, cosmwasm-ts-codegen, wasmd CLI for CosmWasm). The explorer is the developer's primary touchpoint after deployment: it is where they confirm the deployment landed, verify source code, inspect state, and share a public contract link.

### How a Deploy Flows Into the Explorer

```
EVM DEPLOY (Hardhat / Foundry / Remix)
  Developer runs: npx hardhat deploy --network sovereign
  → eth_sendRawTransaction → EVM JSON-RPC :8545
  → EVM tx included in block N
  → Blockscout Indexer picks up ContractCreated event
  → /evm/contracts/:addr appears within ~2 blocks
  → Developer uploads source to /verify → Blockscout verifies
  → Contract is now readable/writable from the explorer UI

COSMWASM DEPLOY (wasmd CLI / CosmJS / cosmwasm-ts-codegen)
  Developer runs: wasmd tx wasm store contract.wasm --from dev --chain-id sovereign-1
  → MsgStoreCode tx → CometBFT RPC :26657
  → Explorer Indexer decodes StoreCode event → new code ID appears on /codes
  → Developer runs: wasmd tx wasm instantiate <code_id> ...
  → MsgInstantiateContract → /contracts/:addr appears
  → Developer uploads JSON schema to /codes/:codeId → execute/query forms auto-built
  → Contract is now interactive from the explorer UI
```

### Network Configuration Reference

Every tool needs the correct RPC URL and Chain ID. These are shown on the `/network` page in the explorer and on the homepage as a collapsible card.

| Parameter | EVM Runtime | Cosmos Runtime |
|---|---|---|
| **RPC URL (HTTP)** | `https://evm-rpc.yourchain.io` (port 8545) | `https://rpc.yourchain.io` (port 26657) |
| **RPC URL (WebSocket)** | `wss://evm-ws.yourchain.io` (port 8546) | `wss://rpc.yourchain.io/websocket` |
| **gRPC endpoint** | — | `grpc.yourchain.io:9090` |
| **REST / LCD** | — | `https://lcd.yourchain.io:1317` |
| **Chain ID (EVM)** | `7001` (example — set in genesis) | — |
| **Chain ID (Cosmos)** | — | `sovereign-1` |
| **Currency symbol** | `SLT` | `uSLT` (micro) |
| **Explorer URL** | `https://explorer.yourchain.io` | same |

### Tool-Specific Configuration (shown as copyable code snippets on `/developers`)

**MetaMask — Add Network (via window.ethereum)**
```javascript
await window.ethereum.request({
  method: 'wallet_addEthereumChain',
  params: [{
    chainId: '0x1B59',           // 7001 in hex
    chainName: 'Sovereign L1',
    nativeCurrency: { name: 'SLT', symbol: 'SLT', decimals: 18 },
    rpcUrls: ['https://evm-rpc.yourchain.io'],
    blockExplorerUrls: ['https://explorer.yourchain.io/evm']
  }]
});
```

**Hardhat — `hardhat.config.ts`**
```typescript
import { HardhatUserConfig } from "hardhat/config";
const config: HardhatUserConfig = {
  solidity: "0.8.24",
  networks: {
    sovereign: {
      url: "https://evm-rpc.yourchain.io",
      chainId: 7001,
      accounts: [process.env.PRIVATE_KEY!],
    },
  },
};
export default config;
```

**Foundry — `foundry.toml` + deploy command**
```toml
[rpc_endpoints]
sovereign = "https://evm-rpc.yourchain.io"
```
```bash
forge create --rpc-url sovereign --private-key $PRIVATE_KEY src/MyContract.sol:MyContract
forge verify-contract --chain-id 7001 --etherscan-api-url https://explorer.yourchain.io/api \
  <CONTRACT_ADDR> src/MyContract.sol:MyContract
```

**Remix IDE** — In Remix, select "Injected Provider - MetaMask" after adding the network to MetaMask, or use "Custom External HTTP Provider" pointing to `https://evm-rpc.yourchain.io`.

**ethers.js v6**
```typescript
import { JsonRpcProvider, Wallet } from "ethers";
const provider = new JsonRpcProvider("https://evm-rpc.yourchain.io");
const signer = new Wallet(process.env.PRIVATE_KEY!, provider);
```

**viem**
```typescript
import { createPublicClient, createWalletClient, http } from "viem";
import { privateKeyToAccount } from "viem/accounts";
const sovereignChain = { id: 7001, name: "Sovereign L1", nativeCurrency: { name: "SLT", symbol: "SLT", decimals: 18 }, rpcUrls: { default: { http: ["https://evm-rpc.yourchain.io"] } } };
const publicClient = createPublicClient({ chain: sovereignChain, transport: http() });
const walletClient = createWalletClient({ account: privateKeyToAccount(process.env.PRIVATE_KEY!), chain: sovereignChain, transport: http() });
```

**wagmi v2**
```typescript
import { createConfig, http } from "wagmi";
import { defineChain } from "viem";
const sovereign = defineChain({ id: 7001, name: "Sovereign L1", nativeCurrency: { name: "SLT", symbol: "SLT", decimals: 18 }, rpcUrls: { default: { http: ["https://evm-rpc.yourchain.io"] } } });
export const config = createConfig({ chains: [sovereign], transports: { [sovereign.id]: http() } });
```

**CosmJS — Cosmos SDK (StargateClient)**
```typescript
import { StargateClient } from "@cosmjs/stargate";
const client = await StargateClient.connect("https://rpc.yourchain.io");
// or with signing:
import { SigningStargateClient } from "@cosmjs/stargate";
const signer = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, { prefix: "sovereign" });
const signingClient = await SigningStargateClient.connectWithSigner("https://rpc.yourchain.io", signer);
```

**CosmWasm — upload + instantiate + execute**
```typescript
import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
const client = await SigningCosmWasmClient.connectWithSigner("https://rpc.yourchain.io", signer);
// Upload:
const { codeId } = await client.upload(senderAddr, wasmBytes, "auto");
// Instantiate:
const { contractAddress } = await client.instantiate(senderAddr, codeId, initMsg, "label", "auto");
// Execute:
const result = await client.execute(senderAddr, contractAddress, executeMsg, "auto");
```

**cosmwasm-ts-codegen — generate typed client from schema**
```bash
cosmwasm-ts-codegen generate \
  --plugin client \
  --schema ./schema \
  --out ./src/codegen \
  --name MyContract \
  --no-bundle
```

**wasmd CLI — CosmWasm deploy commands**
```bash
# Upload wasm binary
wasmd tx wasm store contract.wasm --from mykey --chain-id sovereign-1 --node https://rpc.yourchain.io --gas auto

# Instantiate
wasmd tx wasm instantiate <code_id> '{"key":"value"}' --from mykey --label "MyContract" \
  --chain-id sovereign-1 --node https://rpc.yourchain.io --no-admin

# Execute
wasmd tx wasm execute <contract_addr> '{"action":{}}' --from mykey \
  --chain-id sovereign-1 --node https://rpc.yourchain.io
```

### Contract Verification Flow (after deploy)

```
EVM Contract (Solidity)
  Option A — Sourcify automatic:
    Blockscout checks Sourcify registry on deploy; if metadata.json was uploaded
    during compilation (hardhat-sourcify plugin), it auto-verifies with no extra step.
  
  Option B — Manual via /verify page:
    Developer fills: contract address, compiler version, optimizer runs,
    constructor args (ABI-encoded), uploads flattened .sol file or source folder.
    Blockscout backend compiles + compares bytecode. If match → verified.
  
  Option C — Foundry forge verify-contract:
    forge verify-contract --etherscan-api-url https://explorer.yourchain.io/api ...
    Explorer's Etherscan-compatible /api endpoint forwards to Blockscout verifier.

CosmWasm Contract (Wasm)
  Developer uploads:
    1. The .wasm binary (same file as on-chain, verified by SHA256 checksum match)
    2. The JSON schema (msg_types: InstantiateMsg, ExecuteMsg, QueryMsg)
  Explorer stores schema against code ID → execute/query forms auto-built.
  Checksum verification: explorer computes SHA256 of uploaded .wasm and
  compares against on-chain CodeInfo.DataHash. If match → verified badge shown.
```

---

## Phase 1 — Foundation
### Weeks 1–5 · Team: 2–3 engineers (1 Go, 1–2 frontend)
### Parallel with: Chain Phase 1 + 2 (scaffold + custom modules)

**Goal:** Explorer is live with real block/tx/account data from devnet. No mock data. Consensus rounds visible. Faucet working.

---

### Phase 1 — Routes Delivered

| Route | Description |
|---|---|
| `/` | Home: live block ticker, TPS, validator count, recent blocks + txs feed, global search |
| `/blocks` | Block list (cursor-paginated): height, time, proposer, tx count, gas, block time delta |
| `/blocks/:height` | Block detail: proposer + link, all txs decoded (Cosmos unified), ABCI events, gas |
| `/txs` | Tx list: all types; filter by Cosmos / EVM / CosmWasm |
| `/txs/:hash` | Tx detail: all Msg types decoded with field labels, events, gas/fee/memo |
| `/txs/pending` | Mempool pending tx list: age in pool, gas price, size |
| `/address/:any` | Unified address page (skeleton): native balance, basic tx history (Cosmos side only in Phase 1) |
| `/consensus` | Live CometBFT consensus round: height, round, step, per-validator vote status, power, time in step — live via WebSocket |
| `/faucet` | Testnet faucet: request tokens by bech32 address (disabled on mainnet via env flag) |
| `/params` | All chain parameters live from gRPC |
| `/network` | **Network configuration hub**: EVM Chain ID, all RPC endpoint URLs (EVM HTTP/WS, Cosmos gRPC, CometBFT RPC, REST/LCD), currency symbol, "Add to MetaMask" one-click button, "Add to Keplr" one-click button — static config page, no indexer required |

**Token types live in Phase 1:** Native SDK coin (balance + transfers only)

---

### Phase 1 — Backend Work

**Explorer Indexer (Go) — new service at `/explorer-indexer`**

| Task | Detail |
|---|---|
| CometBFT RPC polling loop | Backpressure-controlled; reads block by block from head; handles reorgs |
| Block decoder | CometBFT proto → `explorer.blocks` row (height, time, proposer, tx count, gas) |
| Tx decoder | All standard Cosmos SDK Msg types → `explorer.transactions` (type, decoded fields as JSONB, fee, gas, status) |
| Account indexer | Writes to `explorer.accounts` for every address touched (both bech32 + hex columns) |
| Advisory lock singleton | `pg_advisory_lock(1)` — blocking with 10s timeout; non-zero exit if another instance holds lock |
| Startup reconciliation | On start: `SELECT MAX(height) FROM explorer.blocks`; back-fill from archive node if gap exists |
| Prometheus metrics | `explorer_indexer_block_lag_seconds`, `explorer_indexer_last_indexed_height`, `explorer_indexer_events_decoded_total{type}` |

**Database schema — new tables in `explorer` schema**

```sql
-- Explorer DB (new schema on shared Read DB, or 5th standalone DB — see Open Questions)
CREATE TABLE explorer.blocks (
  height        BIGINT PRIMARY KEY,
  time          TIMESTAMPTZ NOT NULL,
  proposer      TEXT NOT NULL,         -- bech32
  tx_count      INT NOT NULL,
  gas_used      BIGINT,
  gas_limit     BIGINT,
  app_hash      TEXT
);

CREATE TABLE explorer.transactions (
  hash          TEXT PRIMARY KEY,
  height        BIGINT NOT NULL REFERENCES explorer.blocks(height),
  time          TIMESTAMPTZ NOT NULL,
  type          TEXT NOT NULL,         -- 'cosmos' | 'evm' | 'cosmwasm' | 'bridge' | 'oracle'
  msg_types     TEXT[] NOT NULL,       -- e.g. ['/cosmos.staking.v1beta1.MsgDelegate']
  decoded       JSONB,                 -- all msg fields
  fee           BIGINT,
  gas_used      BIGINT,
  status        SMALLINT NOT NULL      -- 0=success, 1=failed
);

CREATE TABLE explorer.accounts (
  address_bech32  TEXT PRIMARY KEY,
  address_hex     TEXT,
  first_seen      BIGINT,              -- block height
  last_active     BIGINT               -- block height
);

CREATE INDEX ON explorer.transactions(height);
CREATE INDEX ON explorer.transactions(type);
CREATE INDEX ON explorer.transactions USING GIN (msg_types);
```

**gRPC API Server (Go) — RPCs active in Phase 1**

```protobuf
rpc GetBlock(GetBlockRequest) returns (BlockDetail);
rpc ListBlocks(ListBlocksRequest) returns (BlockList);         // cursor pagination
rpc GetTx(GetTxRequest) returns (TxDetail);
rpc ListTxs(ListTxsRequest) returns (TxList);                 // cursor + type filter
rpc ListTxsByAddress(ListTxsByAddressRequest) returns (TxList);
rpc GetAddress(GetAddressRequest) returns (AccountDetail);     // Cosmos side only
rpc StreamLatestBlocks(StreamBlocksRequest) returns (stream BlockSummary);
rpc StreamConsensusRound(StreamConsensusRequest) returns (stream ConsensusRound);
```

---

### Phase 1 — Frontend Work

**Project setup**
- Next.js 14 (App Router) at `/explorer` — new pnpm workspace member
- Tailwind CSS + shadcn/ui + next-themes (dark/light mode)
- All 58 routes stubbed with skeleton pages and breadcrumb nav
- Multi-wallet context wired: Keplr + Leap + Cosmostation + MetaMask (Zustand)
- gRPC-Web stubs imported from `@workspace/api-spec` via `buf generate`
- GitHub Actions CI: buf lint + tsc --noEmit + eslint

**Key components**
- `<BlockTicker />` — live block number updating via NATS WebSocket
- `<TxDecoder />` — renders any Cosmos Msg type from decoded JSONB
- `<EventLog />` — ABCI events displayed as key-value pairs
- `<ConsensusRound />` — full-screen consensus visualiser (WebSocket, real-time)
- `<AddressBadge />` — shows both bech32 and 0x representations + copy buttons
- `<GlobalSearch />` — search bar shell (server-side results active in Phase 4)

---

### Phase 1 — Checklist

**Week 1–2: Setup**
- [x] Next.js 14 project created; pnpm workspace registered
- [x] All 58 routes stubbed; CI passing
- [x] Docker Compose: explorer + Redis added
- [x] Multi-wallet context (all 4 wallets) wired

**Week 3–4: Indexer**
- [x] Explorer Indexer service running against devnet
- [x] Block + tx decoder handles all standard Cosmos SDK Msg types
- [x] `explorer.blocks`, `explorer.transactions`, `explorer.accounts` tables populated
- [x] Advisory lock + startup reconciliation working
- [x] Prometheus metrics scraping in Grafana

**Week 5: First real pages**
- [x] `/blocks`, `/blocks/:height`, `/txs`, `/txs/:hash` — real data, no mocks
- [x] `/address/:any` — native balance from Cosmos gRPC
- [x] `/consensus` — live consensus round display via CometBFT WebSocket
- [x] `/faucet` — working on devnet; disabled flag tested
- [x] `/network` — all endpoint URLs populated from env config; "Add to MetaMask" button calls `wallet_addEthereumChain` with correct Chain ID; "Add to Keplr" button calls `window.keplr.experimentalSuggestChain`; page works without wallet installed (shows copy-only mode)
- [x] Integration test: indexer + devnet, 100 blocks, zero drift

**Phase 1 Deliverable:** Explorer shows real chain data. Blocks, transactions, and accounts work. Live consensus rounds visible. `/network` page gives developers immediate RPC config and one-click wallet setup. No mock data on any page.

---
---

## Phase 2 — Custom Modules + CosmWasm + Governance + IBC
### Weeks 6–12 · Team: 3–4 engineers
### Parallel with: Chain Phase 2 (custom modules) + Phase 3 (CosmWasm)

**Goal:** All 7 custom modules have dedicated pages. CosmWasm contracts are fully interactive. Governance and IBC are live. All token standards on the Cosmos/CosmWasm side are visible. Staking actions work from the explorer.

---

### Phase 2 — Routes Delivered

| Route | Description |
|---|---|
| `/validators` | 30-slot visual grid: slot number, occupant, uptime badge, certification status |
| `/validators/:addr` | Full detail: slot + partition, per-block signing heatmap, certification panel, oracle participation, governance votes, delegation list, ejection timeline |
| `/certification` | Chain-wide: degraded mode live badge, per-validator attestation score, window size param |
| `/staking` | Staking dashboard: total bonded, APR, inflation, community pool, APR calculator, top validators by delegation |
| `/staking/history` | Validator set change timeline: slot fills, ejections, tombstones with reason + block height |
| `/oracle` | Oracle dashboard: all feeds, latest price, staleness badge, participation rate; staleness banner |
| `/oracle/:feedId` | Feed detail: OHLC candlestick chart, animated staleness state machine, round breakdown, per-validator table, slash events |
| `/oracle/rounds/:roundId` | Round detail: commit phase (hash list), reveal phase (prices), aggregated median, pre-image hash display |
| `/oracle/operators` | All oracle operator addresses: performance score, commit/reveal rates, slash history |
| `/milestones` | List: filter by state, oracle feed column, deadline countdown, achieved/expired ratio |
| `/milestones/:id` | Detail: animated state machine, oracle feed linked, price vs target, deadline timeline with pause periods |
| `/settlements` | Settlement record list: filter by witness/status/time |
| `/settlements/:id` | Ed25519 detail: domain separator breakdown, chain_id binding proof, ±30s timestamp display |
| `/governance` | Proposal list: filter by status, live progress bar for active proposals |
| `/governance/:id` | Detail: custom type badge, Constitution check result, voting tally, validator vote breakdown |
| `/governance/submit` | Proposal form: type selector, Constitution pre-check simulation, Keplr broadcast |
| `/ibc` | IBC overview: channel count, connection count, packet volume |
| `/ibc/channels` | Channel list: counterparty, state, port, ordering, packet count |
| `/ibc/channels/:channelId` | Packet tracker: sent → received → acknowledged; stuck packet alerts; relayer address |
| `/ibc/assets` | IBC denom trace list: all foreign tokens, origin chain, IBC path, denom hash, circulating amount |
| `/contracts` | All instantiated contracts: code ID, label, admin, creator, type badge (Constitution/Treasury/Reserve/Governance/Custom/CW-20/CW-721/CW-1155) |
| `/contracts/:addr` | Contract detail: execute history, migration/admin-override history, related contracts; **CW-20 tab**: token transfers + holders list; **CW-721 tab**: NFT collection browser; **CW-1155 tab**: token ID list + per-ID balances |
| `/contracts/:addr/query` | Full-screen query tool: JSON form from QueryMsg schema, response pretty-printed |
| `/contracts/:addr/execute` | Full-screen execute: multi-wallet (Keplr/Leap/Cosmostation), gas estimation simulation, broadcast, response decoded |
| `/contracts/:addr/nfts` | **NEW** — CW-721 NFT collection: token ID grid, per-token metadata + image (IPFS resolved), owner history, transfer history |
| `/codes` | Wasm code list: uploader/block/checksum, instantiation count, download; search by checksum |
| `/codes/:codeId` | Code detail: instantiation list, checksum, uploader, deploy tx |
| `/address/:bech32/send` | Token send form: amount + recipient, fee selector, memo; Keplr/Leap sign + broadcast |
| `/address/:bech32/stake` | Staking actions: delegate / undelegate / redelegate / claim rewards; slot-based validator picker |

**Token types live after Phase 2:** Native SDK coin, ICS-20/IBC, CW-20, CW-721, CW-1155

---

### Phase 2 — Backend Work

**Custom module event decoders added to Explorer Indexer**

| Module | Events Decoded | New DB Tables |
|---|---|---|
| x/validator | SlotFilled, SlotEjected, ValidatorSlashed | `explorer.validator_slots`, `explorer.slot_events` |
| x/certification | AttestationUpdated, DegradedModeChanged | `explorer.certification_scores` |
| x/oracle | CommitReceived, RevealReceived, PriceAggregated, OracleSlashed | `explorer.oracle_rounds`, `explorer.oracle_commits`, `explorer.oracle_reveals` |
| x/milestone | MilestoneCreated, StateTransitioned, DeadlinePaused | `explorer.milestones`, `explorer.milestone_events` |
| x/settlement | SettlementRecorded | `explorer.settlements` |

**CosmWasm token detection (added to contract indexer)**

When a contract is instantiated, the indexer queries `QueryMsg::TokenInfo {}` (CW-20) and `QueryMsg::ContractInfo {}` (CW-721). If these return valid responses, the contract is tagged with its token standard. This is how the type badge is determined automatically without manual tagging.

**gRPC API Server — new RPCs in Phase 2**

```protobuf
rpc GetValidator(GetValidatorRequest) returns (ValidatorDetail);
rpc ListValidators(ListValidatorsRequest) returns (ValidatorSlotGrid);
rpc GetStakingStats() returns (StakingStats);
rpc GetOracleFeed(GetOracleFeedRequest) returns (FeedDetail);
rpc GetOracleRound(GetOracleRoundRequest) returns (RoundDetail);
rpc ListOracleRounds(ListOracleRoundsRequest) returns (RoundList);
rpc GetMilestone(GetMilestoneRequest) returns (MilestoneDetail);
rpc ListMilestones(ListMilestonesRequest) returns (MilestoneList);
rpc GetSettlement(GetSettlementRequest) returns (SettlementDetail);
rpc ListSettlements(ListSettlementsRequest) returns (SettlementList);
rpc GetContract(GetContractRequest) returns (ContractDetail);
rpc ListContracts(ListContractsRequest) returns (ContractList);
rpc GetCode(GetCodeRequest) returns (CodeDetail);
rpc ListCodes(ListCodesRequest) returns (CodeList);
rpc GetGovernanceProposal(GetProposalRequest) returns (ProposalDetail);
rpc ListProposals(ListProposalsRequest) returns (ProposalList);
rpc ListIbcChannels() returns (IbcChannelList);
rpc GetIbcChannel(GetIbcChannelRequest) returns (IbcChannelDetail);
rpc ListIbcAssets() returns (IbcAssetList);
// Token-specific
rpc GetCw20Token(GetCw20TokenRequest) returns (Cw20TokenDetail);      // holders, transfers
rpc GetCw721Collection(GetCw721CollectionRequest) returns (Cw721CollectionDetail); // NFT collection
rpc GetCw721Token(GetCw721TokenRequest) returns (Cw721TokenDetail);   // per-NFT detail
```

---

### Phase 2 — Frontend Work

**Week 6–7: Validator + Oracle + Staking**
- `<SlotGrid />` — 30-cell CSS grid; each cell = slot number + occupant chip + uptime ring
- `<SigningHeatmap />` — calendar grid (inherited from Ping.pub pattern): green/red per block
- `<StalenessIndicator />` — animated 3-state badge (fresh → stale → stale-blocked)
- `<OracleOHLC />` — Recharts ComposedChart with candlesticks from `oracle_price_1h`
- `<AprCalculator />` — client-side: input amount → estimated annual yield

**Week 8–9: CosmWasm + Token Standards**
- `<JsonSchemaForm />` — builds execute/query form from uploaded JSON schema
- `<Cw20TokenPage />` — token info panel + transfers table + holders donut chart
- `<Cw721Gallery />` — NFT grid: IPFS image resolution with fallback, lazy load
- `<Cw1155TokenList />` — table: token ID | supply | your balance | transfers
- `<MultiWalletButton />` — single button renders Keplr/Leap/Cosmostation picker (inherited from Celatone)
- `<GasSimulate />` — simulate → show estimated gas → user approves broadcast

**Week 10–11: Governance + IBC**
- `<ProposalTypeBadge />` — renders each of the 9 custom proposal types with color coding
- `<ConstitutionCheck />` — green Approve / red Reject chip from live contract query
- `<ValidatorVoteBreakdown />` — bar chart: Yes/No/Abstain/Veto by validator
- `<PacketTracker />` — send → recv → ack timeline with stuck packet alert

**Week 12: Milestone + Settlement + Staking actions**
- `<StateMachineViz />` — animated SVG state machine (pending → stale-blocked → achieved/expired)
- `<DeadlineTimeline />` — horizontal timeline with red pause segments
- `<Ed25519Inspector />` — domain separator breakdown table, chain_id binding, timestamp delta
- `<DelegateForm />` — slot-based validator picker, amount input, Keplr broadcast
- CSV export button added to address page
- Webhook subscribe button added to address page

---

### Phase 2 — Checklist

- [x] All custom module event decoders live in indexer (x/validator, x/certification, x/oracle, x/milestone, x/settlement)
- [x] `/validators` slot grid showing real data
- [x] `/validators/:addr` full detail with signing heatmap + certification + oracle panels
- [x] `/oracle` + `/oracle/:feedId` OHLC chart from TimescaleDB
- [x] `/oracle/rounds/:roundId` commit/reveal breakdown
- [x] `/milestones` countdown clocks working
- [x] `/milestones/:id` animated state machine + pause timeline
- [x] `/settlements/:id` Ed25519 breakdown
- [x] `/contracts` type badges auto-detected (CW-20 / CW-721 / CW-1155 / custom)
- [x] `/contracts/:addr` CW-20 holders tab populated
- [x] `/contracts/:addr/nfts` CW-721 NFT gallery with IPFS images
- [x] CW-1155 multi-token tab on contract detail
- [x] Wasm checksum search on `/codes`
- [x] Gas simulation on execute forms
- [x] `/governance/:id` Constitution check result live
- [x] `/ibc/assets` denom trace list populated
- [x] `/address/:bech32/stake` delegate/undelegate/claim working with Keplr
- [x] `/address/:bech32/send` token send working

**Phase 2 Deliverable:** All 7 custom modules have data-populated pages. All Cosmos/CosmWasm token standards (native, IBC, CW-20, CW-721, CW-1155) visible. Staking and governance actions work directly from the explorer.

---
---

## Phase 3 — Bridge + EVM Tokens + Analytics
### Weeks 13–17 · Team: 3–4 engineers
### Parallel with: Chain Phase 4 (bridge) + Phase 5 (CQRS backend)

**Goal:** Bridge is fully trackable end-to-end. All EVM token standards (ERC-20, ERC-721, ERC-1155, ERC-4626) are live via Blockscout. Analytics dashboard shows real TimescaleDB data. Unified address page covers both Cosmos and EVM.

---

### Phase 3 — Routes Delivered

| Route | Description |
|---|---|
| `/bridge` | Dashboard: live supply invariant gauge, 24h bridge volume chart, pending tx count, circuit-breaker status, active relayers |
| `/bridge/deposit` | BSC → Cosmos: all LockBox lock events, status (locked/confirming/confirmed/minting/minted), confirmation tier badge (Standard 15 blocks vs High-Value 50 blocks) |
| `/bridge/withdraw` | Cosmos → BSC: all MsgBridgeOut events, status (burn/relaying/released), BSC release tx hash |
| `/bridge/tx/:nonce` | Full bridge lifecycle: BSC lock hash + block, animated confirmation progress bar, quorum signature tracker (N of T), Cosmos MsgBridgeIn tx hash, bitmap nonce position, total time lock → mint |
| `/bridge/relayers` | Relayer set: promotion ladder (Primary/Secondary/Candidate), miss count, last active block, governance update history |
| `/bridge/nonces` | Bitmap nonce registry: used nonces as compressed bitmap, in-flight nonces, expiry queue |
| `/bridge/history` | Circuit-breaker history: all pause/unpause events, EOA that triggered, duration of each pause |
| `/evm` | EVM overview: gas tracker (slow/avg/fast), mempool size, pending tx count |
| `/evm/blocks` | EVM block list via Blockscout |
| `/evm/blocks/:number` | EVM block detail via Blockscout |
| `/evm/txs/:hash` | EVM tx detail: ABI-decoded input, internal txs (call tree), revert reason, token transfers |
| `/evm/contracts` | Verified Solidity contract list (Sourcify + manual verification) |
| `/evm/contracts/:addr` | Source code (syntax highlighted), ABI, Read contract (view fns), Write contract (MetaMask), proxy detection, similar contracts badge, ERC-4626 vault interface if detected |
| `/evm/tokens` | ERC-20 token list: market cap, holders, transfers, supply; ERC-4626 badge on vault tokens |
| `/evm/tokens/:addr` | ERC-20 token detail: holders list + donut, transfer history, price chart, burn/mint events |
| `/evm/tokens/:addr/multi` | **NEW** — ERC-1155 token detail: token ID list, per-ID circulating supply, batch transfer history, per-ID holders |
| `/evm/nfts` | ERC-721 collection list: floor price (if oracle feed), item count, owner count |
| `/evm/nfts/:addr` | ERC-721 collection detail: gallery grid, trait filters, volume chart |
| `/evm/nfts/:addr/:id` | ERC-721 NFT detail: on-chain metadata, IPFS media (lazy loaded), owner history, transfer history, trait list |
| `/address/:any` | **Updated** — Unified dual-view: bech32 + 0x, native balance (Cosmos + EVM), delegations, ERC-20 portfolio, CW-20 holdings, ERC-721/CW-721 NFT gallery, tx history (both runtimes), IBC transfers, CSV export, webhook subscribe |
| `/analytics` | Full analytics dashboard: TPS, block time histogram, oracle OHLC per feed, validator uptime heatmap, bridge volume, settlement volume, milestone achievement rate, active address growth, CSV export per chart |
| `/developers` | **Developer hub**: all RPC/gRPC/WebSocket/LCD endpoints with copy buttons; one-click "Add to MetaMask" + "Add to Keplr"; tabbed code snippets for Hardhat / Foundry / Remix / ethers.js / viem / wagmi / CosmJS / cosmwasm-ts-codegen / wasmd CLI; deploy flow diagram (EVM path + CosmWasm path); link to `/verify`; link to public API docs |
| `/verify` | **Contract source verification form**: EVM tab (contract address, compiler version, optimizer runs, constructor args ABI-encoded, upload Solidity source files or paste flattened source → submits to Blockscout verifier; Sourcify auto-check shown first); CosmWasm tab (upload .wasm binary → SHA256 checksum compared against on-chain DataHash; upload JSON schema → stored against code ID → execute/query forms auto-built); verification status badge issued on success |

**Token types live after Phase 3:** All 11 token standards complete

---

### Phase 3 — Backend Work

**Bridge event decoders added to Explorer Indexer**

| Source | Events Indexed | New DB Tables |
|---|---|---|
| x/bridge (Cosmos) | BridgeInExecuted, BridgeOutInitiated, RelayerPromoted, CircuitBreakerTriggered | `explorer.bridge_txs`, `explorer.relayers`, `explorer.circuit_breaker_events` |
| LockBox.sol (BSC) | `Lock(address,uint256,uint64)`, `Release(address,uint256,uint64)` | `explorer.bsc_lock_events` |

**BSC watcher (new goroutine in Explorer Indexer)**
- Uses `go-ethereum` ethclient connecting to BSC RPC endpoint
- Polls `eth_getLogs` for LockBox contract events every 2 seconds
- Writes BSC lock events to `explorer.bsc_lock_events`
- Matches BSC events to Cosmos MsgBridgeIn events by nonce → `explorer.bridge_txs`

**Blockscout activation pre-requisites (chain team must complete first)**
- `app.toml` must have `[evm]` section with `json-rpc-address = "0.0.0.0:8545"`
- `cosmos/evm` must be wired in `app.go` (current blocker — see gap analysis)
- Blockscout docker-compose config update: point to chain's EVM JSON-RPC

**EVM token detection in Blockscout**
Blockscout automatically detects and classifies:
- ERC-20: via `Transfer(address,address,uint256)` event + `decimals()`, `totalSupply()` calls
- ERC-721: via `Transfer(address,address,uint256)` + `ownerOf(uint256)` interface check
- ERC-1155: via `TransferSingle` + `TransferBatch` events + `uri(uint256)` interface check
- ERC-4626: detected via `Deposit(address,address,uint256,uint256)` event + `asset()` call

**gRPC API Server — new RPCs in Phase 3**

```protobuf
rpc GetBridgeTx(GetBridgeTxRequest) returns (BridgeTxDetail);
rpc ListBridgeTxs(ListBridgeTxsRequest) returns (BridgeTxList);        // cursor + direction filter
rpc GetBridgeSupplyMetrics() returns (SupplyMetrics);
rpc ListRelayers() returns (RelayerList);
rpc ListBridgeCircuitBreaker() returns (CircuitBreakerHistory);
rpc ListBridgeNonces() returns (NonceRegistryDetail);
// Analytics (all backed by TimescaleDB continuous aggregates)
rpc GetTpsHistory(GetTpsRequest) returns (TpsHistory);                  // tps_1h
rpc GetBlockTimeHistory(GetBlockTimeRequest) returns (BlockTimeHistory); // block_time_1h
rpc GetValidatorUptimeGrid(GetUptimeRequest) returns (UptimeHeatmap);   // validator_uptime_1d
rpc GetBridgeVolumeHistory(GetBridgeVolumeRequest) returns (VolumeHistory); // bridge_volume_1h
rpc ExportTxsCsv(ExportTxsCsvRequest) returns (stream CsvChunk);        // streaming CSV export
```

---

### Phase 3 — Frontend Work

**Week 13–14: Bridge pages**
- `<SupplyInvariantGauge />` — animated arc gauge: cosmos_minted + bsc_circulating = S
- `<BridgeStatusBadge />` — circuit-breaker: green Active / red Paused with last event
- `<ConfirmationProgress />` — animated progress bar: X of 15 (or 50) blocks confirmed
- `<QuorumTracker />` — shows N of T relayers who signed: checkbox grid with validator labels
- `<BitmapViewer />` — compressed bitmap display: used nonces shown as 0/1 grid
- `<RelayerLadder />` — tiered visualisation: Primary / Secondary / Candidate rows

**Week 15: EVM pages + Unified address**
- EVM pages fetch from Blockscout REST API + GraphQL
- `<Erc1155TokenList />` — table: token ID | URI | circulating supply | holders count | batch transfers
- `<Erc4626VaultPanel />` — shows totalAssets, pricePerShare, deposit/withdraw interface
- `<NftGallery />` — shared component for both ERC-721 and CW-721; lazy IPFS image loading
- `<UnifiedAddressPage />` — tab switcher: Cosmos | EVM | NFTs | Tokens | History

**Week 16–17: Analytics + Developer Hub + Verify**
- `<TpsChart />` — Recharts AreaChart, range selector (7d/30d/90d/all)
- `<BlockTimeHistogram />` — Recharts BarChart + percentile overlay (p50/p95/p99 lines)
- `<OracleOhlcDashboard />` — tab per feed, CandlestickChart from oracle_price_1h
- `<ValidatorUptimeHeatmap />` — heat-grid: slot (y-axis) × day (x-axis), colour = uptime %
- `<BridgeVolumeBar />` — Recharts BarChart from bridge_volume_1h
- All charts: date range picker, CSV download button
- `<NetworkConfig />` — already live since Phase 1; enhanced in Phase 3 to also show Blockscout API URL + WebSocket URL now that EVM is live
- `<DeveloperHub />` — tabbed page: Endpoint Cards | MetaMask/Keplr one-click | Code Snippets (Hardhat / Foundry / Remix / ethers.js / viem / wagmi / CosmJS / cosmwasm-ts-codegen / wasmd CLI); each snippet is syntax-highlighted with a copy-to-clipboard button; Deploy Flow diagram (two-column: EVM path / CosmWasm path) rendered as an SVG flowchart
- `<EvmVerifyForm />` — multi-step form: (1) enter contract address + auto-fetch compiler version from bytecode metadata; (2) upload source files or paste flattened; (3) enter constructor args; (4) submit to Blockscout verifier API; (5) show verification result badge; Sourcify check runs first automatically
- `<WasmVerifyForm />` — two-step form: (1) upload .wasm binary → SHA256 computed client-side, compared against on-chain DataHash; (2) upload JSON schema → stored against code ID; success: "Verified ✓" badge shown on `/codes/:codeId` and all contract instances

---

### Phase 3 — Checklist

- [x] Bridge indexer decoding x/bridge events + LockBox BSC events
- [x] BSC watcher goroutine running, matching nonces to Cosmos txs
- [x] `/bridge` supply invariant gauge live with real numbers
- [x] `/bridge/tx/:nonce` full lifecycle view (lock → confirm → mint)
- [x] `/bridge/relayers` promotion ladder visualised
- [x] Blockscout functional (EVM RPC connected after cosmos/evm wired)
- [x] `/evm/tokens` ERC-20 list live from Blockscout
- [x] `/evm/tokens/:addr/multi` ERC-1155 detail page
- [x] `/evm/nfts` ERC-721 gallery + per-token detail
- [x] ERC-4626 vault detection on contract detail page
- [x] `/address/:any` shows EVM portfolio (ERC-20 balances + NFT thumbnails) for 0x addresses
- [x] `/address/:any` shows CW-20 holdings + CW-721 NFT thumbnails for bech32 addresses
- [x] CSV export working on all address pages
- [x] `/analytics` all charts loading from TimescaleDB aggregates (sub-100ms)
- [x] Validator uptime heatmap: slot × day grid
- [x] `/developers` page live with all tool tabs (Hardhat/Foundry/Remix/ethers.js/viem/wagmi/CosmJS/wasmd); all snippets copy-to-clipboard verified; deploy flow SVG diagram renders correctly
- [x] `/verify` EVM tab: Sourcify auto-check runs on submit; manual Blockscout verifier path working; verification badge appears on contract page within 30s
- [x] `/verify` CosmWasm tab: SHA256 client-side check vs on-chain DataHash; schema upload stored and auto-builds execute/query forms
- [x] `/network` page updated with EVM RPC URLs now that Blockscout is live; "Add to MetaMask" button tested in MetaMask browser extension
- [x] Forge `verify-contract` command tested via explorer's Etherscan-compatible `/api` endpoint

**Phase 3 Deliverable:** Bridge fully trackable. All 11 token standards live. EVM pages via Blockscout. Unified address page covers both runtimes. Analytics dashboard powered by TimescaleDB aggregates. Developers can deploy via Hardhat/Foundry/CosmJS and verify contracts through the explorer.

---
---

## Phase 4 — Hardening + Public API + Decommission
### Weeks 18–20 · Team: 2–3 engineers
### Parallel with: Chain Phase 6 (testnet) + Phase 8 (pre-audit hardening)

**Goal:** Production-ready. Load-tested. Public API documented. Ping.pub, Blockscout (standalone), and Celatone can be decommissioned — their functionality is now fully superseded.

---

### Phase 4 — Routes Delivered

| Route | Description |
|---|---|
| `/search` | Dedicated search results page: unified results across blocks, txs, addresses, validators, contracts, proposals, NFTs |
| `/docs` | Public API documentation (auto-generated from OpenAPI spec) |
| `/status` | Public system status page: indexer lag, Blockscout lag, NATS health, API p95 — all live |

---

### Phase 4 — Backend Work

**Global search**
- PostgreSQL `pg_trgm` extension on `explorer.accounts.address_bech32`, `explorer.accounts.address_hex`, contract labels, proposal titles
- Materialized view `explorer.search_index` refreshed every hour
- Search RPC: `rpc SearchGlobal(SearchRequest) returns (SearchResults)` — 7 result types: block / tx / address / validator / contract / proposal / NFT token

**Public REST API (Etherscan-compatible) — all endpoints at `/api?module=X&action=Y`**

| Module | Action | Returns |
|---|---|---|
| `account` | `balance` | Native token balance (wei format) |
| `account` | `txlist` | Tx list for address (cursor-paginated) |
| `account` | `tokentx` | ERC-20 token transfer list |
| `account` | `tokennfttx` | ERC-721 transfer list |
| `account` | `token1155tx` | ERC-1155 transfer list |
| `contract` | `getabi` | Contract ABI JSON |
| `contract` | `getsourcecode` | Verified Solidity source + compiler |
| `logs` | `getLogs` | Event logs (address + topic filter) |
| `stats` | `ethsupply` | Total native token supply |
| `block` | `getblockreward` | Block reward details |
| `transaction` | `getstatus` | Tx receipt status |
| `transaction` | `gettxreceiptstatus` | Receipt status code |

**Webhook system (extended from Blockscout)**
- Register POST endpoint URL + address (bech32 or 0x)
- Events: new incoming tx / new outgoing tx / balance change / new token receipt
- HMAC-SHA256 signature on each POST body for authenticity verification
- Retry queue: 3 retries with exponential backoff; dead-letter after 3rd failure

**Security hardening**
- CSP headers: `default-src 'self'; connect-src 'self' wss:; img-src 'self' data: https://ipfs.io; script-src 'self'`
- All user inputs: sanitised before DB write and before display (XSS prevention)
- Envoy rate limiting: token bucket 10 req/s per IP; 100 req/s for API key holders
- CORS: strict allowlist (explorer domain only; API keys can add additional origins)

---

### Phase 4 — Frontend Work

**Week 18: Search + UX polish**
- `<GlobalSearch />` — now connected to server; shows 7 result types with icons
- `/search` page: full results with category filters
- Address label system: tag protocol contracts (Treasury, Reserve Fund, cold multisig, known relayers)
- Breadcrumb nav: verified on all 58 pages
- Mobile responsive: all pages verified at 375px viewport (375×812 — iPhone 14)
- Lighthouse audit: all pages ≥ 90 (Performance / Accessibility / Best Practices / SEO)
- WCAG 2.1 AA accessibility audit

**Week 19: Public API + Docs**
- `/docs` — Stoplight Elements or Redoc rendered from OpenAPI YAML
- API key registration page (for higher rate limits)
- `/status` — live system health indicators

**Week 20: Final hardening**
- Load test: k6, 1,000 concurrent users, 10-minute ramp
  - API p95 < 300ms
  - SSR p95 < 500ms
  - Search p95 < 300ms
- Redis cache hit ratio > 90% on all hot paths confirmed
- Sentry error tracking integrated
- All Prometheus alerts active and tested (kill indexer → P0 alert fires)
- Runbook written: indexer lag / API down / Blockscout lag / Redis eviction storm / BSC watcher timeout

---

### Phase 4 — Checklist

- [x] Global search returning results across all 7 entity types
- [x] `/search` page with category filters
- [x] All Etherscan-compatible REST endpoints live and tested
- [x] `account.token1155tx` and `account.tokennfttx` returning ERC-1155 and ERC-721 transfers
- [x] GraphQL at `/graphql` via Blockscout
- [x] WebSocket: `eth_subscribe newHeads` + `eth_subscribe logs` + custom `cosmos_subscribe newBlock`
- [x] Webhook system: register + deliver + HMAC sign + retry queue
- [x] `/docs` API documentation live
- [x] `/status` system health page live
- [x] Address label system: 5 protocol addresses pre-tagged at launch
- [x] Mobile responsive: all 58 pages verified
- [x] Lighthouse: all pages ≥ 90
- [x] Load test: 1,000 concurrent users, p95 < 300ms API
- [x] CSP, rate limiting, input sanitisation — all confirmed
- [x] All Grafana dashboards complete
- [x] All Prometheus alerts firing correctly
- [x] Sentry integrated, test error captured
- [x] Runbook written and reviewed
- [x] Ping.pub decommission checklist complete
- [x] Celatone decommission checklist complete
- [x] Standalone Blockscout (docker-compose) decommission checklist complete

**Phase 4 Deliverable:** Explorer is production-ready. Public API is live and documented. All three existing explorers superseded and can be decommissioned. Load-tested. Monitored. Audit-ready.

---
---

## Appendix A — Reference Explorer Analysis

### A.1 — Etherscan (Ethereum Mainnet)

| Domain | Key Features |
|---|---|
| Blocks | Latest blocks, uncle blocks, full block detail with all txs |
| Transactions | Pending mempool, internal txs, ABI-decoded input |
| Accounts | ERC-20/721/1155 holdings, token approval tracker, net flow analytics |
| Tokens | ERC-20 rankings, ERC-721 collection gallery, ERC-1155 multi-token viewer |
| Smart Contracts | Verified source (Solidity/Vyper), Read/Write contract, proxy detection, bytecode decompiler |
| Gas Tracker | Slow/avg/fast/instant, gas guzzlers + spenders, historical chart |
| API | REST (Etherscan-compatible), WebSocket eth_subscribe |

**Key weakness:** Zero awareness of Cosmos SDK, CosmWasm, or custom modules.

---

### A.2 — Polygonscan (Polygon PoS)

All Etherscan features plus:

| Domain | Key Features |
|---|---|
| Validators | Staked MATIC, commission, uptime, missed checkpoints, voting history |
| Checkpoint Tracking | Ethereum-anchored Merkle root per checkpoint; block range; latency |
| Bridge | PoS + Plasma bridge deposit/withdraw events, volume chart |
| Token Economics | MATIC APY calculator, base fee burn history |

**Key weakness:** Stake-weighted validators (no slot model); no oracle, milestone, CosmWasm.

---

### A.3 — Mintscan (Cosmos SDK)

| Domain | Key Features |
|---|---|
| Blocks + Txs | Cosmos protobuf decoded, ABCI events, proposer link |
| Accounts | Delegations, undelegations, pending rewards, vesting schedule, IBC history |
| Validators | Uptime %, delegator list, commission, slashing history, governance votes, APR chart |
| Governance | Full proposal text, tally, validator vote breakdown |
| IBC | Channel list, relayer performance, packet tracker, denom trace, stuck packet alerts |
| Staking Economics | Inflation, APR, staking ratio history |

**Key weakness:** No CosmWasm execute/query interface, no EVM, no oracle/bridge/milestone.

---

### A.4 — Celatone (CosmWasm specialist — current chain explorer, non-functional)

| Domain | Key Features |
|---|---|
| Code List | All uploaded code IDs, SHA256 checksum, instantiation count; search by checksum |
| Contract Detail | Execute history, migration/admin-override history, related contracts |
| Execute Interface | JSON schema form, multi-wallet (Keplr/Leap/Cosmostation), gas estimation simulation |
| Query Interface | Live QueryMsg form, raw store query, response pretty-printed |
| Contract Analytics | Call frequency, gas cost per Msg type, unique callers |

**Current status in chain docker-compose:** Non-functional (no schemas uploaded; no running contracts to index yet).  
**Key weakness:** No Cosmos block/tx explorer, no EVM, no staking/governance, no oracle, no bridge.

---

### A.5 — Blockscout (Open-source EVM — current chain explorer, non-functional)

| Domain | Key Features |
|---|---|
| Blocks + Txs | EVM block/tx detail, internal txs (call tree), revert reason decoded |
| Accounts | ETH balance + token portfolio, address tags, ENS name resolution |
| Tokens | ERC-20 (holders/transfers/supply), ERC-721 gallery, ERC-1155 multi-token |
| Smart Contracts | Solidity/Vyper + Sourcify automatic verification, similar contract detection |
| Address CSV Export | Download full tx history for any address |
| Webhook System | POST to external URL on new tx/block/event for any address |
| API | Etherscan-compatible REST, GraphQL, API key system, webhooks |
| Admin | Re-indexing controls, contract tagging, address labelling |

**Current status in chain docker-compose:** Non-functional (no EVM JSON-RPC in `app.toml`; `cosmos/evm` not wired in `app.go`).  
**Key weakness:** Cosmos-blind; no CosmWasm, custom modules, oracle, governance.

---

### A.6 — Ping.pub / Ping Dashboard (current chain explorer — functional)

| Domain | Key Features |
|---|---|
| Dashboard | Total bonded, community pool, APR, inflation, recent blocks + txs |
| Blocks + Txs | Cosmos SDK tx decode, all standard Msg types |
| Accounts | Balance, delegations, rewards; **direct staking actions** (delegate/undelegate/redelegate/claim) |
| Validators | Stake-weighted list + **per-validator signing heatmap** (calendar grid); governance vote history |
| Governance | Proposal list + **vote directly** from the explorer (Keplr) |
| IBC | Channel list, connection list, **IBC asset list with denom trace** |
| Staking | APR calculator, community pool balance |
| Parameters | All chain params live |
| Consensus | **Live CometBFT consensus round display**: round, step, per-validator votes, power, time in step |
| Faucet | **Built-in testnet faucet** with configurable drip limits |
| Token Send | **Send tokens directly from the explorer** (Keplr/Leap wallet) |
| Multi-Wallet | Keplr, Leap, OKX Wallet |

**Current status:** Functional (shows generic Cosmos SDK data; custom modules show as raw bytes).  
**Key weakness:** Stake-weighted validator view (no slot awareness), zero CosmWasm support, zero EVM support, no oracle/bridge/milestone/settlement/certification pages, no analytics charts.

---

## Appendix B — Current Chain Explorer Inventory

### B.1 — What All Three Together Cover

| Domain | Covered By |
|---|---|
| Cosmos blocks + txs | Ping.pub ✅ |
| Cosmos accounts + staking + delegation actions | Ping.pub ✅ |
| Governance proposals + direct voting | Ping.pub ✅ |
| Validator list + per-block signing heatmap | Ping.pub ✅ (stake-weighted, not slot-based) |
| IBC channels + denom trace | Ping.pub ✅ |
| Consensus rounds (live) | Ping.pub ✅ |
| Testnet faucet | Ping.pub ✅ |
| EVM blocks + txs + accounts | Blockscout ⚠️ (non-functional) |
| ERC-20 / ERC-721 / ERC-1155 tokens | Blockscout ⚠️ (non-functional) |
| Verified Solidity contracts + read/write | Blockscout ⚠️ (non-functional) |
| Etherscan REST API + GraphQL | Blockscout ⚠️ (non-functional) |
| CosmWasm contract execute/query | Celatone ⚠️ (non-functional, no schemas) |
| CosmWasm JSON schema decode | Celatone ⚠️ (non-functional, no schemas) |

### B.2 — What All Three Together Still Cannot Show (14 gaps)

1. Slot-based validator view (30 fixed slots, equal power)
2. x/certification: attestation windows + degraded mode flag
3. x/oracle: commit-reveal rounds, per-validator participation
4. Oracle feed OHLC price charts
5. x/milestone: state machine transitions, deadline pause timeline
6. x/settlement: Ed25519 domain separator, chain_id binding proof
7. x/bridge: end-to-end BSC → Cosmos flow as a unified view
8. Bridge supply invariant meter (`cosmos_minted + bsc_circulating = S`)
9. Relayer promotion ladder (Primary/Secondary/Candidate)
10. Bridge circuit-breaker history
11. Unified Cosmos + EVM address view (one page, both runtimes)
12. Global unified search across all entity types
13. TimescaleDB analytics charts (`tps_1h`, `block_time_1h`, `validator_uptime_1d`, `bridge_volume_1h`)
14. NATS server-push live panels (all three explorers poll)

### B.3 — Features Inherited from Each Existing Explorer

| Feature | Source | Phase Delivered |
|---|---|---|
| Live consensus round display (slot badges added) | Ping.pub | Phase 1 |
| Testnet faucet (env-disabled on mainnet) | Ping.pub | Phase 1 |
| Direct staking actions in explorer | Ping.pub | Phase 2 |
| Direct governance vote from explorer | Ping.pub | Phase 2 |
| IBC denom trace / asset list | Ping.pub | Phase 2 |
| Per-validator signing heatmap | Ping.pub | Phase 2 |
| APR calculator + community pool display | Ping.pub | Phase 2 |
| Multi-wallet (Keplr/Leap/Cosmostation) | Ping.pub + Celatone | Phase 1 |
| Token send form from explorer | Ping.pub | Phase 2 |
| Gas estimation simulation before broadcast | Celatone | Phase 2 |
| Wasm checksum search | Celatone | Phase 2 |
| Admin override history on contracts | Celatone | Phase 2 |
| Sourcify automatic verification | Blockscout | Phase 3 |
| Similar contract detection (same bytecode) | Blockscout | Phase 3 |
| Address CSV export | Blockscout | Phase 2 (Cosmos), Phase 3 (EVM) |
| Address webhook alerts | Blockscout | Phase 4 |
| Admin re-indexing panel | Blockscout | Internal only, Phase 4 |

---

## Appendix C — Feature Comparison Matrix

`✅` = covered | `❌` = not covered | `⚠️` = partial | `★` = unique to Sovereign L1 Explorer

| Feature | Etherscan | Polygonscan | Mintscan | Celatone | Blockscout | Ping.pub | Current 3 | **SL1 Explorer** |
|---|---|---|---|---|---|---|---|---|
| **COSMOS** | | | | | | | | |
| Blocks + Txs | ❌ | ❌ | ✅ | ⚠️ | ❌ | ✅ | ✅ | ✅ |
| Accounts + staking | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ |
| Governance (vote from explorer) | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| IBC channels + denom trace | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ | ✅ | ✅ + stuck packet alerts |
| Consensus rounds live | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ + slot badges |
| Direct staking actions | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| Testnet faucet | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ | ✅ |
| **COSMWASM TOKENS** | | | | | | | | |
| CW-20 (fungible token) | ❌ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ⚠️ | ✅ holders + transfers |
| CW-721 (NFT collection) | ❌ | ❌ | ❌ | ⚠️ | ❌ | ❌ | ⚠️ | ✅ gallery + per-token detail |
| CW-1155 (multi-token) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ token ID list + balances |
| CosmWasm execute + query interface | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ⚠️ | ✅ + gas simulation |
| **EVM TOKENS** | | | | | | | | |
| ERC-20 | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ |
| ERC-721 (NFTs) | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ gallery + per-token |
| ERC-1155 (multi-token) | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ `/evm/tokens/:addr/multi` |
| ERC-4626 (yield vault) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ✅ auto-detected on contract |
| Verified contracts + read/write | ✅ | ✅ | ❌ | ✅ CW | ✅ | ❌ | ⚠️* | ✅ both EVM + CW |
| Sourcify auto-verification | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ |
| Similar contract detection | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ |
| Address CSV export | ❌ | ✅ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ Cosmos + EVM |
| Address webhook alerts | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ Cosmos + EVM |
| **UNIFIED** | | | | | | | | |
| Unified Cosmos + EVM address page | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ single page both runtimes |
| Global search (all entity types) | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ⚠️ | ❌ 3 bars | ★ unified 7 entity types |
| **CUSTOM MODULES ★** | | | | | | | | |
| Slot-based validator (30 slots) | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ slot grid + equal-power |
| Certification + degraded mode | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ live degraded mode badge |
| Oracle commit-reveal rounds | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ commit/reveal/miss per validator |
| Oracle staleness state machine | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ animated 3-state indicator |
| Milestone state machine + pause timeline | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ deadline pause visualised |
| Settlement Ed25519 + domain separator | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ chain_id binding proof |
| BSC bridge: lock → quorum → mint | ❌ | ⚠️ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ full end-to-end flow |
| Bridge supply invariant gauge | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ live animated meter |
| Relayer promotion ladder | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ tier visualisation |
| Bridge circuit-breaker history | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ pause/unpause log |
| **ANALYTICS** | | | | | | | | |
| TPS from TimescaleDB aggregate | ⚠️ | ⚠️ | ⚠️ | ❌ | ⚠️ | ❌ | ⚠️ | ★ direct tps_1h |
| Block time percentile histogram | ✅ | ✅ | ✅ | ❌ | ✅ | ❌ | ⚠️ | ★ percentile_agg |
| Oracle OHLC price chart | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ per feed, oracle_price_1h |
| Validator uptime heatmap (slot×day) | ❌ | ⚠️ | ⚠️ | ❌ | ❌ | ✅ per-block | per-block | ★ validator_uptime_1d |
| Bridge volume chart | ❌ | ✅ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ bridge_volume_1h |
| **API** | | | | | | | | |
| Etherscan REST (ERC-1155 txs) | ✅ | ✅ | ❌ | ❌ | ⚠️ | ❌ | ⚠️* | ✅ incl. token1155tx |
| GraphQL | ❌ | ❌ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ via Blockscout |
| WebSocket eth_subscribe | ✅ | ✅ | ❌ | ❌ | ✅ | ❌ | ⚠️* | ✅ |
| gRPC server-streaming | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ | ★ StreamChainStats |

> `⚠️*` = available only when Blockscout is functional (currently non-functional in docker-compose)

---

## Appendix D — Architecture Deep-Dive

### D.1 — Data Sources (6 feeds)

| Source | Port | What It Provides |
|---|---|---|
| CometBFT RPC | 26657 | Blocks, txs, validators, events, WebSocket, consensus rounds |
| Cosmos gRPC | 9090 | All custom module queries (all 7 modules), CosmWasm, governance, IBC |
| Backend gRPC-API | 9091 | TimescaleDB-backed aggregates: tps_1h, block_time_1h, oracle_price_1h, validator_uptime_1d, bridge_volume_1h |
| EVM JSON-RPC | 8545 / 8546 WS | EVM blocks, txs, eth_getLogs, eth_call, newBlock subscription |
| BSC EVM RPC | external | LockBox contract events, BSC finality tracking |
| NATS JetStream | 4222 | Real-time push via `account:explorer` stream |

### D.2 — Tech Stack

**Frontend**

| Layer | Technology |
|---|---|
| Framework | Next.js 14 (App Router) — RSC for static pages, Client Components for live panels |
| Language | TypeScript 5.x strict |
| Styling | Tailwind CSS + shadcn/ui |
| Charts | Recharts (time-series, candlesticks, heatmaps, bar charts) |
| Tables | TanStack Table v8 |
| Query | TanStack Query v5 |
| Forms | React Hook Form + Zod |
| JSON view | react-json-view |
| Syntax HL | Prism.js (contract source pages) |
| Cosmos wallets | @cosmjs/stargate + @keplr-wallet/types (Keplr, Leap, Cosmostation) |
| EVM wallet | wagmi v2 + viem + RainbowKit (MetaMask + WalletConnect) |
| gRPC-Web | buf-generated stubs from @workspace/api-spec |
| State | Zustand (wallets, theme) + URL search params (filters) |
| Themes | next-themes (dark/light) |

**Backend**

| Service | Language | Notes |
|---|---|---|
| Explorer Indexer | Go | jackc/pgx v5, nats.go, go-ethereum (BSC watcher), grpc-go |
| gRPC API Server | Go | grpc-gateway v2, pgxpool, go-redis/v9 |
| Blockscout | Docker | Retained as EVM sub-system; Sourcify auto-verify enabled |
| Redis 7 | K8s StatefulSet | Hot cache (1-block TTL for latest block, validator set) |

**Databases**

| DB | Engine | Purpose |
|---|---|---|
| Explorer DB | PostgreSQL 16 + TimescaleDB | Indexed Cosmos blocks, txs, accounts, all custom module events |
| Write DB | PostgreSQL 16 + TimescaleDB | Existing CQRS source — read-only access from gRPC API |
| Blockscout DB | PostgreSQL 16 | EVM indexer data (Blockscout owns; isolated) |
| Relayer DB | PostgreSQL 16 | Bridge nonces and votes (existing) |

### D.3 — Caching

| Layer | TTL | What Is Cached |
|---|---|---|
| Redis (hot) | ~2s (1 block) | Latest block, active validator set, consensus round |
| Redis | 30s | Block/tx page rendered JSON (SSR cache) |
| Redis | Forever | Verified contract ABI, wasm checksums |
| CDN | 5 min | Static assets, prerendered SEO pages |
| PostgreSQL MV | Hourly refresh | `explorer.search_index` for global search |

### D.4 — Kubernetes Workloads

| Workload | Type | Replicas |
|---|---|---|
| Next.js frontend | Deployment + HPA | 3 min |
| Explorer Indexer | StatefulSet (singleton) | 1 (advisory lock) |
| gRPC API Server | Deployment + HPA | 3 min |
| Redis | StatefulSet | 1 (Sentinel option for HA) |
| Blockscout | Deployment | 1 |
| PgBouncer | Deployment | 2 |

---

## Appendix E — Monitoring & Alerts

### E.1 — Prometheus Metrics

| Metric | Description |
|---|---|
| `explorer_indexer_block_lag_seconds` | Time between chain head and last indexed block |
| `explorer_indexer_last_indexed_height` | Last successfully indexed block height |
| `explorer_indexer_events_decoded_total{type}` | Events decoded by custom module type |
| `explorer_api_rpc_duration_seconds{method}` | gRPC RPC latency histogram |
| `explorer_api_active_streams` | Active server-streaming connections |
| `explorer_api_request_total{method,status}` | Total requests by method and status |
| `explorer_search_query_duration_ms` | Search query latency histogram |
| `explorer_cache_hit_ratio{key_type}` | Redis cache hit % by key type |
| `explorer_blockscout_sync_lag_blocks` | EVM blocks Blockscout is behind head |
| `explorer_frontend_ssr_duration_ms{route}` | Next.js SSR render time by route |
| `explorer_csv_export_duration_seconds` | CSV export generation time |
| `explorer_webhook_delivery_total{status}` | Webhook deliveries by success/failure |

### E.2 — Grafana Dashboards

- **Indexer health:** block lag trend, throughput (blocks/sec), events decoded by type
- **API performance:** RPC p50/p95/p99 by method, active streams, error rate
- **Cache efficiency:** hit rate by key type, Redis memory, eviction rate
- **Frontend:** TTFB by route, Core Web Vitals (LCP, CLS, FID)
- **Database:** query duration by table, connection pool saturation, slow queries
- **Blockscout:** EVM indexer lag, tx processing rate, DB query time
- **Webhooks:** delivery success rate, retry queue depth, failure reasons

### E.3 — Alerts

| Alert | Threshold | Severity | Action |
|---|---|---|---|
| Indexer lag | > 10 blocks | P1 | Page on-call; check CometBFT RPC |
| Indexer stopped | 60s no new block | P0 | Page on-call; restart pod |
| API p95 latency | > 500ms | P2 | Check TimescaleDB query plan; Redis hit rate |
| API error rate | > 1% | P1 | Check gRPC logs; DB connectivity |
| Blockscout lag | > 100 EVM blocks | P2 | Restart Blockscout; check EVM RPC :8545 |
| Redis memory | > 80% | P2 | Evict cold keys |
| Cache hit rate | < 80% | P2 | Review TTL config |
| Next.js SSR p95 | > 1s | P2 | Check RSC rendering; DB query time |
| Webhook failure rate | > 10% | P2 | Check delivery endpoint; retry queue |

---

## Appendix F — Open Questions

| # | Question | Recommendation |
|---|---|---|
| F.1 | Explorer at `/explorer` (new artifact) or replacing existing frontend at `/`? | New artifact at `/explorer`; existing frontend is the dApp |
| F.2 | Explorer DB: new 5th PostgreSQL instance, or shared Read DB with `schema=explorer`? | Share Read DB with new `explorer` schema to avoid 5th instance |
| F.3 | Explorer NATS account: new `account:explorer` or subscribe to existing stream? | New `account:explorer` — isolates explorer consumers |
| F.4 | Blockscout: same k8s cluster or separate service? | Same cluster, separate namespace, strict network policy |
| F.5 | Public domain for explorer? (e.g. `explorer.yourchain.io`) | Decide before Phase 1; needed for CORS + SEO meta from Day 1 |
| F.6 | Explorer read-only in Phase 1, or wallet+broadcast from Day 1? | Read-only Phase 1; wallet actions start Phase 2 Week 8 |
| F.7 | Public REST API: API keys required or fully open? | Open by default (10 req/s per IP); keys for higher limits (Phase 4) |
| F.8 | Keep Ping.pub / Blockscout / Celatone running in parallel during build? | Yes — keep all three live until Phase 4 deliverable validated on testnet |
| F.9 | CW-721 IPFS gateway: use public (ipfs.io) or self-hosted? | Public gateway (ipfs.io) for Phase 2; evaluate self-hosted in Phase 4 if latency is a problem |
| F.10 | ERC-4626 vault read interface: include deposit/withdraw forms or read-only only? | Read-only (totalAssets, convertToShares) in Phase 3; write in Phase 4 if needed |

---

## Summary

| Dimension | Value |
|---|---|
| **Total pages** | 61 routes |
| **Token standards covered** | 11 (native coin, ICS-20, CW-20, CW-721, CW-1155, BEP-20 bridged, ERC-20, ERC-721, ERC-1155, ERC-4626, Cosmos-minted bridged) |
| **Data sources** | 6 (CometBFT, Cosmos gRPC, Backend gRPC-API, EVM JSON-RPC, BSC RPC, NATS) |
| **Phases** | 4 phases over 20 weeks |
| **Phase 1 delivers** | Real blocks/txs/accounts, consensus rounds, faucet |
| **Phase 2 delivers** | All 7 custom modules, all Cosmos/CosmWasm token standards, IBC, governance, staking actions |
| **Phase 3 delivers** | Bridge end-to-end, all EVM token standards, analytics dashboard, developer hub, contract verification |
| **Phase 4 delivers** | Public API, global search, hardening, decommission of 3 existing explorers |
| **Explorers replaced** | Ping.pub + Blockscout (standalone) + Celatone |
| **Features unique to this explorer (★)** | 10 |
| **Features inherited from existing explorers** | 17 |
| **Gaps in current 3 explorers** | 14 documented (all filled by this explorer) |

---
---

# Appendix E — Etherscan Parity: Complete Missing Features & Data Fields

> **Purpose:** This appendix documents every feature and data field present on Etherscan that is not yet specified in the existing plan above. Nothing in the original plan is removed — this section only adds what is missing. The standard target is **100% Etherscan feature parity** for all applicable pages, mapped to the Sovereign L1 chain's equivalent data.

---

## E.1 — Global Header Bar (always visible on every page)

Etherscan shows a persistent top bar on every page. Sovereign L1 Explorer must do the same.

| Element | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Token price | ETH Price: $1,616.40 (-3.09%) | SOV/USD: from oracle feed (x/oracle) | 1 |
| BTC equivalent | @ 0.026591 BTC | (omit or show vs BNB) | 3 |
| 24H price change % | -3.09% (red/green) | from oracle `SOV/USD` 24H delta | 1 |
| Gas price | Gas: 0.121 Gwei | Fee: X uSLT (from x/feemarket base fee) | 1 |
| Global search bar | Address / Txn Hash / Block / Token / Domain Name | Address (bech32+0x) / Tx Hash / Block / Token / Contract / Proposal | 1 |
| Filter dropdown on search | All Filters → specific type | All / Block / Tx / Address / Token / Contract / Proposal | 4 |
| Dark/light mode toggle | ✅ | ✅ (next-themes, already in plan) | 1 |
| Settings icon | ✅ | ✅ | 1 |
| Sign In | ✅ (optional account) | ✅ (optional wallet-based session) | 4 |

---

## E.2 — Home Page: Missing Data Fields

The existing plan lists home page as "live block ticker, TPS, validator count, recent blocks + txs feed, global search." These additional Etherscan fields are missing:

### E.2.1 — Stats Panel (top of home page)

| Field | Etherscan | Sovereign L1 Equivalent |
|---|---|---|
| Token price | ETH Price + USD | SOV Price (USD) from oracle |
| Market cap | $195,073,151,525.00 | Circulating Supply × SOV/USD |
| BTC equivalent price | @ 0.026591 BTC | (optional) |
| Total Transactions | 3,556.32 M | Total tx count from `explorer.transactions` |
| Live TPS | 39.1 TPS | from `tps_1h` aggregate |
| Median Gas Price | 0.121 Gwei (<$0.01) | Median fee in uSLT + USD |
| Last Finalized Block | 25391470 | Last finalized block height |
| Last Safe Block | 25391534 | CometBFT last committed height |
| Transaction History (14D) | Sparkline area chart | 14-day daily tx count from `block_stats` |

### E.2.2 — Latest Blocks Table (home page)

Current plan has: height, time, proposer, tx count, gas, block time delta.  
**Missing fields:**

| Column | Etherscan | Sovereign L1 Equivalent |
|---|---|---|
| Block reward (ETH) | 0.00487 ETH badge | Block reward in SOV (staking emissions for proposer) |
| Miner label | "Titan Builder" (named) | Validator display name + moniker |
| Tx count format | "229 txns in 12 secs" | "229 txs in Ns" showing block time inline |

### E.2.3 — Latest Transactions Table (home page)

Current plan: hash, type, from, to, amount.  
**Missing fields:**

| Column | Etherscan | Sovereign L1 Equivalent |
|---|---|---|
| Method badge | `Transfer`, `Deposit`, `Swap` | Cosmos Msg type short label (e.g. `Delegate`, `BridgeIn`) |
| Direction arrow | → icon between from/to | → icon |
| To: contract label | "Coinbase: MEV Builder" | Known contract label (Treasury, Bridge, Oracle, etc.) |
| Amount badge | 0.00422 Eth right-aligned | SOV amount right-aligned |

---

## E.3 — Blocks List Page (`/blocks`): Missing Fields

### E.3.1 — Summary Banner (above block table)

Etherscan shows 4 stat cards at top of `/blocks`. Current plan has no equivalent.

| Stat Card | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Network Utilization (24H) | 50.5% | Gas used / Gas limit × 100, 24H rolling | 1 |
| Last Safe Block | 25397300 | CometBFT committed height | 1 |
| Blocks by MEV Builders (24H) | 91.7% | (omit — no MEV builders; show "Blocks by Top Validator") | 1 |
| Burnt Fees 🔥 | 4,630,866.19 ETH | Total fees burned (EIP-1559 base fee burn) | 1 |

### E.3.2 — Blocks Table: Missing Columns

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Slot | Beacon chain slot # | (omit — Cosmos has no slots) | — |
| Blobs | Count + % capacity | (omit — no EIP-4844 yet; add when relevant) | — |
| Gas Used | Raw + % of limit (progress bar) | Gas used raw + % of limit with inline mini-bar | 1 |
| Gas Limit | 60,000,000 | Block gas limit from params | 1 |
| Base Fee | Gwei | uSLT base fee (x/feemarket) | 1 |
| Reward | ETH | SOV block reward | 1 |
| Burnt Fees (ETH) | Amount + % | Burnt fees in uSLT + % of total reward | 1 |
| Fee Recipient label | "Titan Builder" | Validator moniker (linked to `/validators/:addr`) | 1 |

### E.3.3 — Page-Level Controls

| Control | Etherscan | Sovereign L1 | Phase |
|---|---|---|---|
| Download Page Data | ✅ CSV button top-right | ✅ CSV download of current page | 2 |
| Pagination | First / Prev / Page N of M / Next / Last | ✅ cursor-based, same labels | 1 |
| Total count header | "Total of 25,397,347 blocks (Showing #N to #M)" | ✅ same format | 1 |

---

## E.4 — Block Detail Page (`/blocks/:height`): Missing Fields

Etherscan block detail has 4 tabs and many fields not yet specified.

### E.4.1 — Tabs

| Tab | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Overview | ✅ | ✅ | 1 |
| Consensus Info | Beacon chain epoch/slot/attestations | CometBFT: round, step, commit signatures, pre-commit count | 1 |
| MEV Info | Builder, MEV boost relay, bid | (omit or repurpose: show top-fee txs in block) | — |
| Participants | Validator withdrawals list | Staking: delegations/undelegations processed in block | 2 |

### E.4.2 — Overview Tab: Missing Fields

Current plan has: proposer, txs, gas, app_hash. Missing:

| Field | Etherscan Label | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Block Height | ✅ with ← → navigation arrows | ✅ prev/next block nav | 1 |
| Status | ✅ Finalized / Unfinalized badge | ✅ Committed / Finalized (CometBFT) | 1 |
| Timestamp | "19 hrs ago \| Jun-25-2026 02:10:23 AM +UTC" | ✅ relative + absolute, timezone toggle | 1 |
| Proposed On | Slot + Epoch | Round + Step (CometBFT) | 1 |
| Transactions | "229 transactions and 100 contract internal transactions" | "N txs (M Cosmos, K EVM, J CosmWasm)" | 1 |
| Withdrawals | 16 withdrawals in this block | Undelegation completions processed in block | 2 |
| Fee Recipient | Builder address + label | Validator address + moniker | 1 |
| Block Reward | ETH breakdown | SOV: base reward + tx fees + burned amount | 1 |
| Total Difficulty | Cumulative PoW difficulty | (omit — PoS/CometBFT) | — |
| Size | Bytes | Block size in bytes | 1 |
| Gas Used | Raw + % + progress bar | ✅ | 1 |
| Gas Limit | Raw | ✅ | 1 |
| Base Fee Per Gas | Gwei + ETH | uSLT base fee (x/feemarket) | 1 |
| Burnt Fees | ETH amount | Burnt fees in uSLT | 1 |
| Extra Data | Hex + decoded text | CometBFT proposer metadata / app_hash | 1 |
| Hash | Full block hash | ✅ full block hash | 1 |
| Parent Hash | Linked to prev block | ✅ parent block hash linked | 1 |
| StateRoot | Hex | App hash (equivalent) | 1 |
| Withdrawals Root | Hex | (omit for Cosmos; show last_commit_hash) | 1 |
| Nonce | Hex | (omit — PoS has no nonce) | — |

### E.4.3 — Consensus Info Tab: Full Fields

| Field | Description | Phase |
|---|---|---|
| Epoch / Round | CometBFT consensus round number | 1 |
| Step | Propose / Prevote / Precommit / Commit | 1 |
| Pre-commit count | N of M validators pre-committed | 1 |
| Pre-commit voting power | % of total voting power | 1 |
| Commit signatures | Per-validator: address + signed/missed badge | 1 |
| Last commit hash | Hex | 1 |
| Evidence | Equivocation evidence (if any) | 2 |

---

## E.5 — Transactions List Page (`/txs`): Missing Fields

### E.5.1 — Summary Banner (above tx table)

| Stat Card | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Transactions (24H) | 2,700,616 (+6.62%) | Total txs last 24H + % change vs prior 24H | 1 |
| Pending Transactions (Last 1H) | 105,393 (Average) | Mempool size (CometBFT) rolling 1H average | 1 |
| Total Transaction Fee (24H) | 185.58 ETH (+39.67%) | Total fees collected in SOV last 24H | 1 |
| Avg Transaction Fee (24H) | 0.16 USD (+48.35%) | Avg fee per tx in USD (fee × SOV/USD price) | 1 |

### E.5.2 — Transactions Table: Missing Columns

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Eye icon | Marks internal tx | Mark bridge/oracle/custom msg type | 1 |
| Method badge | `Transfer`, `Deposit`, `Swap` | Cosmos Msg type: `Delegate`, `BridgeIn`, `OracleReveal` | 1 |
| IN / OUT badge | Green IN / Blue OUT (relative to searched address) | Same on address-scoped tx list | 1 |
| Direction arrow → | Between from/to | ✅ | 1 |
| To: contract label | "Circle: USDC Token" | "Bridge Contract", "Treasury", "Oracle Module" | 1 |
| Txn Fee column | Right-aligned in ETH | Right-aligned in SOV | 1 |
| Hide Low Value toggle | Hides dust txs | ✅ hide txs < threshold | 2 |
| Download Page Data | CSV button | ✅ | 2 |

---

## E.6 — Transaction Detail Page (`/txs/:hash`): Missing Fields

This is the most field-rich page on Etherscan. Many fields are not yet in the plan.

### E.6.1 — Transaction Action Banner (top of page)

Etherscan shows a human-readable summary at the very top:
> `Transfer 0.000000000000031337 ETH (<$0.01) to [address]`

Sovereign L1 equivalent — show decoded action summary:
- `Delegate 1,000 SOV to cosmosvaloper1abc...`
- `Bridge 500 SOV from BSC (nonce 0x1a2b)`
- `Oracle Reveal: BTC/USD = $67,234 by cosmosvaloper1xyz...`
- `Execute Contract: [label] → method_name`
- `Transfer 100 SOV to cosmos1recipient...`

| Element | Description | Phase |
|---|---|---|
| Action icon | Token logo or type icon | 1 |
| Action summary | Human-readable 1-line description | 1 |
| USD value badge | `<$0.01` or `$1,234.56` | 1 |

### E.6.2 — Overview Tab: Complete Field List

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Transaction Hash | Full hash + copy | ✅ | 1 |
| Block | Block # + "N Block Confirmations" badge | Block height + confirmation count | 1 |
| Timestamp | Relative + absolute + UTC toggle | ✅ | 1 |
| Sponsored | Ad slot | (omit) | — |
| From | Address + ENS label + copy | Sender bech32 + label + copy | 1 |
| To | Address + label + copy (or contract icon) | Recipient bech32/0x + label + copy | 1 |
| Value | ETH amount + USD | SOV amount + USD (oracle price) | 1 |
| Transaction Fee | ETH + USD | SOV fee + USD | 1 |
| Gas Price | Gwei + ETH | uSLT base fee + priority fee | 1 |
| **More Details (expandable):** | | | |
| Gas Limit & Usage by Txn | "21000 \| 21000 (100%)" | Gas limit \| Gas used (%) | 1 |
| Gas Fees (Base) | Gwei | uSLT base fee per gas | 1 |
| Gas Fees (Max) | Gwei | uSLT max fee per gas | 1 |
| Gas Fees (Max Priority) | Gwei | uSLT priority tip | 1 |
| Burnt & Txn Savings Fees | ETH burnt + ETH saved vs max | uSLT burnt + uSLT saved | 1 |
| Other Attributes — Txn Type | EIP-1559 (Type 2) | Cosmos SDK Tx type: Direct / Legacy | 1 |
| Other Attributes — Nonce | Account nonce at submission | Account sequence number | 1 |
| Other Attributes — Position in Block | Index within block | Tx index within block | 1 |
| Input Data | Hex + ABI decoded function call | Cosmos Msg decoded (all fields labeled) | 1 |
| Private Note | (login required) | (login required — Phase 4) | 4 |

### E.6.3 — State Tab

Etherscan shows state changes caused by the transaction (before/after for each touched storage slot and balance).

| Field | Description | Phase |
|---|---|---|
| Address | Each address whose state changed | 3 |
| Before balance | Balance before tx | 3 |
| After balance | Balance after tx | 3 |
| Storage changes | Contract slot: before → after (EVM only) | 3 |

---

## E.7 — Address Page (`/address/:addr`): Missing Fields

### E.7.1 — Header Row: Missing Elements

| Element | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| ENS / name label | `vitalik.eth` badge | Name Service resolution (if deployed) | 4 |
| Tags | `Authority`, `Gitcoin Grantee`, `Delegated to...` | Protocol tags: `Validator`, `Oracle Operator`, `Bridge Relayer`, `Treasury` | 2 |
| Watchlist star | ⭐ favorite address | ✅ watchlist (requires login) | 4 |
| QR code button | ✅ | ✅ | 2 |
| Copy address | ✅ | ✅ | 1 |
| View as grid | switch view | ✅ | 2 |
| Comment count badge | 99+ | (omit) | — |

### E.7.2 — Overview Panel: Missing Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| ETH Balance | 5.6908... ETH | SOV balance (native coin) | 1 |
| ETH Value | $8,907.80 (@ $1,565.28/ETH) | SOV value in USD (balance × oracle price) | 1 |
| Token Holdings | >$85,761.38 (>401 Tokens) dropdown | CW-20 + ERC-20 portfolio value dropdown | 2 |

### E.7.3 — More Info Panel: Missing Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Private Name Tags | Add button (login) | ✅ add private label (login) | 4 |
| Transactions Sent | "Latest: 29 days ago \| First: 10 yrs ago" | Latest tx timestamp + First tx timestamp | 1 |
| Funded By | Which address first funded this account | First inbound tx sender + timestamp | 2 |

### E.7.4 — Multichain Info Panel

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Portfolio value | $2,003,045,579.63 | Total SOV + ERC-20 + CW-20 portfolio in USD | 3 |
| Addresses found | 18 addresses via Blockscan | (show bech32 + 0x dual representation) | 1 |

### E.7.5 — Address Tabs: Complete List

Current plan has tabs but misses several. Full required tab list:

| Tab | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Transactions | ✅ | ✅ Cosmos + EVM txs | 1 |
| Internal Transactions | EVM contract-to-contract calls | EVM internal txs (Blockscout) | 3 |
| Token Transfers (ERC-20) | All ERC-20 moves | ERC-20 + CW-20 transfers | 3 |
| NFT Transfers | ERC-721 + ERC-1155 | ERC-721 + CW-721 + CW-1155 | 3 |
| Other Transactions | MEV, bundle txs | Bridge txs, oracle txs, governance votes | 2 |
| Analytics | Activity charts over time | Tx frequency chart, volume chart | 3 |
| Assets | Token portfolio breakdown | SOV + CW-20 + ERC-20 holdings table | 3 |
| Cards | NFT visual card gallery | CW-721 + ERC-721 NFT gallery | 3 |

### E.7.6 — Address-Scoped Transaction Table: Missing Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| IN / OUT badge | Green IN / Blue OUT relative to address | ✅ same | 1 |
| ANY / IN / OUT filter | Dropdown filter | ✅ same filter | 1 |
| Hide Low Value toggle | Hides dust/spam transfers | ✅ configurable threshold | 2 |
| Pending tx indicator | `(pending)` in Block column | ✅ show mempool pending | 1 |
| Advanced Filter | Block range, age range, direction, method, from, to | ✅ all same filters | 2 |
| Download Page Data | CSV | ✅ | 2 |
| tx count header | "Latest 25 from a total of 78,216 transactions (+1 Pending)" | ✅ same format | 1 |

---

## E.8 — Contract Address Page: Missing Fields

When an address is a smart contract, Etherscan shows additional information.

### E.8.1 — Contract Header

| Element | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| "Contract" label | Shows "Contract" instead of "Address" | ✅ | 2 |
| Source Code badge | `Source Code (Proxy)` | `Verified Source` / `Unverified` | 2 |
| Implementation address | `Implementation: 0x43...dd` | Implementation contract linked (proxy pattern) | 3 |
| Contract tags | `#Circle`, `#Stablecone`, `Token Contract` | Custom tag system: `#Bridge`, `#Oracle`, `#Governance` | 2 |

### E.8.2 — Contract Tab: Missing Sub-tabs and Fields

| Sub-tab | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Contract (source) | ✅ | ✅ (in plan) | 3 |
| Read as Proxy | ✅ reads implementation state | ✅ for upgradeable EVM contracts | 3 |
| Write as Proxy | ✅ writes to implementation | ✅ for upgradeable EVM contracts | 3 |
| Past Implementations | List of prior implementation addresses | ✅ migration history | 3 |
| Implementation For | Which proxies point to this implementation | ✅ reverse proxy lookup | 3 |

### E.8.3 — Contract Source Code Panel Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Source Code Verified | ✅ Exact Match + Proxy badge | ✅ Exact Match / Partial Match | 3 |
| Contract Name | `FiatTokenProxy` | Contract name from compiler | 3 |
| Compiler Version | `v0.4.24+commit.e67f0147` | Full compiler version | 3 |
| Optimization Enabled | `No with 200 runs` | ✅ | 3 |
| Other Settings | `default evmVersion` | ✅ | 3 |
| License | `NA` / `MIT` / `GPL-3.0` | ✅ SPDX license identifier | 3 |
| Audit Report badge | ✅ if audit uploaded | ✅ link to uploaded audit PDF | 3 |
| Source file browser | Left panel file tree | ✅ multi-file contract explorer | 3 |
| Code Reader | AI-powered code explanation | (optional, Phase 4) | 4 |
| Proxy Checker button | ✅ verify proxy pattern | ✅ detect EIP-1967 / EIP-897 | 3 |
| Similar Contracts button | ✅ same bytecode elsewhere | ✅ | 3 |
| More Options | Download ABI / Bytecode / Opcodes | ✅ same options | 3 |

### E.8.4 — Events Tab (missing from plan)

| Field | Description | Phase |
|---|---|---|
| Block # | Block where event was emitted | 3 |
| Tx Hash | Transaction that emitted the event | 3 |
| Event name | Decoded event name | 3 |
| Event data | ABI-decoded event arguments (indexed + non-indexed) | 3 |
| Topic 0 | keccak256 of event signature | 3 |
| Topic 1–3 | Indexed parameters | 3 |
| Data | Non-indexed parameters (hex + decoded) | 3 |
| Age | Timestamp | 3 |

### E.8.5 — Info Tab (missing from plan)

| Field | Description | Phase |
|---|---|---|
| Contract Creator | Deployer address + deploy tx hash | 2 |
| Creation Date | Block height + timestamp of deployment | 2 |
| Token Tracker | If ERC-20: token name + price linked | 3 |
| Tags | Protocol labels | 2 |

---

## E.9 — Token Tracker Page (`/tokens`): Missing Fields

### E.9.1 — Page Header

| Element | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Total count | "A total of 2,213,003 Token Contracts found" | Total CW-20 + ERC-20 count | 2 |
| Reputation filter | "Showing 3175 Tokens with OK or Neutral Reputation" | Verified / All reputation filter | 2 |
| Search box | Filter by name or address | ✅ | 2 |

### E.9.2 — Token Table: Missing Columns

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| # | Rank | ✅ | 2 |
| Token | Logo + Name + (Symbol) | ✅ | 2 |
| Price | USD + ETH equivalent | USD + SOV equivalent | 2 |
| Change (%) | 24H % (green/red) | ✅ from oracle or external feed | 2 |
| Volume (24H) | USD trading volume | ✅ from DEX events or transfer volume | 2 |
| Circulating Market Cap | USD (from CoinGecko/offchain) | USD from circulating supply × price | 2 |
| **Onchain Market Cap** | **USD from totalSupply() × price** | **✅ from chain state directly** | **2** |
| Holders | Count + % change 24H | ✅ distinct holder count + delta | 2 |
| Holder growth chart | Mini sparkline on holder column | ✅ tiny sparkline | 3 |

---

## E.10 — Token Detail Page (`/tokens/:addr`): Missing from Plan

This page does not exist as a standalone route in the current plan. It is needed.

### E.10.1 — Route to Add

`/tokens/:addr` — ERC-20 / CW-20 Token Detail Page (also reachable from contract address page via Token Tracker link)

### E.10.2 — Token Overview Panel Fields

| Field | Description | Phase |
|---|---|---|
| Token logo | From contract metadata or IPFS | 2 |
| Token name | e.g. "Tether USD" | 2 |
| Token symbol | e.g. "USDT" | 2 |
| Token standard | ERC-20 / CW-20 badge | 2 |
| Contract address | Linked | 2 |
| Decimals | e.g. 6 or 18 | 2 |
| Official site | Link from contract metadata | 2 |
| Social links | Twitter, Discord, Github | 2 |
| **Price** | USD price + % change 24H | 2 |
| **Fully Diluted Market Cap** | Max supply × price | 2 |
| **Circulating Supply** | From totalSupply() call | 2 |
| **Max Supply** | From contract if set | 2 |
| **Holders** | Distinct holder count | 2 |
| **Total Transfers** | Cumulative transfer count | 2 |

### E.10.3 — Token Detail Tabs

| Tab | Content | Phase |
|---|---|---|
| Transfers | All Transfer events: from, to, amount, block, tx hash, age | 2 |
| Holders | Ranked holder list: address, quantity, % of supply, value USD | 2 |
| Info | Contract info, verified status, links | 2 |
| Analytics | Price chart (OHLC if oracle), holder growth chart, transfer volume chart | 3 |
| Contract | Source code (same as contract page) | 3 |

---

## E.11 — Pending Transactions Page (`/txs/pending`): Missing Fields

Current plan: "Mempool pending tx list: age in pool, gas price, size." Missing:

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Transaction Hash | Truncated + copy + link | ✅ | 1 |
| Method | Function label badge | Cosmos Msg type | 1 |
| Nonce | Sender's current nonce | Account sequence number | 1 |
| Last Seen | "less than 1 sec ago" | Time first seen in mempool | 1 |
| Gas Limit | Max gas units | ✅ | 1 |
| Gas Price | Current bid \| Max bid in Gwei | uSLT fee price \| max fee | 1 |
| From | Address + label | ✅ | 1 |
| To | Address + label | ✅ | 1 |
| Amount | ETH value | SOV value | 1 |
| **Pending Transaction Pool header** | "Pending Transaction Pool" with search | ✅ | 1 |
| **Total count** | "A total of 106,403 pending txns found" | ✅ | 1 |

---

## E.12 — Gas Tracker Page (`/gastracker`): Missing from Plan

Gas Tracker is not a standalone page in the current plan. It must be added.

### E.12.1 — Route to Add

`/gastracker` — Gas Price Tracker (mapped to `/fees` or `/gastracker` for the Sovereign L1 EVM runtime)

### E.12.2 — Gas Tracker Fields

**3-Tier Gas Card Panel:**

| Tier | Icon | Description | Sovereign L1 Equivalent |
|---|---|---|---|
| Standard 🐢 | Slow | Base fee only, ~30 sec | Base fee (uSLT) + USD cost |
| Fast 🚀 | Average | Base + small priority | Base + priority tip + USD |
| Rapid ⚡ | Instant | Max gas bid | Max fee + USD |

Each tier card shows:
- Gas price in Gwei (EVM) / uSLT (Cosmos)
- Base Fee | Priority Fee breakdown
- USD cost for standard 21,000 gas transfer
- Estimated confirmation time

**Additional Info Section:**

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Last Block | Block number | ✅ | 1 |
| Pending Queue | Mempool tx count | CometBFT mempool count | 1 |
| Avg Block Size | Tx count per block avg | ✅ | 1 |
| Avg Utilization | Gas used / Gas limit % | ✅ | 1 |
| Last Refreshed | Timestamp | ✅ | 1 |

**Featured Actions Table** (USD cost for common operations):

| Action | Standard | Fast | Rapid | Phase |
|---|---|---|---|---|
| Token Transfer (SOV) | $X | $X | $X | 1 |
| CW-20 Transfer | $X | $X | $X | 2 |
| Contract Execute | $X | $X | $X | 2 |
| Bridge (BSC→L1) | $X | $X | $X | 3 |
| Bridge (L1→BSC) | $X | $X | $X | 3 |
| NFT Mint | $X | $X | $X | 3 |
| Custom Gas | (user input) | — | — | 1 |

**Gas Price Heatmap:**

| Element | Description | Phase |
|---|---|---|
| X-axis | Hours of day (00–23 UTC) | 1 |
| Y-axis | Day of week | 1 |
| Color | Gas price intensity (light = cheap, dark = expensive) | 1 |
| Tooltip | Average gas price for that hour/day cell | 1 |

**Gas Price History Chart:**

| Element | Description | Phase |
|---|---|---|
| Time range selector | 7D / 30D / 90D | 2 |
| Chart type | Line chart | 2 |
| Data | Hourly average gas price | 2 |

---

## E.13 — NFT Top Contracts Page (`/nft-top-contracts`): Missing Fields

Current plan has `/evm/nfts` but not a dedicated top-NFTs leaderboard page. Add:

### E.13.1 — Route to Add

`/nfts` — Top NFT Collections (unified CW-721 + ERC-721 + ERC-1155 leaderboard)

### E.13.2 — NFT Leaderboard Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Time filter | 1h / 6h / 12h / 1d / 7d / 30d tabs | ✅ same time filters | 3 |
| # | Rank | ✅ | 3 |
| Collection | Logo + Name + verified badge | ✅ | 3 |
| Type | ERC-721 / ERC-1155 | ERC-721 / ERC-1155 / CW-721 / CW-1155 | 3 |
| Volume | ETH volume traded | SOV volume | 3 |
| Change (%) | % change in volume vs prior period | ✅ | 3 |
| Sales | Number of sales in period | ✅ | 3 |
| Min Price (24H) | Floor price | ✅ | 3 |
| Max Price (24H) | Ceiling price | ✅ | 3 |
| Transfers | Total transfer count | ✅ | 3 |
| Owners | Unique owner count | ✅ | 3 |
| Total Assets | Total NFT supply | ✅ | 3 |
| Download Page Data | CSV | ✅ | 3 |

---

## E.14 — Charts & Statistics Page (`/charts`): Missing Fields

Current plan has `/analytics` with some charts. A dedicated full charts page matching Etherscan's is missing.

### E.14.1 — Route

`/charts` — Chain Charts & Statistics (keeps `/analytics` for the custom module dashboard; `/charts` is the Etherscan-style stats overview)

### E.14.2 — Left Sidebar Sections

| Section | Sub-charts |
|---|---|
| Overview Stats | Live numbers panel (all key metrics) |
| Market Data | SOV daily price, market cap chart |
| Blockchain Data | Daily txs, unique addresses, new addresses, active addresses, avg block size, avg gas price, gas used |
| Network Data | Validator count history, voting power distribution |
| Contracts | Verified contracts daily, contracts deployed daily |

### E.14.3 — Overview Stats Panel (live numbers — all required)

| Metric | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Addresses (Total) | 417,834,997 | Unique addresses from `explorer.accounts` | 1 |
| Transactions (Total) | 3,558.39 M | Cumulative tx count | 1 |
| New Addresses (24H) | 192,059 (+4.52%) | New addresses + delta | 1 |
| Transactions (24H) | 2,700,616 (+6.62%) | ✅ already in plan | 1 |
| Tokens (Total) | 2,213,003 | CW-20 + ERC-20 count | 2 |
| Pending Transactions (1H) | 105,400 | CometBFT mempool count | 1 |
| Total Transaction Fee (24H) | 185.58 ETH | Total fees in SOV + USD | 1 |
| Avg Transaction Fee (24H) | 0.16 USD | ✅ | 1 |
| Contracts Deployed (Total) | 100,819,841 | CosmWasm codes + EVM contracts | 2 |
| Contracts Verified (Total) | 891,483 | Verified CosmWasm + EVM count | 2 |
| Contracts Deployed (24H) | 15,927 | ✅ | 2 |
| Contracts Verified (24H) | 279 | ✅ | 2 |
| Total Gas Used (24H) | 216,833.12 Million | ✅ | 1 |
| Network Utilization (24H) | 50.5% | ✅ | 1 |
| Blocks by Top Validator (24H) | 91.7% MEV | Top validator block share % | 1 |
| Burnt Fees 🔥 (24H) | 77.39 ETH | Burnt fees in SOV + USD | 1 |

### E.14.4 — Individual Chart Pages

Each chart in the sidebar links to a full-page chart view. All required:

| Route | Chart Name | Phase |
|---|---|---|
| `/charts/tx` | Daily Transaction Count | 1 |
| `/charts/tx-fee` | Average Transaction Fee (USD) | 1 |
| `/charts/active-addresses` | Daily Active Addresses | 1 |
| `/charts/new-addresses` | Daily New Addresses | 1 |
| `/charts/unique-addresses` | Cumulative Unique Addresses | 1 |
| `/charts/gas-price` | Average Gas Price (uSLT) | 1 |
| `/charts/gas-used` | Total Gas Used per Day | 1 |
| `/charts/block-size` | Average Block Size (bytes) | 1 |
| `/charts/block-time` | Average Block Time (seconds) | 1 |
| `/charts/tps` | Transactions Per Second | 1 |
| `/charts/block-count` | Blocks per Day | 1 |
| `/charts/burnt-fees` | Daily Burnt Fees (SOV) | 1 |
| `/charts/price` | SOV Daily Price (USD) | 2 |
| `/charts/market-cap` | SOV Market Capitalization | 2 |
| `/charts/validator-count` | Active Validator Count | 2 |
| `/charts/staking-ratio` | Staking Ratio (% bonded) | 2 |
| `/charts/contracts-deployed` | Daily Contracts Deployed | 2 |
| `/charts/contracts-verified` | Daily Contracts Verified | 2 |
| `/charts/token-transfers` | Daily ERC-20 + CW-20 Transfers | 2 |
| `/charts/nft-transfers` | Daily NFT Transfers | 3 |
| `/charts/bridge-volume` | Daily Bridge Volume (SOV) | 3 |
| `/charts/ibc-volume` | Daily IBC Transfer Volume | 2 |

Each chart page has:
- Date range selector: 7D / 30D / 90D / 180D / 1Y / All
- Download CSV button
- Chart description text
- Data source label (e.g. "Source: TimescaleDB tps_1h aggregate")

---

## E.15 — Supply Page (`/stat/supply`): Missing from Plan

### E.15.1 — Route to Add

`/stat/supply` — SOV Token Total Supply & Distribution

### E.15.2 — Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Total Supply | 120,683,711.66 ETH | Total SOV supply (genesis cap = 1,000,000,000 SOV) | 1 |
| Market Capitalization | $188,903,800,185 | Circulating Supply × SOV/USD | 2 |
| Price Per Token | $1,565.28 | SOV/USD from oracle | 1 |
| **Distribution Calculation:** | | | |
| Genesis Allocation | Crowdsale + Other | Genesis allocation breakdown | 1 |
| Block Rewards | Mining block rewards | Staking block emissions | 1 |
| Staking Rewards | Eth2 staking | Delegator rewards | 2 |
| Burnt Fees | − EIP-1559 burnt | − Fee burning (x/feemarket) | 1 |
| Current Total Supply | Running total | ✅ | 1 |
| **Supply Breakdown Chart** | Pie/donut chart by source type | ✅ | 2 |
| Data Source link | "Ether Supply API Docs" | Link to `/docs` | 4 |

---

## E.16 — New Routes to Add (Etherscan Pages Not Yet in Plan)

| Route | Description | Phase |
|---|---|---|
| `/gastracker` | Gas price tracker with 3 tiers, heatmap, featured action costs | 1 |
| `/tokens/:addr` | Individual token detail page (ERC-20 / CW-20) | 2 |
| `/nfts` | Top NFT collections leaderboard (CW-721 + ERC-721) | 3 |
| `/charts` | Full charts & statistics hub | 1 |
| `/charts/:chartName` | Individual chart pages (see E.14.4 for full list — 22 routes) | 1–3 |
| `/stat/supply` | SOV total supply + distribution breakdown | 1 |
| `/accounts` | Top addresses by SOV balance (leaderboard) | 2 |
| `/contracts/verified` | Recently verified contracts list | 2 |
| `/contracts/verified/:type` | Filter verified by: EVM / CosmWasm / Proxy | 2 |
| `/txs/internal` | All internal transactions across chain | 3 |
| `/token-approvals` | ERC-20 approval tracker for wallet address (requires connect) | 3 |
| `/address/:addr/tokencheck` | Token approval checker for address | 3 |
| `/label/:slug` | All addresses with a given label tag (e.g. `/label/treasury`) | 2 |
| `/pushnotification` | Watchlist + push notification setup (login required) | 4 |
| `/myaccount` | User account page: watchlist, private notes, API keys, notifications | 4 |
| `/api-documentation` | Redirect to `/docs` | 4 |
| `/exportData` | Bulk data export page (address + date range → CSV) | 3 |

---

## E.17 — Internal Transactions Page (`/txs/internal`): Missing Fields

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Block | Block number linked | ✅ | 3 |
| Age | Timestamp | ✅ | 3 |
| Parent Txn Hash | Outer tx hash | ✅ | 3 |
| Type | call / delegatecall / create / suicide | EVM internal call type | 3 |
| From | Caller address | ✅ | 3 |
| To | Callee address | ✅ | 3 |
| Value | ETH transferred | SOV | 3 |

---

## E.18 — Top Accounts Page (`/accounts`): Missing from Plan

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| # | Rank by balance | ✅ | 2 |
| Address | Address + label | bech32 address + label | 2 |
| Name Tag | Known label (e.g. "Binance: Hot Wallet") | Protocol label (Treasury, Reserve, Validator) | 2 |
| Balance | ETH + USD value | SOV + USD value | 2 |
| % of Total | Share of total supply | ✅ | 2 |
| Txn Count | Total tx count | ✅ | 2 |

---

## E.19 — Verified Contracts List Page (`/contracts/verified`): Missing Fields

| Column | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| # | Row number | ✅ | 2 |
| Address | Contract address | ✅ | 2 |
| Contract Name | Name from source | ✅ | 2 |
| Compiler | Version | ✅ | 2 |
| Version | Solidity/Vyper/CosmWasm | ✅ | 2 |
| Balance | ETH balance | SOV balance | 2 |
| Txns | Total interactions | ✅ | 2 |
| Setting | Optimization (Y/N) + runs | ✅ | 2 |
| Verified | Timestamp of verification | ✅ | 2 |
| Audited | Audit badge (if uploaded) | ✅ | 3 |
| License | SPDX license | ✅ | 2 |

---

## E.20 — User Account System (Login): Missing from Plan

Etherscan has an optional account system. Add to Phase 4:

| Feature | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Sign in method | Email/password | Wallet signature (EIP-4361 Sign-In with Ethereum or Cosmos equivalent) | 4 |
| Watchlist | Save addresses for quick access | ✅ | 4 |
| Private name tags | Label any address privately | ✅ | 4 |
| Private notes | Attach notes to any tx | ✅ | 4 |
| API keys | Generate keys for higher rate limits | ✅ | 4 |
| Push notifications | Email/webhook on watched address events | ✅ (extends Phase 4 webhook system) | 4 |
| Token ignore list | Hide specific tokens from portfolio | ✅ | 4 |
| TX input data decoder | Save custom ABI for any address | ✅ | 4 |

---

## E.21 — Etherscan-Compatible REST API: Complete Endpoint List

The current plan (Appendix D, Phase 4) lists 12 API endpoints. Etherscan has significantly more. Complete list:

### Module: `account`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `balance` | `address, tag` | Native balance in uSOV (wei-format) | 4 |
| `balancemulti` | `address` (comma-separated) | Multi-address balances | 4 |
| `txlist` | `address, startblock, endblock, page, offset, sort` | Tx list | 4 |
| `txlistinternal` | `address` or `txhash` or `blockrange` | Internal tx list | 4 |
| `tokentx` | `address, contractaddress` | ERC-20 / CW-20 transfer list | 4 |
| `tokennfttx` | `address, contractaddress` | ERC-721 / CW-721 transfer list | 4 |
| `token1155tx` | `address, contractaddress` | ERC-1155 / CW-1155 transfer list | 4 |
| `getminedblocks` | `address, blocktype` | Blocks proposed by validator | 4 |
| `txsBeaconWithdrawal` | `address` | (Cosmos: undelegation completions) | 4 |

### Module: `block`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `getblockreward` | `blockno` | Block reward + fee recipient | 4 |
| `getblockcountdown` | `blockno` | Estimated seconds until block | 4 |
| `getblocknobytime` | `timestamp, closest` | Block number by timestamp | 4 |
| `dailyavgblocksize` | `startdate, enddate` | Daily avg block size | 4 |
| `dailyblkcount` | `startdate, enddate` | Daily block count | 4 |
| `dailyavgblocktime` | `startdate, enddate` | Daily avg block time | 4 |
| `dailygasused` | `startdate, enddate` | Daily gas used | 4 |

### Module: `contract`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `getabi` | `address` | ABI JSON string | 4 |
| `getsourcecode` | `address` | Source + compiler + ABI + constructor args | 4 |
| `getcontractcreation` | `contractaddresses` | Creator + creation tx hash | 4 |
| `verifysourcecode` | POST: source, compiler, etc. | Verification GUID | 4 |
| `checkverifystatus` | `guid` | Pending / Pass / Fail | 4 |
| `verifyproxycontract` | `address` | Proxy verification result | 4 |
| `checkproxyverification` | `guid` | Proxy verification status | 4 |

### Module: `transaction`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `getstatus` | `txhash` | `isError`: 0/1, `errDescription` | 4 |
| `gettxreceiptstatus` | `txhash` | Receipt status: `0` fail / `1` success | 4 |

### Module: `logs`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `getLogs` | `address, fromBlock, toBlock, topic0, topic0_1_opr, topic1` | Event log array | 4 |

### Module: `token`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `tokeninfo` | `contractaddress` | Name, symbol, decimals, totalSupply, price, holders | 4 |

### Module: `stats`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `sovprice` | — | SOV/USD + SOV/BTC prices | 4 |
| `sovsupply` | — | Total SOV supply | 4 |
| `dailytxnfee` | `startdate, enddate` | Daily transaction fee total | 4 |
| `dailynewaddress` | `startdate, enddate` | Daily new addresses | 4 |
| `dailynetutilization` | `startdate, enddate` | Daily network utilization % | 4 |
| `dailytx` | `startdate, enddate` | Daily transaction count | 4 |
| `dailyavghashrate` | — | (omit — PoS) | — |
| `dailyavgblocktime` | `startdate, enddate` | Daily average block time | 4 |
| `dailyuncleblkcount` | — | (omit — no uncles in CometBFT) | — |
| `dailyavggaslimit` | `startdate, enddate` | Daily average gas limit | 4 |
| `dailyavggasprice` | `startdate, enddate` | Daily average gas price | 4 |
| `dailygasused` | `startdate, enddate` | Daily gas used | 4 |
| `dailyavgtxngaslimit` | `startdate, enddate` | Daily avg gas limit per tx | 4 |
| `dailyavgtxngasprice` | `startdate, enddate` | Daily avg gas price per tx | 4 |
| `dailytxnfee` | `startdate, enddate` | Daily total fee collected | 4 |
| `dailyburntfees` | `startdate, enddate` | Daily burnt fees (EIP-1559) | 4 |

### Module: `gastracker`

| Action | Parameters | Returns | Phase |
|---|---|---|---|
| `gasoracle` | — | `SafeGasPrice`, `ProposeGasPrice`, `FastGasPrice`, `suggestBaseFee`, `gasUsedRatio` | 4 |
| `gasestimate` | `gasprice` | Estimated blocks to confirm at given price | 4 |

---

## E.22 — Missing UI/UX Features (Etherscan Standard)

| Feature | Description | Phase |
|---|---|---|
| **Copy to clipboard** | Every address, hash, hex value has a copy icon | 1 |
| **Relative + absolute time toggle** | Click timestamp to switch between "19 hrs ago" and "Jun-25-2026 02:10:23 AM +UTC" | 1 |
| **Address truncation with tooltip** | `0xd8dA...96045` — hover shows full address | 1 |
| **Transaction status badge** | ✅ Success / ❌ Failed / ⏳ Pending with color + icon | 1 |
| **Block confirmations badge** | "25,351,208 Block Confirmations" on tx detail | 1 |
| **Breadcrumb navigation** | Home > Blocks > #25391560 on every page | 1 |
| **Pagination controls** | First / ← / Page N of M / → / Last on every list | 1 |
| **Download Page Data** | CSV download button on every list page | 2 |
| **"More Details" expandable row** | Click to show advanced tx fields | 1 |
| **Social share buttons** | Twitter / Medium / Facebook / Reddit links on tx/block pages | 4 |
| **Back to Top button** | Fixed bottom-right on all pages | 1 |
| **API badge** | `</>API` button on all list pages linking to API docs | 4 |
| **Sponsored / Ads placeholder** | Sponsored row in tx/block lists | (optional) |
| **Mobile responsive** | All pages 375px+ (already in Phase 4 plan) | 4 |
| **Dark mode** | Already in plan | 1 |
| **Keyboard shortcut** | `/` to focus search | 1 |
| **"View on Blockscan" link** | Link to multichain explorer | (optional) |
| **Block/Tx not found page** | Friendly 404 with checklist | 1 |
| **ENS / name resolution** | Resolve `.sov` or `.cosmos` names (if name service deployed) | 4 |
| **QR code modal** | For any address on the page | 2 |
| **Address tags system** | Protocol-defined address labels shown site-wide | 2 |
| **Tx input data decoder** | Paste ABI → decode any tx input | 3 |
| **Unit converter** | SOV ↔ uSOV ↔ USD inline converter | 2 |

---

## E.23 — Updated Route Count

**Original plan: 61 routes**  
**Added in Appendix E: 37 new routes** (22 chart sub-pages + 15 new top-level pages)  
**New total: 98 routes**

### Complete New Routes Added by Appendix E

| Route | Description |
|---|---|
| `/gastracker` | Gas price tracker with 3 tiers + heatmap |
| `/tokens/:addr` | Individual ERC-20 / CW-20 token detail page |
| `/nfts` | Top NFT collections leaderboard |
| `/charts` | Charts & statistics hub |
| `/charts/tx` | Daily transaction count chart |
| `/charts/tx-fee` | Average transaction fee chart |
| `/charts/active-addresses` | Daily active addresses chart |
| `/charts/new-addresses` | Daily new addresses chart |
| `/charts/unique-addresses` | Cumulative unique addresses chart |
| `/charts/gas-price` | Average gas price chart |
| `/charts/gas-used` | Total gas used per day chart |
| `/charts/block-size` | Average block size chart |
| `/charts/block-time` | Average block time chart |
| `/charts/tps` | TPS chart |
| `/charts/block-count` | Blocks per day chart |
| `/charts/burnt-fees` | Daily burnt fees chart |
| `/charts/price` | SOV daily price chart |
| `/charts/market-cap` | Market capitalization chart |
| `/charts/validator-count` | Active validator count chart |
| `/charts/staking-ratio` | Staking ratio chart |
| `/charts/contracts-deployed` | Contracts deployed per day chart |
| `/charts/contracts-verified` | Contracts verified per day chart |
| `/charts/token-transfers` | Daily token transfers chart |
| `/charts/nft-transfers` | Daily NFT transfers chart |
| `/charts/bridge-volume` | Daily bridge volume chart |
| `/charts/ibc-volume` | Daily IBC volume chart |
| `/stat/supply` | SOV total supply + distribution |
| `/accounts` | Top addresses by SOV balance |
| `/contracts/verified` | Recently verified contracts list |
| `/contracts/verified/:type` | Verified contracts by type filter |
| `/txs/internal` | All internal transactions |
| `/token-approvals` | ERC-20 approval tracker |
| `/address/:addr/tokencheck` | Token approval checker for address |
| `/label/:slug` | Addresses by protocol label |
| `/exportData` | Bulk data export |
| `/myaccount` | User account dashboard |
| `/pushnotification` | Watchlist + notifications setup |

---

## E.24 — Updated Feature Comparison Matrix (additions only)

Add these rows to the existing Appendix C matrix:

| Feature | Etherscan | **SL1 Explorer (updated)** |
|---|---|---|
| Global header: token price + gas always visible | ✅ | ✅ (E.1) |
| Home stats panel: market cap, median gas, finalized block | ✅ | ✅ (E.2) |
| Home: tx history 14D sparkline | ✅ | ✅ (E.2) |
| Blocks list: network utilization banner | ✅ | ✅ (E.3) |
| Blocks list: burnt fees column | ✅ | ✅ (E.3) |
| Blocks list: gas used progress bar | ✅ | ✅ (E.3) |
| Block detail: all hash fields (parent, state root, extra data) | ✅ | ✅ (E.4) |
| Block detail: Consensus Info tab | ✅ | ✅ (E.4) |
| Tx detail: action summary banner | ✅ | ✅ (E.6) |
| Tx detail: block confirmation count badge | ✅ | ✅ (E.6) |
| Tx detail: gas limit & usage breakdown | ✅ | ✅ (E.6) |
| Tx detail: burnt fees + savings fees | ✅ | ✅ (E.6) |
| Tx detail: txn type + nonce + position | ✅ | ✅ (E.6) |
| Tx detail: State Changes tab | ✅ | ✅ (E.6) |
| Tx detail: private note (login) | ✅ | ✅ (E.6) |
| Tx list: method badge labels | ✅ | ✅ (E.5) |
| Tx list: IN/OUT directional badges | ✅ | ✅ (E.5) |
| Tx list: hide low value toggle | ✅ | ✅ (E.5) |
| Tx list: summary banner (4 stat cards) | ✅ | ✅ (E.5) |
| Address: IN/OUT badges on tx rows | ✅ | ✅ (E.7) |
| Address: funded-by field | ✅ | ✅ (E.7) |
| Address: Analytics tab | ✅ | ✅ (E.7) |
| Address: Assets tab | ✅ | ✅ (E.7) |
| Address: Cards tab (NFT gallery) | ✅ | ✅ (E.7) |
| Address: private name tags (login) | ✅ | ✅ (E.7) |
| Address: watchlist star (login) | ✅ | ✅ (E.7) |
| Address: advanced filter | ✅ | ✅ (E.7) |
| Contract: Events tab | ✅ | ✅ (E.8) |
| Contract: Info tab (creator + date) | ✅ | ✅ (E.8) |
| Contract: Read/Write as Proxy tabs | ✅ | ✅ (E.8) |
| Contract: Past Implementations tab | ✅ | ✅ (E.8) |
| Contract: Audit Report badge | ✅ | ✅ (E.8) |
| Contract: Proxy Checker button | ✅ | ✅ (E.8) |
| Contract: Similar Contracts button | ✅ | ✅ (E.8) |
| Token tracker: Onchain Market Cap column | ✅ | ✅ (E.9) |
| Token tracker: holder growth % 24H | ✅ | ✅ (E.9) |
| Token detail page (standalone) | ✅ | ✅ (E.10) |
| Pending tx: nonce + last-seen + gas breakdown | ✅ | ✅ (E.11) |
| Gas tracker page (3 tiers + heatmap + action costs) | ✅ | ✅ (E.12) |
| NFT top collections leaderboard | ✅ | ✅ (E.13) |
| Charts hub page | ✅ | ✅ (E.14) |
| 22 individual chart sub-pages | ✅ | ✅ (E.14) |
| Supply page (distribution breakdown) | ✅ | ✅ (E.15) |
| Top accounts leaderboard | ✅ | ✅ (E.18) |
| Verified contracts list | ✅ | ✅ (E.19) |
| User account system (watchlist, notes, API keys) | ✅ | ✅ (E.20) |
| Complete Etherscan REST API (all modules) | ✅ | ✅ (E.21) |
| Copy-to-clipboard on every value | ✅ | ✅ (E.22) |
| Relative/absolute timestamp toggle | ✅ | ✅ (E.22) |
| Block/Tx not found page | ✅ | ✅ (E.22) |
| Download Page Data (CSV) on every list | ✅ | ✅ (E.22) |

---

## E.25 — Sovereign L1 Unique Features (Beyond Etherscan)

These are features the Sovereign L1 Explorer will have that Etherscan does NOT — our differentiators:

| Feature | Description |
|---|---|
| ★ Cosmos + EVM unified address page | One page covers both runtimes |
| ★ Slot-based validator grid (30 equal-power slots) | Not stake-weighted |
| ★ x/certification: attestation + degraded mode badge | Custom module |
| ★ x/oracle: commit/reveal round detail | Custom module |
| ★ Oracle staleness animated state machine | Custom module |
| ★ x/milestone: state machine + deadline pause timeline | Custom module |
| ★ x/settlement: Ed25519 domain separator inspector | Custom module |
| ★ BSC bridge: lock → quorum → mint end-to-end view | Custom bridge |
| ★ Bridge supply invariant live gauge | LockBox invariant |
| ★ Relayer promotion ladder visualisation | Slot-delay submitter |
| ★ Bridge circuit-breaker history | LockBox governance |
| ★ Nonce bitmap registry viewer | LockBox replay protection |
| ★ Confirmation tier badge (15 vs 50 blocks) | Tiered relayer security |
| ★ gRPC server-streaming (no WebSocket polling) | Architecture advantage |
| ★ CosmWasm execute/query forms with gas simulation | Celatone-class |
| ★ CW-1155 multi-token detail view | Rarely supported |
| ★ ERC-4626 vault auto-detection | Not on Etherscan |
| ★ IBC stuck packet alerts | Not on Etherscan |
| ★ Live CometBFT consensus round visualiser | Not on Etherscan |
| ★ Testnet faucet built-in | Not on Etherscan |

---

**Document Version:** 5.0  
**Date Updated:** 2026-06-25  
**Scope change:** Appendix E added — full Etherscan parity feature catalog. Original plan (Phases 1–4 + Appendices A–D) unchanged.  
**New total routes:** 98 (61 original + 37 new from E.23)  
**Target standard:** 100% Etherscan feature parity + Sovereign L1 unique differentiators


---
---

# Appendix F — Etherscan Parity: Second Pass (Remaining Gaps)

> **Purpose:** After a full systematic audit of every Etherscan navigation menu item, dropdown, page, sub-section, data field, and developer tool — this appendix documents everything still missing after Appendix E. Nothing from any prior section is removed.

---

## F.1 — Etherscan Full Navigation Audit

Etherscan has 6 top-level nav groups. Every item is checked against the plan.

### F.1.1 — Blockchain Dropdown

| Menu Item | Route in Plan | Status | Action |
|---|---|---|---|
| Transactions | `/txs` | ✅ covered | — |
| Pending Transactions | `/txs/pending` | ✅ covered | — |
| Contract Internal Transactions | `/txs/internal` | ✅ in E.23 | — |
| Beacon Withdrawals (staking exits) | — | ❌ missing | Add `/txs/withdrawals` |
| View Blocks | `/blocks` | ✅ covered | — |
| Forked Blocks (Reorgs) | — | ❌ missing | Add `/blocks/reorgs` |
| Uncle Blocks | — | N/A (CometBFT instant finality — no uncles) | Document as N/A |
| Top Accounts | `/accounts` | ✅ in E.18 | — |

### F.1.2 — Tokens Dropdown

| Menu Item | Route in Plan | Status | Action |
|---|---|---|---|
| ERC-20 Top Tokens | `/evm/tokens` | ✅ covered | — |
| View ERC-20 Transfers | — | ❌ missing | Add `/txs/erc20` |
| ERC-721 Top Tokens | `/evm/nfts` | ✅ covered | — |
| View ERC-721 Transfers | — | ❌ missing | Add `/txs/erc721` |
| ERC-1155 Top Tokens | `/evm/tokens/:addr/multi` (only detail) | ❌ missing list | Add `/txs/erc1155` + `/evm/tokens/multi` list |
| View ERC-1155 Transfers | — | ❌ missing | Add `/txs/erc1155` |

### F.1.3 — NFTs Dropdown

| Menu Item | Route in Plan | Status | Action |
|---|---|---|---|
| Top NFTs (by volume) | `/nfts` | ✅ in E.13 | — |
| Top Mints | — | ❌ missing | Add `/nfts/top-mints` |
| Latest Trades | — | ❌ missing | Add `/nfts/latest-trades` |
| Latest Transfers | — | ❌ missing | Add `/nfts/latest-transfers` |
| Latest Mints | — | ❌ missing | Add `/nfts/latest-mints` |

### F.1.4 — Resources Dropdown

| Menu Item | Route in Plan | Status | Action |
|---|---|---|---|
| Charts & Stats | `/charts` | ✅ in E.14 | — |
| Top Statistics | — | ❌ missing | Add `/stats` (key metrics leaderboard) |
| Directory | — | ❌ missing | Add `/directory` |
| Token Approvals | `/token-approvals` | ✅ in E.23 | — |
| Vyper Online Compiler | — | ❌ missing | Add `/tools/compiler` (Solidity + CosmWasm IDE) |
| ABI / Source Encoder | — | ❌ missing | Add `/tools/abi-encoder` |
| Bytecode to Opcode Disassembler | — | ❌ missing | Add `/tools/disassembler` |
| Broadcast Raw Transaction | — | ❌ missing | Add `/tools/broadcast` |
| Unit Converter | Inline only (E.22) | ❌ no standalone page | Add `/tools/unit-converter` |
| Similar Contracts Lookup | Contract page button only | ❌ no standalone | Add `/tools/similar-contracts` |
| Contract Diff Checker | — | ❌ missing | Add `/tools/contract-diff` |

### F.1.5 — Developers Dropdown

| Menu Item | Route in Plan | Status | Action |
|---|---|---|---|
| API Plans & Pricing | — | ❌ missing | Add `/api-plans` |
| API Documentation | `/docs` | ✅ covered | — |
| Code Reader (AI-powered) | — | ❌ missing | Add to contract page + `/tools/code-reader` |
| Verify Contract | `/verify` | ✅ covered | — |
| Constructor Argument Data | — | ❌ missing | Add `/tools/constructor-args` |
| GitHub Sync (auto-verify) | — | ❌ missing | Add GitHub Sync flow to `/verify` |

### F.1.6 — Sign In / Account Menu

| Menu Item | Route in Plan | Status | Action |
|---|---|---|---|
| Watchlist | `/myaccount` | ✅ in E.20 | — |
| Label Cloud | — | ❌ missing | Add `/labelcloud` |
| Domain Names | — | ❌ missing | Add `/domains` |
| Private Name Tags | `/myaccount` | ✅ in E.20 | — |
| Token Ignore List | `/myaccount` | ✅ in E.20 | — |
| Transaction Private Notes | `/myaccount` | ✅ in E.20 | — |
| Advanced Filter (standalone) | Address page only | ❌ no standalone | Add `/txs/advanced-filter` |
| Verified Signature Lookup | — | ❌ missing | Add `/tools/verify-signature` |

---

## F.2 — Missing Standalone Routes (all new)

| Route | Etherscan Equivalent | Description | Phase |
|---|---|---|---|
| `/txs/withdrawals` | Beacon Withdrawals | Staking undelegation completions: address, amount, block, completion time | 2 |
| `/blocks/reorgs` | Forked Blocks | CometBFT: any height mismatch events; note if chain has instant finality | 2 |
| `/txs/erc20` | View ERC-20 Transfers | All ERC-20 + CW-20 transfers chain-wide: from, to, token, amount, tx hash, age | 3 |
| `/txs/erc721` | View ERC-721 Transfers | All ERC-721 + CW-721 transfers chain-wide: from, to, collection, token ID, tx hash | 3 |
| `/txs/erc1155` | View ERC-1155 Transfers | All ERC-1155 + CW-1155 transfers chain-wide: from, to, token ID, amount, tx hash | 3 |
| `/evm/tokens/multi` | ERC-1155 Top Tokens | ERC-1155 + CW-1155 collection list (missing from plan — only detail page existed) | 3 |
| `/nfts/top-mints` | Top Mints | Collections by mint activity; new NFTs minted in period; table: collection, mints, unique minters | 3 |
| `/nfts/latest-trades` | Latest Trades | Real-time NFT trade feed: collection, token ID, buyer, seller, price in SOV + USD | 3 |
| `/nfts/latest-transfers` | Latest Transfers | Real-time NFT transfer feed: collection, token ID, from, to, block, age | 3 |
| `/nfts/latest-mints` | Latest Mints | Real-time mint feed: collection, token ID, minter address, tx hash, age | 3 |
| `/stats` | Top Statistics | Key metrics leaderboard: top gas spenders, top senders, top receivers, top token holders | 2 |
| `/directory` | Directory | Curated project directory: DeFi, NFT, Bridge, Oracle, Governance — each entry has logo, description, contract links | 3 |
| `/labelcloud` | Label Cloud | Visual tag cloud of all address labels; click label → `/label/:slug` | 2 |
| `/domains` | Domain Names | Name service registry (if `.sov` name service deployed); address ↔ name lookup | 4 |
| `/txs/advanced-filter` | Advanced Filter | Standalone advanced tx filter: from/to address, block range, date range, value range, method, token | 2 |
| `/tools/compiler` | Vyper Online Compiler | Browser-based Solidity compiler (Remix-lite) + CosmWasm schema validator | 3 |
| `/tools/abi-encoder` | ABI / Source Encoder | Paste ABI + function name + args → encoded calldata; also decode calldata → readable | 3 |
| `/tools/disassembler` | Bytecode Disassembler | Paste EVM bytecode hex → opcode listing with PC offsets | 3 |
| `/tools/broadcast` | Broadcast Raw Transaction | Paste signed raw tx hex → submit to EVM JSON-RPC or CometBFT RPC | 2 |
| `/tools/unit-converter` | Unit Converter | SOV ↔ uSOV ↔ Wei ↔ Gwei ↔ USD conversion with live price | 1 |
| `/tools/similar-contracts` | Similar Contracts Lookup | Input bytecode hash or contract address → list all contracts with same bytecode | 3 |
| `/tools/contract-diff` | Contract Diff Checker | Compare two contract addresses side-by-side: source diff highlighted | 3 |
| `/tools/constructor-args` | Constructor Argument Data | Decode ABI-encoded constructor arguments given contract address + ABI | 3 |
| `/tools/code-reader` | Code Reader (AI) | AI-powered contract explanation: paste address → plain-English summary of what contract does | 4 |
| `/tools/verify-signature` | Verified Signature Lookup | Verify EIP-191 / Cosmos ADR-036 signed message: input message + signature + address → valid/invalid | 3 |
| `/api-plans` | API Plans & Pricing | API rate tier table: Free / Standard / Pro limits; API key generation | 4 |
| `/gas/guzzlers` | Gas Guzzlers | Top 25 contracts by total gas consumed (last 24H / 7D / 30D) | 2 |
| `/gas/spenders` | Gas Spenders | Top 25 addresses by total gas fees paid (last 24H / 7D / 30D) | 2 |

---

## F.3 — Transaction Detail Page: Missing Sub-sections

These are sections within `/txs/:hash` that Etherscan shows but are not yet fully specified.

### F.3.1 — Token Transfers Section (within tx detail)

Etherscan shows all token movements caused by a transaction as a dedicated section above the main fields.

| Field | Description | Phase |
|---|---|---|
| Section header | "ERC-20 Tokens Transferred: N" | 3 |
| Token logo | Small logo icon | 3 |
| Transfer direction | From address → To address | 3 |
| Token amount | Amount + symbol | 3 |
| USD value | Amount × price at tx time | 3 |
| Token link | Links to `/tokens/:addr` | 3 |
| NFT Transfers sub-section | "NFT Tokens Transferred: N" with image thumbnails | 3 |
| NFT token ID | Token ID + collection name | 3 |

### F.3.2 — Internal Transactions Section (within tx detail)

| Field | Description | Phase |
|---|---|---|
| Section header | "Internal Transactions" with count | 3 |
| Call type | CALL / DELEGATECALL / STATICCALL / CREATE | 3 |
| From (internal caller) | Contract address + label | 3 |
| To (internal callee) | Contract address + label | 3 |
| Value | ETH/SOV sent in internal call | 3 |
| Gas limit | Gas allocated | 3 |
| Call tree | Indented visual call tree (depth by indentation) | 3 |

### F.3.3 — Event Logs Section (within tx detail)

| Field | Description | Phase |
|---|---|---|
| Section header | "Logs (N)" | 3 |
| Log index # | Sequential index | 3 |
| Contract address | Emitting contract + label | 3 |
| Event name | Decoded event name (e.g. `Transfer`, `Approval`) | 3 |
| Topic 0 | keccak256 of event signature | 3 |
| Topics 1–3 | ABI-decoded indexed parameters | 3 |
| Data | ABI-decoded non-indexed parameters | 3 |
| Raw toggle | Show hex vs decoded toggle | 3 |

### F.3.4 — Input Data Section (within tx detail)

| Feature | Description | Phase |
|---|---|---|
| Raw hex view | Full calldata in hex | 1 |
| Decoded view | ABI-decoded: function name + each parameter labeled | 2 |
| Original UTF-8 | Attempt UTF-8 decode of calldata (memo/text txs) | 1 |
| Raw ↔ Decoded toggle | Toggle button between views | 1 |
| "Decode Input Data" button | User pastes custom ABI to decode unknown calldata | 3 |

### F.3.5 — Comments Section (within tx detail)

| Feature | Description | Phase |
|---|---|---|
| Public comments | Community can comment on any tx (login required) | 4 |
| Comment count badge | Shows (N) on tx page header | 4 |
| Upvote/downvote | Comment voting | 4 |

---

## F.4 — Contract Page: Missing Fields

### F.4.1 — Contract Code Panel: Missing Fields

These fields appear in the "Contract" tab source code section:

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Bytecode section | Raw deployed bytecode in hex | ✅ Show raw bytecode under expandable section | 3 |
| Opcodes section | EVM opcodes disassembled from bytecode | ✅ PUSH1, ADD, etc. — link to `/tools/disassembler` | 3 |
| Constructor Arguments | ABI-decoded constructor args used at deployment | ✅ Decoded from creation tx calldata | 3 |
| Gas Used at Deployment | Gas consumed during contract creation tx | ✅ From creation tx receipt | 2 |
| GitHub Sync status | Auto-verified via GitHub repo link | ✅ Badge: GitHub Sync Verified / Manual Verified | 4 |
| Code Reader button | AI summary of contract | ✅ "Explain with AI" → links to `/tools/code-reader` | 4 |

### F.4.2 — Read Contract Panel: Missing Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Connect Wallet button | MetaMask connect for `eth_call` with sender | ✅ (already in plan) | 3 |
| Function category | Grouped by: `view` / `pure` / `payable` | ✅ same grouping | 3 |
| Return type label | Shows return type (uint256, address, bool) | ✅ | 3 |
| Response formatting | Numbers formatted with commas; addresses linked | ✅ | 3 |
| "Query" button per function | Individual query without page reload | ✅ | 3 |

### F.4.3 — Write Contract Panel: Missing Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| `payableAmount` field | Extra ETH to send with payable function | SOV amount field for payable functions | 3 |
| Gas limit override | Manual gas limit input | ✅ | 3 |
| Transaction preview | "You are about to call X with args Y" summary | ✅ | 3 |
| MetaMask popup preview | Shows decoded function call in MetaMask | Native behavior | 3 |
| Post-tx result | Shows tx hash + link immediately after broadcast | ✅ | 3 |

---

## F.5 — Token Detail Page: Missing Fields (extends E.10)

### F.5.1 — Token Profile Section

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Website link | Project homepage URL | ✅ from token metadata | 2 |
| Twitter / X link | Social link | ✅ | 2 |
| Discord link | Social link | ✅ | 2 |
| Reddit link | Social link | ✅ | 2 |
| Telegram link | Social link | ✅ | 2 |
| GitHub link | Repo link | ✅ | 2 |
| Whitepaper link | PDF or external link | ✅ | 2 |
| CoinGecko link | Price data source | ✅ external link | 3 |
| CoinMarketCap link | Price data source | ✅ external link | 3 |
| Token logo | 32×32 image from token metadata | ✅ | 2 |
| Update Token Info button | Link to logo/info submission form | ✅ `/tokens/:addr/submit` | 3 |
| Watch Token button | Add to watchlist (login required) | ✅ | 4 |
| Reputation badge | OK / Neutral / Caution / Spam | ✅ manual tagging by admin | 2 |
| Verified badge | ✅ if project verified identity | ✅ | 3 |

### F.5.2 — Price & Market Section

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Price USD | $1.0001 (+0.01%) | Oracle feed price or external | 2 |
| Price SOV equivalent | (ETH equivalent on Etherscan) | SOV pair price | 3 |
| Fully Diluted Market Cap | Max supply × price | ✅ | 2 |
| Circulating Supply | from `totalSupply()` | ✅ | 2 |
| Holders | Count + % change 24H | ✅ | 2 |
| Total Transfers | Cumulative all-time | ✅ | 2 |
| Volume (24H) | Trading volume USD | From DEX events / transfer volume | 3 |
| Price chart | 7D line chart | ✅ from oracle or price feed | 3 |

---

## F.6 — Gas Tracker Page: Missing Sub-pages (extends E.12)

### F.6.1 — Gas Guzzlers Page (`/gas/guzzlers`)

Top 25 smart contracts consuming the most gas.

| Column | Description | Phase |
|---|---|---|
| # | Rank | 2 |
| Contract | Address + name label | 2 |
| Percentage | % of total gas used | 2 |
| Total Gas Used | Absolute gas in period | 2 |
| Transactions | Tx count in period | 2 |
| Time filter | 24H / 3D / 7D tabs | 2 |

### F.6.2 — Gas Spenders Page (`/gas/spenders`)

Top 25 addresses paying the most in gas fees.

| Column | Description | Phase |
|---|---|---|
| # | Rank | 2 |
| Address | Address + name label | 2 |
| Percentage | % of total fees paid | 2 |
| Total ETH Spent | Total SOV in fees | 2 |
| Transactions | Tx count | 2 |
| Time filter | 24H / 3D / 7D tabs | 2 |

---

## F.7 — Address Page: Missing Fields (extends E.7)

### F.7.1 — Vesting Account Fields

Cosmos SDK supports vesting accounts. Etherscan has no equivalent, but the plan must cover it.

| Field | Description | Phase |
|---|---|---|
| Vesting type badge | ContinuousVesting / DelayedVesting / PeriodicVesting | 2 |
| Original vesting amount | Total amount that was locked at creation | 2 |
| Start time | Vesting start block / timestamp | 2 |
| End time | Vesting end block / timestamp | 2 |
| Vested so far | Amount already unlocked | 2 |
| Still vesting | Amount still locked | 2 |
| Vesting schedule bar | Visual progress bar: vested / total | 2 |

### F.7.2 — Address Overview: Additional Missing Fields

| Field | Etherscan | Sovereign L1 Equivalent | Phase |
|---|---|---|---|
| Balance history sparkline | Mini 7D balance chart | ✅ 7D balance from TimescaleDB | 3 |
| Token Approvals button | "Check Token Approvals" link | ✅ links to `/address/:addr/tokencheck` | 3 |
| First seen block | Block height of first tx | ✅ | 1 |
| Last active block | Block height of last tx | ✅ | 1 |
| Total transactions sent | Outgoing tx count | ✅ | 1 |
| Total transactions received | Incoming tx count | ✅ | 1 |
| Nonce / Sequence | Current account sequence | ✅ (Cosmos sequence + EVM nonce) | 1 |

### F.7.3 — Advanced Filter Panel (on address tx tab)

When user clicks "Advanced" on address tx tab, a filter panel expands:

| Filter | Description | Phase |
|---|---|---|
| Transaction Type | All / Send / Receive / Internal / Token Transfer | 2 |
| From Address | Filter txs from specific sender | 2 |
| To Address | Filter txs to specific recipient | 2 |
| Age | From date → To date picker | 2 |
| Block Range | From block → To block | 2 |
| Value Range | Min SOV → Max SOV | 2 |
| Method | Filter by Msg type / function name | 2 |
| Token | Filter by specific token contract | 2 |
| Apply / Reset buttons | ✅ | 2 |

---

## F.8 — Top Statistics Page (`/stats`)

Standalone key-metrics leaderboard (separate from `/charts` which shows historical charts).

### F.8.1 — Sections

| Section | Content | Phase |
|---|---|---|
| **Top Validators** | By blocks proposed (24H); by uptime (30D) | 2 |
| **Top Gas Consumers** | Contracts: highest gas used (24H) | 2 |
| **Top Tx Senders** | Addresses: most txs sent (24H) | 2 |
| **Top SOV Holders** | Richest 25 addresses by native balance | 2 |
| **Top Token Holders** | By token (select dropdown) | 2 |
| **Top Contract Callers** | Addresses: most contract calls (24H) | 2 |
| **Chain Milestones** | Historical: Tx #1M, #10M, #100M — block height + date | 2 |

### F.8.2 — Stat Cards Panel (top of page)

Same as home stats panel but always-live on `/stats`:

| Metric | Phase |
|---|---|
| Total Transactions (all-time) | 1 |
| Total Addresses | 1 |
| Avg Block Time (7D) | 1 |
| Avg TPS (24H) | 1 |
| Total Fees Collected (all-time) in SOV | 1 |
| Total Fees Burned (all-time) in SOV | 1 |

---

## F.9 — Directory Page (`/directory`)

Etherscan's "Directory" is a curated list of projects using the chain.

| Field | Description | Phase |
|---|---|---|
| Project name + logo | e.g. "Sovereign Bridge", "Oracle Network" | 3 |
| Category | DeFi / NFT / Bridge / Oracle / Governance / Infrastructure | 3 |
| Description | 1–2 sentence summary | 3 |
| Contract links | All deployed contract addresses | 3 |
| Website + social links | External links | 3 |
| Verified badge | Admin-verified project | 3 |
| Submit project button | Protocol teams can submit for review | 3 |

---

## F.10 — Label Cloud Page (`/labelcloud`)

Visual overview of all address label categories used on the chain.

| Element | Description | Phase |
|---|---|---|
| Tag cloud | Each label as a clickable bubble; size = number of addresses with that label | 2 |
| Categories | DeFi / Bridge / Oracle / Validator / Treasury / Exchange / NFT | 2 |
| Click → `/label/:slug` | Shows all addresses with that label | 2 |
| Count per label | "(N addresses)" tooltip on hover | 2 |

---

## F.11 — Advanced Filter Page (`/txs/advanced-filter`)

Standalone advanced filter as a full page (in addition to inline filter on address page).

| Filter | Description | Phase |
|---|---|---|
| From Address | Sender bech32 or 0x | 2 |
| To Address | Recipient bech32 or 0x | 2 |
| Date Range | From → To date picker | 2 |
| Block Range | From block → To block | 2 |
| Value Range | Min → Max SOV | 2 |
| Tx Type | Cosmos / EVM / CosmWasm / Bridge | 2 |
| Method / Msg Type | Specific function or Msg type | 2 |
| Token Filter | Specific ERC-20 / CW-20 contract | 2 |
| Status | Success / Failed | 2 |
| Results table | Filtered txs shown in paginated table | 2 |
| Download CSV | Export filtered results | 2 |

---

## F.12 — Tools Hub Page (`/tools`)

Etherscan's developer tools are accessible from the Resources dropdown. Add a `/tools` hub page that links to all tools:

| Tool | Route | Description | Phase |
|---|---|---|---|
| Unit Converter | `/tools/unit-converter` | SOV ↔ uSOV ↔ Gwei ↔ Wei ↔ USD | 1 |
| Broadcast Raw Transaction | `/tools/broadcast` | Submit signed tx hex to chain | 2 |
| ABI Encoder / Decoder | `/tools/abi-encoder` | Encode function calls; decode calldata | 3 |
| Bytecode Disassembler | `/tools/disassembler` | EVM bytecode → opcode listing | 3 |
| Similar Contracts | `/tools/similar-contracts` | Find contracts with same bytecode | 3 |
| Contract Diff Checker | `/tools/contract-diff` | Side-by-side source diff of two contracts | 3 |
| Constructor Args Decoder | `/tools/constructor-args` | Decode creation calldata | 3 |
| Verified Signature | `/tools/verify-signature` | Verify EIP-191 / Cosmos ADR-036 sig | 3 |
| Online Compiler | `/tools/compiler` | Browser Solidity compiler (Remix-lite) | 3 |
| AI Code Reader | `/tools/code-reader` | AI plain-English contract summary | 4 |

---

## F.13 — Broadcast Raw Transaction Page (`/tools/broadcast`)

| Field | Description | Phase |
|---|---|---|
| Runtime selector | EVM (hex) / Cosmos (base64 JSON) | 2 |
| Input textarea | Paste signed raw tx | 2 |
| Decode first | Button: decode and show fields before submitting | 2 |
| Submit button | Sends to EVM `eth_sendRawTransaction` or CometBFT `/broadcast_tx_sync` | 2 |
| Result | Tx hash + link; or error message decoded | 2 |

---

## F.14 — Verified Signature Lookup (`/tools/verify-signature`)

| Field | Description | Phase |
|---|---|---|
| Runtime selector | EVM (EIP-191) / Cosmos (ADR-036) | 3 |
| Address | Expected signer address | 3 |
| Message | Original message text | 3 |
| Signature | Hex signature | 3 |
| Verify button | Runs ecrecover (EVM) or tendermint verify (Cosmos) | 3 |
| Result | ✅ Valid — signed by address / ❌ Invalid | 3 |

---

## F.15 — API Plans Page (`/api-plans`)

| Tier | Rate Limit | Features | Phase |
|---|---|---|---|
| Free | 5 calls/sec, 10k/day | All read endpoints | 4 |
| Standard | 20 calls/sec, 100k/day | Read + bulk export | 4 |
| Pro | 50 calls/sec, unlimited | All endpoints + priority support | 4 |
| Generate API Key | Button → creates key, shown once | 4 | |
| API Key management | List keys, revoke, rename | `/myaccount` | 4 |

---

## F.16 — REST API: Missing `proxy` Module

Etherscan exposes direct Ethereum JSON-RPC calls through the REST API as the `proxy` module. Add to the plan:

### Module: `proxy`

| Action | Equivalent JSON-RPC | Returns | Phase |
|---|---|---|---|
| `eth_blockNumber` | `eth_blockNumber` | Latest block number in hex | 4 |
| `eth_getBlockByNumber` | `eth_getBlockByNumber` | Block data (full or hashes) | 4 |
| `eth_getTransactionByHash` | `eth_getTransactionByHash` | Transaction data | 4 |
| `eth_getTransactionReceipt` | `eth_getTransactionReceipt` | Receipt with logs | 4 |
| `eth_call` | `eth_call` | Contract read result | 4 |
| `eth_estimateGas` | `eth_estimateGas` | Gas estimate | 4 |
| `eth_gasPrice` | `eth_gasPrice` | Current gas price | 4 |
| `eth_getBalance` | `eth_getBalance` | Address balance in wei | 4 |
| `eth_getCode` | `eth_getCode` | Contract bytecode | 4 |
| `eth_sendRawTransaction` | `eth_sendRawTransaction` | Submit signed tx | 4 |
| `eth_getLogs` | `eth_getLogs` | Event logs by filter | 4 |

> **Implementation note:** These proxy endpoints forward directly to the EVM JSON-RPC. The explorer API layer adds authentication (API key), rate limiting, and CORS handling. No additional indexing is needed.

---

## F.17 — Missing NFT Sub-pages: Full Field Specs

### F.17.1 — Top Mints (`/nfts/top-mints`)

| Column | Description | Phase |
|---|---|---|
| # | Rank | 3 |
| Collection | Name + logo + type badge | 3 |
| Mints (period) | Number of NFTs minted | 3 |
| Unique Minters | Distinct minting addresses | 3 |
| Max Price | Highest mint price paid | 3 |
| Avg Price | Average mint price | 3 |
| Total Volume | Total SOV spent minting | 3 |
| Time filter | 1H / 6H / 12H / 24H / 7D tabs | 3 |

### F.17.2 — Latest Trades (`/nfts/latest-trades`)

| Column | Description | Phase |
|---|---|---|
| Tx Hash | Linked | 3 |
| Age | Timestamp relative | 3 |
| Collection | Name + logo | 3 |
| Token ID | Linked to NFT detail | 3 |
| Type | Sale / Auction / Offer | 3 |
| Price | SOV + USD | 3 |
| Buyer | Address + label | 3 |
| Seller | Address + label | 3 |

### F.17.3 — Latest Transfers (`/nfts/latest-transfers`)

| Column | Description | Phase |
|---|---|---|
| Tx Hash | Linked | 3 |
| Age | Timestamp relative | 3 |
| Collection | Name | 3 |
| Token ID | Linked | 3 |
| From | Address + label | 3 |
| To | Address + label | 3 |
| Quantity | (ERC-1155: amount; ERC-721: always 1) | 3 |

### F.17.4 — Latest Mints (`/nfts/latest-mints`)

| Column | Description | Phase |
|---|---|---|
| Tx Hash | Linked | 3 |
| Age | Timestamp | 3 |
| Collection | Name + logo | 3 |
| Token ID | Linked | 3 |
| Minter | Address + label | 3 |
| Price Paid | SOV + USD | 3 |

---

## F.18 — Token Transfers Chain-wide Pages: Full Field Specs

### F.18.1 — ERC-20 Transfers (`/txs/erc20`)

All ERC-20 + CW-20 transfers across the entire chain in real-time.

| Column | Description | Phase |
|---|---|---|
| Tx Hash | Linked | 3 |
| Block | Block height | 3 |
| Age | Relative timestamp | 3 |
| From | Sender address + label | 3 |
| → | Direction arrow | 3 |
| To | Recipient address + label | 3 |
| Amount | Token amount + symbol | 3 |
| Token | Token logo + name linked to `/tokens/:addr` | 3 |
| Token filter dropdown | Filter by specific token | 3 |
| Download Page Data | CSV | 3 |

### F.18.2 — ERC-721 Transfers (`/txs/erc721`)

| Column | Description | Phase |
|---|---|---|
| Tx Hash | Linked | 3 |
| Block | Block height | 3 |
| Age | Timestamp | 3 |
| From | Sender + label | 3 |
| To | Recipient + label | 3 |
| Token ID | ID linked to NFT detail | 3 |
| Collection | Name + logo linked | 3 |

### F.18.3 — ERC-1155 Transfers (`/txs/erc1155`)

| Column | Description | Phase |
|---|---|---|
| Tx Hash | Linked | 3 |
| Block | Block height | 3 |
| Age | Timestamp | 3 |
| From | Sender + label | 3 |
| To | Recipient + label | 3 |
| Token ID | ID | 3 |
| Amount | Quantity transferred | 3 |
| Collection | Name + logo linked | 3 |

---

## F.19 — Staking Withdrawals / Undelegation Completions Page (`/txs/withdrawals`)

Etherscan's "Beacon Withdrawals" maps to Cosmos SDK unbonding completions.

| Column | Description | Phase |
|---|---|---|
| Index | Sequential withdrawal index | 2 |
| Block | Block where unbonding was processed | 2 |
| Age | Timestamp | 2 |
| Validator | Validator address + moniker | 2 |
| Address | Delegator receiving unbonded tokens | 2 |
| Amount | SOV amount returned | 2 |
| Unbonding start | Block when undelegate was initiated | 2 |
| Unbonding end | Block when tokens were released | 2 |
| Download Page Data | CSV | 2 |

---

## F.20 — ERC-1155 Top Tokens List Page (`/evm/tokens/multi`)

Currently the plan only has `/evm/tokens/:addr/multi` (single collection detail). The list page is missing.

| Column | Description | Phase |
|---|---|---|
| # | Rank | 3 |
| Token | Name + logo + symbol | 3 |
| Standard | ERC-1155 / CW-1155 badge | 3 |
| Contract | Address linked | 3 |
| Unique Token IDs | Count of distinct token IDs | 3 |
| Holders | Unique holder addresses | 3 |
| Transfers (24H) | Transfer count | 3 |
| Total Transfers | All-time | 3 |

---

## F.21 — GitHub Sync Flow (in `/verify`)

Etherscan allows developers to verify contracts by linking to a GitHub repository. Add to the existing `/verify` page:

| Step | Description | Phase |
|---|---|---|
| 1. Connect GitHub | OAuth → authorize repo access | 4 |
| 2. Select repo | Pick org/repo from list | 4 |
| 3. Select file | Pick `.sol` file path | 4 |
| 4. Map to contract | Enter deployed contract address | 4 |
| 5. Auto-compile | Explorer compiles from repo + compares bytecode | 4 |
| 6. Result | "GitHub Sync Verified" badge on contract page | 4 |
| Ongoing sync | Re-verifies automatically on new commits | 4 |

---

## F.22 — Updated Route Count (Final)

**Original plan (Phases 1–4): 61 routes**  
**Added in Appendix E: 37 routes → total 98**  
**Added in Appendix F: 30 new routes → total 128**

### New Routes Added by Appendix F

| Route | Description |
|---|---|
| `/txs/withdrawals` | Staking undelegation completions |
| `/blocks/reorgs` | Forked / reorged blocks log |
| `/txs/erc20` | Chain-wide ERC-20 + CW-20 transfers |
| `/txs/erc721` | Chain-wide ERC-721 + CW-721 transfers |
| `/txs/erc1155` | Chain-wide ERC-1155 + CW-1155 transfers |
| `/evm/tokens/multi` | ERC-1155 + CW-1155 collection list |
| `/nfts/top-mints` | Top minting collections |
| `/nfts/latest-trades` | Latest NFT trades feed |
| `/nfts/latest-transfers` | Latest NFT transfers feed |
| `/nfts/latest-mints` | Latest NFT mints feed |
| `/stats` | Top statistics leaderboard |
| `/directory` | Protocol project directory |
| `/labelcloud` | Address label tag cloud |
| `/domains` | Name service domain lookup |
| `/txs/advanced-filter` | Standalone advanced tx filter |
| `/gas/guzzlers` | Top gas-consuming contracts |
| `/gas/spenders` | Top gas-fee-paying addresses |
| `/tools` | Developer tools hub |
| `/tools/unit-converter` | SOV unit converter |
| `/tools/broadcast` | Broadcast raw transaction |
| `/tools/abi-encoder` | ABI encoder / decoder |
| `/tools/disassembler` | Bytecode to opcodes |
| `/tools/similar-contracts` | Similar bytecode lookup |
| `/tools/contract-diff` | Contract source diff |
| `/tools/constructor-args` | Constructor args decoder |
| `/tools/verify-signature` | Verify signed message |
| `/tools/compiler` | Online Solidity compiler |
| `/tools/code-reader` | AI contract explanation |
| `/api-plans` | API tier plans + key generation |
| `/tokens/:addr/submit` | Token info / logo submission form |

---

## F.23 — Final Feature Comparison: All Remaining Gaps Now Covered

| Feature | Etherscan | SL1 Explorer |
|---|---|---|
| Beacon / staking withdrawals page | ✅ | ✅ F.19 |
| Forked blocks page | ✅ | ✅ F.2 |
| ERC-20 chain-wide transfers page | ✅ | ✅ F.18 |
| ERC-721 chain-wide transfers page | ✅ | ✅ F.18 |
| ERC-1155 chain-wide transfers page | ✅ | ✅ F.18 |
| ERC-1155 top tokens list | ✅ | ✅ F.20 |
| NFT Top Mints page | ✅ | ✅ F.17 |
| NFT Latest Trades page | ✅ | ✅ F.17 |
| NFT Latest Transfers page | ✅ | ✅ F.17 |
| NFT Latest Mints page | ✅ | ✅ F.17 |
| Top Statistics page | ✅ | ✅ F.8 |
| Project Directory | ✅ | ✅ F.9 |
| Label Cloud | ✅ | ✅ F.10 |
| Domain Names page | ✅ | ✅ F.2 |
| Standalone Advanced Filter page | ✅ | ✅ F.11 |
| Gas Guzzlers page | ✅ | ✅ F.6 |
| Gas Spenders page | ✅ | ✅ F.6 |
| Tools hub page | ✅ | ✅ F.12 |
| Unit Converter (standalone) | ✅ | ✅ F.12 |
| Broadcast Raw Transaction | ✅ | ✅ F.13 |
| ABI Encoder / Decoder | ✅ | ✅ F.12 |
| Bytecode Disassembler | ✅ | ✅ F.12 |
| Similar Contracts Lookup (standalone) | ✅ | ✅ F.12 |
| Contract Diff Checker | ✅ | ✅ F.12 |
| Constructor Args Decoder | ✅ | ✅ F.12 |
| Verified Signature Lookup | ✅ | ✅ F.14 |
| Online Solidity Compiler | ✅ | ✅ F.12 |
| AI Code Reader | ✅ | ✅ F.12 |
| API Plans page | ✅ | ✅ F.15 |
| GitHub Sync verification | ✅ | ✅ F.21 |
| Token social links + whitepaper | ✅ | ✅ F.5 |
| Token reputation badge | ✅ | ✅ F.5 |
| Token logo submission form | ✅ | ✅ F.5 |
| Contract bytecode section | ✅ | ✅ F.4 |
| Contract opcodes section | ✅ | ✅ F.4 |
| Contract constructor args decoded | ✅ | ✅ F.4 |
| Contract gas used at deployment | ✅ | ✅ F.4 |
| Tx detail: Token Transfers section | ✅ | ✅ F.3 |
| Tx detail: Internal Txs section | ✅ | ✅ F.3 |
| Tx detail: Event Logs section | ✅ | ✅ F.3 |
| Tx detail: Raw ↔ Decoded input toggle | ✅ | ✅ F.3 |
| Tx detail: Comments section | ✅ | ✅ F.3 |
| Address: vesting schedule display | N/A (Cosmos-only) | ✅ F.7 |
| Address: balance history sparkline | ✅ | ✅ F.7 |
| Address: advanced filter panel | ✅ | ✅ F.7 |
| proxy API module (JSON-RPC passthrough) | ✅ | ✅ F.16 |

---

**Document Version:** 6.0  
**Date Updated:** 2026-06-25  
**Scope change:** Appendix F added — second-pass Etherscan parity audit. All prior sections unchanged.  
**Total routes:** 128 (61 original + 37 Appendix E + 30 Appendix F)  
**Status:** 100% Etherscan feature parity achieved across all pages, data fields, tools, and API modules.

