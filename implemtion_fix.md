# Sovereign L1 — Dual-Account Deployment & Etherscan/Solscan-Parity Explorer: Implementation Plan

> Source repo analyzed: `Sovereign-Layer-2-blockchain-` (local clone path referenced below as `$REPO_ROOT`, e.g. `/Users/majedurrahman/Sovereign`).
> This plan was produced by deeply reading the actual code (chain, contracts, relayer/oracle, backend, explorer, explorer-api, explorer-indexer, db schemas, scratch scripts) — not the project's own "planned-vs-implemented" claims, which overstate completeness in several places relevant to this work.
>
> **Hard constraint for every phase below: no mock data, no hardcoded/fallback values, no placeholder UI numbers. Every figure shown in the explorer must come from a real on-chain event or a real indexed row. If data isn't available yet, show an explicit empty/loading/error state — never a fake number.**

---

## 0. Context: what's actually in the repo today

Verified by direct inspection:

- `scratch/deploy_and_test.sh` funds a single "faucet" key, then deploys EVM contracts via `explorer/scripts/deploy_evm.ts` (hardcoded to `cd /Users/majedurrahman/Sovereign/explorer` — breaks on any machine/path other than the original author's Mac), then stores/instantiates CW20/CW721/CW1155 wasm from `/tmp/cw721_nft.wasm` etc.
- **The CW20/CW721/CW1155 `.wasm` binaries referenced by the script do not exist.** `contracts/cw20`, `contracts/cw721`, `contracts/cw1155` have Rust source but were never compiled to `artifacts/`. Only `constitution.wasm`, `cw_counter.wasm`, `governance.wasm`, `reserve_fund.wasm`, `treasury.wasm` exist in `artifacts/` and `artifacts/checksums.txt`.
- `deploy_evm.ts` derives its EVM signer from a well-known test mnemonic (`"test test test ... junk"`, the standard Hardhat/Anvil mnemonic) — this is the same account for every deploy, and its `RECIPIENT` constant is an EVM address with a **stale comment claiming it's a Cosmos bech32 address** (`0xe1f1a5093254b350c55514f8b9dbb40b996170c4 // cosmos1u8c62zfj2je4p324znutnka5pwvkzuxyyk63dz`) — the two account systems are conflated in the script rather than treated as genuinely separate holders.
- The explorer schema (`db/read_schema/*.sql`) has tables for bridge volume, validator uptime, oracle participation, settlements, milestones, contracts registry, verified contracts, governance votes, daily aggregates — but **no `token_transfers`, `token_holders`, `nft_ownership`, or `nft_transfers` tables** for either EVM (ERC-20/721/1155) or CosmWasm (CW20/721/1155) assets. Without these, no explorer/Etherscan-style page can show real transfer history or holder lists.
- The explorer frontend already has the right *routes* (`/evm/tokens/[addr]`, `/evm/nfts/[addr]`, `/contracts/[addr]/nfts`, `/txs/[hash]`, `/evm/txs/[hash]`, etc.) but `evm/tokens/[addr]/page.tsx` **falls back to hardcoded values** when the API call doesn't return data (`data.name || "Sovereign Stablecoin"`, `data.symbol || "sUSDT"`, `data.totalSupply || "10,000,000"`). This is exactly the fallback-data pattern the user has asked to eliminate.
- `explorer-api` exposes `/api/rest/v1/explorer/tokens/cw20/{addr}` (referenced by the frontend) but there is no corresponding backing table/query verified to return real transfer/holder data — the endpoint's response shape doesn't match what a transfer/holder list needs.
- The immediate failure the user hit (`chaind tx bank send faucet ... ERROR: Transaction failed with code 1`) has three plausible root causes visible in the script/config, in likely order of probability:
  1. The `faucet` key in the `test` keyring inside the `chain-node` container either isn't funded in genesis or has an outdated/mismatched account sequence (e.g. re-running the script against a chain that was reset or restarted without re-adding the key).
  2. Fee denom/amount mismatch — the tx uses `--fees 10000atoken`, which requires `atoken` to exist as a fee-payable denom in `app.toml`'s `minimum-gas-prices`; if that config drifted, the tx is rejected before ever reaching a broadcast-level error the script's naive txhash-grep can parse.
  3. The script's error handling only greps for `"txhash":"..."` and treats any non-zero `code` from the **first** `q tx` poll as fatal — a transient CheckTx failure (mempool, sequence race) surfaces as this exact message even when a retry would succeed.
  The plan below fixes the account/funding model at the root (Phase 1) instead of patching around the symptom.

---

## Phase 1 — Fix the dual-account funding & deployment model

**Goal:** one dedicated **EVM account** that ends up holding all ERC-20/721/1155 assets, one dedicated **Cosmos account** that ends up holding all CW20/721/1155 assets, both funded deterministically from genesis (not from a fragile runtime `bank send`), and a deploy script that is portable (no hardcoded absolute paths) and fails loudly with a real root cause instead of a bare "code 1".

1.0. **Consolidate genesis to one generator with two environment outputs (dev + prod), not six ad hoc files.** The repo currently has six genesis-related files with overlapping/unclear purpose:
   - `chain/genesis.json` — the file the devnet chain actually boots from today. Verified directly (`node` JSON check): `auth.accounts` and `bank.balances` are empty arrays here, and in `scripts/default_genesis.json` and `scripts/missing_genesis.json` too — **none of the three currently carries any real account/balance data**, so nothing of value is lost by deleting the latter two.
   - `scripts/generate_genesis.go` — a real generator+verifier that checks economic invariants from `doc/governance/genesis_parameters.md` (ADR-011): total supply = bridge-minted + escrow balance, reward bucket covers a 5-year block floor, x/vm and x/feemarket and x/erc20 params are set correctly. Confirmed by reading it directly: it already takes `--out` and `--chain-id` flags and currently sets `auth.params`/`bank.params`/`bank.denom_metadata`, but **does not yet populate `auth.accounts`/`bank.balances` at all** — that part still needs to be added. **This is the one script to keep and extend; everything else is redundant or unrelated.**
   - `scripts/default_genesis.json`, `scripts/missing_genesis.json`, `scripts/print_default_genesis.go`, `reference-kubernetes-setup/prerequsite/scripts/createGenesis.sh` — a stale/manual snapshot with no account data, a diagnostic dump of null/empty sections, a debug-only SDK-default dump, and an unrelated Hyperledger Fabric `configtxgen` script (leftover from a different blockchain stack, not Cosmos/CosmWasm). **Delete all four.**
   - **Keep exactly two generated genesis outputs, not one** (per explicit requirement: one for dev, one for prod), both produced by the *same* extended `generate_genesis.go`, never hand-edited:
     - `chain/genesis.dev.json` — generated with a `--env dev` flag: includes the two test-keyring holder accounts (`sovereign1-evm-holder`, `sovereign1-cosmos-holder` from 1.1 below) in `auth.accounts`/`bank.balances`, funded deterministically for local devnet use. This is what `docker-compose`/local scripts point at.
     - `chain/genesis.prod.json` — generated with a `--env prod` flag: same economic invariants and module params, but **no test-keyring accounts baked in at all** — `auth.accounts`/`bank.balances` for real validator/treasury allocations only, sourced from real keys generated through production key management (HSM/KMS), never the `test` keyring. This file is a template/skeleton until real production key material exists; it must fail loudly (invariant check) rather than silently boot with dev accounts if someone forgets to swap them.
   - Action: extend `generate_genesis.go` to (a) add the `--env dev|prod` flag controlling whether test-holder accounts are injected, (b) populate `auth.accounts`/`bank.balances` accordingly for each environment, (c) default `--out` to `chain/genesis.<env>.json`, then delete `scripts/default_genesis.json`, `scripts/missing_genesis.json`, `scripts/print_default_genesis.go`, and `reference-kubernetes-setup/prerequsite/scripts/createGenesis.sh`. Update `scratch/deploy_and_test.sh`/`docker-compose.yml` to point at `chain/genesis.dev.json` explicitly.

1.1. **Define the two accounts explicitly** and generate them from the *same* mnemonic-independent, chain-side keyring so both exist before any script runs:
   - `sovereign1-evm-holder` — an `eth_secp256k1` key in the `test` keyring; its `0x...` address is the one and only signer used for all EVM contract deploys and ERC-20/721/1155 mints/transfers.
   - `sovereign1-cosmos-holder` — a standard `secp256k1` key in the `test` keyring; its `cosmos1...` address is the one and only signer/owner used for all CW20/721/1155 instantiate/mint/transfer calls.
   - Do not reuse the `faucet` key as an asset holder. In the **dev** genesis (1.0), `faucet` is not even needed as a *runtime* funder anymore — both holders are funded directly in `chain/genesis.dev.json` (see 1.2). `faucet` is kept only as a fallback funding path for 1.3 below (already-running chains), never as the primary path.

1.2. **Funding is genesis-only for a fresh chain — there is exactly one authoritative funding path, not two.** `chain/genesis.dev.json` (produced by `generate_genesis.go --env dev`, per 1.0) contains `auth.accounts`/`bank.balances` entries for both `sovereign1-evm-holder` and `sovereign1-cosmos-holder` directly. No runtime `bank send` step exists in this path at all — this supersedes and removes the "is it already funded" length-check hack currently in `scratch/deploy_and_test.sh`. Do **not** add balances to any file other than through `generate_genesis.go`'s dev-env code path (all four other genesis files are deleted per 1.0; there is nothing else to edit).

1.3. **Keep a funding-check fallback for already-running chains only** (so re-running the deploy script against a devnet that's already up — not a fresh `docker compose up` — still works without re-generating genesis). In this fallback path only, `faucet` performs a one-time `bank send` to top up either holder if its balance is below a documented minimum. On non-zero `code` from that fallback send, print the full `raw_log`, the signer's current `account_number`/`sequence` (`chaind q account <addr>`), and the current `minimum-gas-prices` from the live node's `app.toml`, before exiting. Never swallow the real Cosmos SDK error string. This fallback must be clearly logged as "fallback funding path (chain already running)" so it's never confused with the genesis path in 1.2.

1.4. **Remove every hardcoded absolute path** (`/Users/majedurrahman/Sovereign/explorer`, `/tmp/cw721_nft.wasm`, etc.) from `scratch/deploy_and_test.sh` and `explorer/scripts/deploy_evm.ts`. Resolve paths relative to `$REPO_ROOT` (detected via `git rev-parse --show-toplevel` or a `REPO_ROOT` env var), so the script runs identically for any user/machine.

1.5. **Build the missing CW20/CW721/CW1155 wasm artifacts.** Add a `make build-cw-assets` (or extend the existing CosmWasm optimizer step) that compiles `contracts/cw20`, `contracts/cw721`, `contracts/cw1155` with `cosmwasm/optimizer` (the same tool already used for the other 5 contracts), writes them to `artifacts/`, and appends real sha256 checksums to `artifacts/checksums.txt`. The deploy script must reference these built artifacts, not a `/tmp/*.wasm` path that nothing produces.

1.6. **Rewrite `deploy_evm.ts` and the CW deploy steps to target the two dedicated holders**, not the well-known Hardhat mnemonic and not the `faucet` key directly:
   - EVM: `TestERC20`/`TestERC721`/`TestERC1155` (and any ERC-4626 vault) are deployed by, and mint their initial/test supply to, the EVM holder address only.
   - Cosmos: CW20 `initial_balances`, CW721 `minter`, and CW1155 mints all target the Cosmos holder address only. Remove the second/incidental recipient address currently baked into the CW20 init message unless the user wants a second test wallet — if so, make it a named, documented "test recipient" account, not an unlabeled bech32 string.

1.7. **Decide and document indexer write-path ownership now, not during implementation.** EVM log ingestion (Transfer/TransferSingle/TransferBatch events) will be owned by `explorer-indexer` — it already subscribes to chain data as its sole responsibility, whereas `backend/module/projection` is CQRS read-model projection for the application's own domain events (bridge/oracle/governance), a different concern. `backend/module/projection` must not also subscribe to raw EVM/CosmWasm logs — if it needs derived token data, it reads `explorer-indexer`'s tables, it does not re-ingest independently. This resolves the ownership question raised in Phase 2 step 2 below; no further "confirm during implementation" is needed for this specific boundary.

1.8. **Verification for this phase:** `./scratch/deploy_and_test.sh` runs end-to-end on a clean `docker compose up` with zero manual key-funding steps, and finishes by printing: EVM holder address + its ERC-20/721/1155 balances (queried live via `eth_call`/JSON-RPC, not asserted from the deploy step's own log), and Cosmos holder address + its CW20/721/1155 balances (queried live via `chaind query wasm contract-state smart`, not asserted). No number in this verification output may come from a hardcoded expectation — every value is queried back from the chain after deployment.

---

## Phase 2 — Explorer data model: real transfer & holder tracking (prerequisite for parity pages)

**Goal:** the schema and indexers needed so token/NFT detail pages can show real Etherscan/Solscan-style history — this must land before Phase 3's UI work, or the UI will have nothing real to render.

1. **New tables** (`db/read_schema/`, next migration number after `000007`):
   - `evm_token_transfers` — `tx_hash, log_index, block_height, block_time, token_address, from_address, to_address, value, token_standard('ERC20'|'ERC721'|'ERC1155'|'ERC4626'), token_id (nullable)`. ERC-4626 vaults emit standard ERC-20 `Transfer` events for the vault share token itself, plus `Deposit`/`Withdraw` events — index both: `Transfer` rows land here with `token_standard='ERC4626'`, and `Deposit`/`Withdraw` land in a new `evm_vault_events` table (`tx_hash, log_index, vault_address, underlying_asset_address, sender, owner, assets, shares, event_type('deposit'|'withdraw')`) so the vault's detail page can show real share-price/exchange-rate history instead of a fabricated APY number.
   - `evm_token_holders` — materialized/maintained balance per `(token_address, holder_address)`, updated transactionally as transfers are indexed (not recomputed by summing on every request — mirror how Etherscan keeps a live holders table).
   - `evm_nft_ownership` — current owner per `(token_address, token_id)`, plus `token_uri`, `metadata_json` fetched from the actual `tokenURI()`/`uri()` call (never fabricated placeholder metadata).
   - `cw_token_transfers`, `cw_token_holders` — same shape for CW20 `Transfer`/`Send` events, keyed by `contract_address, from, to, amount, height, tx_hash`.
   - `cw_nft_ownership`, `cw_nft_transfers` — same shape for CW721 (`transfer_nft`, `send_nft`, `mint`) and CW1155 events, keyed by `contract_address, token_id, owner`.
   - Extend `explorer.contracts` with `token_name, token_symbol, decimals, total_supply` columns populated from the actual contract query at indexing time (`name()`/`symbol()`/`decimals()`/`totalSupply()` for EVM; CW20 `token_info` query; CW721 `contract_info` query) — this is what replaces the frontend's hardcoded `"Sovereign Stablecoin"`/`"sUSDT"` fallback.
2. **Indexer changes**, owned exclusively by `explorer-indexer` per the decision in Phase 1.7 (`backend/module/projection` reads these tables, it does not also ingest raw chain logs — this avoids double-counting by construction, not by a runtime check):
   - Subscribe to EVM `Transfer` (ERC-20/721) and `TransferSingle`/`TransferBatch` (ERC-1155) logs via the JSON-RPC log filter/websocket, and to CosmWasm wasm events (`wasm-transfer`, `wasm-mint`, etc.) via CometBFT's event subscription — write rows into the tables above inside the same transaction as the block/tx row so there is never a partial state.
   - On first sighting of a new contract address, do a synchronous contract-metadata fetch (name/symbol/decimals/totalSupply, or CW20/CW721 info query) and populate `explorer.contracts`. If the metadata call fails, store the row with `metadata_status = 'pending'` and retry on a backoff — do not write a guessed name.
   - Apply the same "swallowed NATS failure" lesson identified in the earlier architecture review: any failure to publish/index a transfer must be retried until it succeeds or explicitly surfaced as a stuck/backfill item visible in an internal ops table — never silently dropped.
3. **API endpoints** (`explorer-api`), all backed by the tables above, no synthetic data:
   - `GET /api/rest/v1/explorer/tokens/evm/{addr}` → real name/symbol/decimals/totalSupply + holder count + 24h transfer count.
   - `GET /api/rest/v1/explorer/tokens/evm/{addr}/transfers?cursor=` → paginated real transfer log (cursor-based, per the project's own ADR-012 pagination convention).
   - `GET /api/rest/v1/explorer/tokens/evm/{addr}/holders?cursor=` → paginated real holder list sorted by balance desc, with `% of supply` computed from the real `total_supply`.
   - `GET /api/rest/v1/explorer/nfts/evm/{addr}/{tokenId}` → real current owner, real `tokenURI` metadata, real transfer history for that specific token id.
   - `GET /api/rest/v1/explorer/vaults/evm/{addr}` → real underlying asset address, total assets, total shares, share price (`totalAssets/totalShares`, computed from live state, not assumed 1:1), and paginated `deposit`/`withdraw` history from `evm_vault_events`.
   - Mirrored `.../tokens/cw20/{addr}`, `.../nfts/cw721/{addr}/{tokenId}` etc. for the Cosmos side. CW1155 uses the same shape as CW721 but keyed by `(contract_address, token_id, owner)` with a `balance` instead of single ownership, since CW1155/ERC-1155 are semi-fungible (multiple owners can hold the same token id) — the holders/owner list for these must reflect that, not assume one owner per token id.
   - `GET /api/rest/v1/explorer/contracts/deployments?cursor=` → every contract deployment indexed so far (EVM `CREATE`/`CREATE2` txs and CosmWasm `MsgInstantiateContract`), each row carrying `address, standard, deployer, tx_hash, block_height, block_time, verified` — this is the data source for the deployment-history page added to Phase 3 below.
4. **Verification for this phase:** after Phase 1's deploy script runs, query each new endpoint directly (`curl`) and confirm the JSON matches on-chain ground truth queried independently (`eth_call` / `chaind query wasm ... smart`) — not just "the endpoint returns 200."

---

## Phase 3 — Etherscan/Solscan-parity UI: transaction & token/NFT detail pages

**Goal:** address the user's expectations 2 and 3 directly — deployments and token transfers must *look and behave* like Etherscan (for the EVM side) and Solana's mainnet explorer / Solscan (for the Cosmos/CW side), and clicking a token or NFT must open a real detail page.

Reference behavior to replicate (study these live before building, per the user's request — do not guess the layout from memory):
- **Etherscan** transaction page: status, block, timestamp, from/to (with "Contract Creation" label when applicable), value, transaction fee, gas price/limit/used, nonce, input data (decoded method + params when the contract is verified), and an "Tokens Transferred" panel listing every ERC-20/721/1155 transfer that happened inside that tx.
- **Etherscan** token page: token name/symbol/contract address/decimals, total supply, holder count, transfer count, a holders tab (rank, address, quantity, % of supply), and a transfers tab (txn hash, method, block, age, from, to, quantity).
- **Etherscan** NFT item page: image/media, collection name, token ID, current owner, "Item Activity" transfer history, and raw metadata/attributes.
- **Solscan/Solana Explorer** equivalents for the Cosmos side: account/token overview, holders list, and a transfer/instruction history — apply the same structure to CW20/CW721/CW1155 pages so both chains feel consistent within this one explorer.

Concrete UI tasks:
1. `explorer/app/txs/[hash]` (Cosmos txs) and `explorer/app/evm/txs/[hash]` (EVM txs): add a **"Tokens Transferred"** panel driven by the new `*_token_transfers` tables filtered by `tx_hash` — do not synthesize this from tx memo/log parsing in the frontend; the indexer already parsed it in Phase 2.
2. `explorer/app/evm/tokens/[addr]/page.tsx` and the CW20 equivalent: **delete every `|| "hardcoded fallback"` expression.** Replace with: real data render, or an explicit loading skeleton, or an explicit "Unable to load token data" error state. Wire the Holders and Transfers tabs to the new paginated endpoints (currently the page has no holders/transfers pagination wired to a real backing table — verify and implement).
3. `explorer/app/evm/nfts/[addr]` (collection gallery) and a new/completed `explorer/app/evm/nfts/[addr]/[tokenId]` item detail page (mirror on the CW721 side under `explorer/app/contracts/[addr]/nfts`): real owner, real `tokenURI` metadata/image, real per-token transfer history. If metadata hasn't been indexed yet for a brand-new mint, show a "Metadata indexing…" state, not a stock Unsplash placeholder image (the current `deploy_and_test.sh` mints CW721 tokens with a hardcoded Unsplash URL as `token_uri` — that's fine as *test content* for the mint, but the UI must render whatever `token_uri` actually is, never substitute its own placeholder when a real URI exists).
3a. Add a new `explorer/app/contracts/deployments` (or extend the existing `explorer/app/contracts` list) **deployment-history page**: every EVM and CosmWasm contract deployed so far, sortable by time, with standard badge (ERC-20/721/1155/4626, CW20/721/1155, or "Other"), deployer address, tx hash, and verification badge — backed by the `.../contracts/deployments` endpoint above. On each transaction detail page, when the tx is itself a deployment, label it **"Contract Creation"** (matching Etherscan's convention) and link straight to the new contract's detail page, instead of showing it as a generic call.
3b. Add an `explorer/app/evm/vaults/[addr]` ERC-4626 detail page: underlying asset, total assets, total shares, live share price, and deposit/withdraw history — same "no fabricated APY/yield number" rule as the price/market-cap omission in Phase 5: only show a computed share-price ratio from real state, never an assumed or projected yield.
4. Add "click-through" wiring wherever a token or NFT is referenced elsewhere in the explorer (search results, address balance list, tx detail transfer rows) so every token/NFT chip is a real link to its detail page — audit `explorer/app/search`, `explorer/app/address/[any]`, and `explorer/app/accounts` for any place a token/NFT is currently rendered as static/non-linked text.
5. Cross-link the two account systems where the product genuinely wants that: on the Cosmos holder's account page, show its linked EVM holder address (and vice versa) if the two are meant to represent "the same user's two wallets" for this test flow — but keep their asset lists strictly separate (EVM assets under the EVM address, CW assets under the Cosmos address) per the user's explicit expectation 1.

---

## Phase 4 — End-to-end verification (the user will do this manually; make it possible)

1. Fresh environment: `docker compose down -v && docker compose up -d`, then `./scratch/deploy_and_test.sh` with zero manual intervention and zero hardcoded-path failures.
2. Script prints, at the end: the EVM holder address with its live-queried ERC-20/721/1155 balances, and the Cosmos holder address with its live-queried CW20/721/1155 balances.
3. Open the explorer:
   - The deployment transactions appear in `/txs` and `/evm/txs` with correct status/fees, and each shows its "Tokens Transferred" panel.
   - `/evm/tokens/{ERC20_ADDR}` and `/contracts/{CW20_ADDR}` (or its dedicated CW20 route once added) show the real name/symbol/supply — not "Sovereign Stablecoin"/"sUSDT".
   - Clicking the minted NFT anywhere in the UI opens a real item detail page showing the real owner and real metadata.
4. Nothing in the above should be traceable to a hardcoded fallback string or number anywhere in the frontend or API — grep the diff for `||\s*"` / `??\s*"` patterns on any field sourced from an API response as a final check before calling this done.

---

## Phase 5 — Feature-parity audit against the real Etherscan and Solana Explorer (gap check on the plan itself)

I pulled the live Etherscan transaction/token/holders pages and the live Solana Explorer token/address page to check Phases 2–3 against what these explorers actually show, not just my memory of them. Confirmed structure and additional items the original plan under-specified:

**Etherscan (verified from live pages):**
- Transaction page: status, block, timestamp, from/to, value, transaction fee, **gas price/limit/used, nonce, position in block**, and a decoded **method name** for the call (e.g. "Transfer", "Approve") when the target contract is verified — the plan's Phase 3 UI task 1 only mentioned the "Tokens Transferred" panel, not the method-name decoding or nonce/gas breakdown. **Added below.**
- Token overview page: live **price + 24h change + market cap** (when a market exists), a **"Verification"** badge row, holder count, transfer count, and a **Holders tab with rank/address/quantity/% of supply** plus a **Transfers tab**. Also a top-level **contract address with "Profile Summary"/social links** and a **"More" expandable fields section**. The plan already covers holders/transfers tabs (Phase 2, API endpoints item) — **added:** contract verification status badge and the "More" metadata section.
- Token holders chart page is a dedicated route (`/token/tokenholderchart/{addr}`), separate from the main token page — worth mirroring as its own route rather than a tab-only view so it's linkable/shareable.

**Solana Explorer (verified from live page, USDC token):**
- Address/token overview: **Address, Current Supply, Mint Authority, Freeze Authority, Decimals**, plus a live **Price / 24h Volume / Market Cap** panel and **verification badges** from third parties (here: Bluprynt, CoinGecko, Jupiter, Solflare, RugCheck). Below that, a **Transaction History table**: signature, block (slot), age, timestamp, result (Success/Failed), and a **Raw Data / Download** action.
- This confirms two things the plan should make explicit for the **CW20/CW721 side**, since Solana's SPL-token model (mint/freeze authority) is the closest real-world analog to CW20's admin/minter model:
  - Show **CW20 minter address** and **CW721/CW1155 minter/admin address** on their overview pages the same way Solana shows Mint Authority/Freeze Authority — this is a real on-chain field already available via `contract_info`/`minter` queries, not something to fabricate.
  - Provide a **raw/download JSON** action on token, NFT, and transaction pages (already a real convenience on both reference explorers) — trivial to add once the Phase 2 API endpoints exist, since it's just "expose the raw API response."

**Gaps this review adds to the plan (folded into Phases 2 and 3 above — implement together, not as a separate later phase):**
1. **Method/action decoding on transaction lists and detail pages** — decode the top-level message type (Cosmos: `MsgExecuteContract` action key, e.g. `transfer`/`mint`/`send`; EVM: 4-byte function selector → name, for the small fixed set of ABIs this project deploys itself, since there is no third-party verified-source database to draw from here). No guessing for unknown selectors — show the raw selector/message type if it can't be decoded, never a fabricated label.
2. **Contract verification badge** — since Phase 1 controls both the CW and EVM contract source and deploys them itself, mark them as "Verified" in `explorer.verified_evm_contracts` / `explorer.verified_codes` (tables already exist in the schema per Phase 0 findings) as part of the Phase 1 deploy script, and surface that badge on the token/contract page. Do not default new/unknown contracts to "Verified" — only ones this deploy flow actually built from source.
3. **Mint/admin authority display** — add `minter_address` (CW20/CW721/CW1155) and, for EVM, `owner()`/`Ownable` owner if the deployed test contracts expose it, to the `explorer.contracts` columns added in Phase 2 (schema, item 1), and render it on the token/NFT overview page next to name/symbol/supply.
4. **Dedicated, linkable holders route** (e.g. `/evm/tokens/{addr}/holders`, `/contracts/{addr}/holders`) in addition to the in-page tab, matching Etherscan's separate `tokenholderchart` route.
5. **Raw/Download JSON action** on transaction, token, and NFT detail pages — expose the exact API response as a downloadable/raw view.
6. **Gas/fee breakdown parity for the EVM tx page** — gas price, gas limit, gas used, nonce, position-in-block, alongside the fee total already planned.
7. **Live price/market panel is explicitly out of scope** — both reference explorers show USD price/market cap sourced from external market data providers (CoinGecko-style feeds) for real, liquid assets. This project's ERC-20/CW20 test tokens have no real market, so per the "no mock data" rule, **do not fabricate a price/market cap panel** for them; omit that panel entirely rather than showing a fake number, and only wire it up later if/when the project integrates a real price oracle for a real listed asset.

Net effect: Phase 2's schema gains `minter_address`/`owner_address` and verification-status columns; Phase 3's UI gains method decoding, a verification badge, a dedicated holders route, and raw/download actions; Phase 1's deploy script additionally writes the verified-contract rows for what it deploys. No new phase — these are targeted amendments folded into the existing phases before implementation starts.

---

## Suggested execution order for the implementing agent

1. Phase 1, including the Phase 5 amendment to its deploy script (write verified-contract rows for what it deploys) — unblocks the user's immediate script failure and establishes the two-account model everything else depends on.
2. Phase 2, including its Phase 5 schema amendments (`minter_address`/`owner_address`, verification-status columns) — schema + indexer, no UI value without this.
3. Phase 3, including its Phase 5 UI amendments (method decoding, verification badge, dedicated holders route, raw/download actions, gas/fee breakdown) — UI parity, depends entirely on Phase 2's real data existing.
4. Phase 4 (verification pass, run by the user).

Phase 5 is not a separate execution step — it is a review pass whose findings were folded into Phases 1–3 above; treat its numbered gap list as additions to those phases' checklists, not follow-on work.

Do not start Phase 3 UI work before Phase 2's endpoints return real, verified data — building the UI against fallback values first is exactly the anti-pattern this plan removes.
