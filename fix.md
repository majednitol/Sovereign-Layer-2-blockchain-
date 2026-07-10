# CSOV / ESOV / WSOV — Full Repo Rename Plan

**Purpose:** convert the current conflicting 3-denom setup (`utoken`, `atoken`, `usov`) into the intended clean 3-token model:

| New symbol | Base denom | Decimals | Role |
|---|---|---|---|
| **CSOV** | `ucsov` | 6 | Native Cosmos-side staking/governance/fee token (replaces `utoken`) |
| **ESOV** | `aesov` | 18 | Native gas token for the chain's own embedded EVM (replaces `atoken`/`evmtoken`) |
| **WSOV** | `uwsov` | 6 | Bridge-minted token, minted by `x/bridge` when BSC funds are locked, burned when released back to BSC (replaces `usov`/`sov`) |

This is a pure rename — no mint/burn logic changes are required (see prior report: `x/bridge/keeper.go` already mints on `ProcessBridgeIn` and burns on `ProcessBridgeOut` correctly; it just mints/burns the wrong denom name today).

**Do not do a blind find-and-replace of the string `sov`** — it also appears in unrelated identifiers (`sovereign1...` addresses, `sovereign-l2`, `SovereignEvm`, chain ID `sovereign-testnet-1`, package/module names). Only touch the exact denom tokens listed below (`usov`, `sov` denom unit, `utoken`, `atoken`, `evmtoken`, `token` denom unit, and their `TOKEN`/`SOV` display symbols where used as a currency label, not as part of the project name).

### Why the `u`/`a` prefixes?

This follows the same convention the codebase already uses (`utoken`, `usov`) — it is not new syntax:

- **Cosmos SDK stores and transacts only in integers**, in the smallest ("base") unit of a token — there are no decimals on-chain, the same way Ethereum's ledger only knows "wei," never "0.5 ETH."
- `u` = **micro** = 10⁻⁶. So `ucsov` ("micro-CSOV") is the base unit; 1,000,000 `ucsov` = 1 `CSOV` display unit.
- `a` = **atto** = 10⁻¹⁸, matching Ethereum/EVM's wei-style precision. `aesov` ("atto-ESOV") is the base unit; 10¹⁸ `aesov` = 1 `ESOV`.
- `uwsov` follows the same micro-unit pattern as `ucsov`, at 6 decimals, matching the bridge token's existing precision today.
- The base denom (`ucsov`/`aesov`/`uwsov`) is what genesis, `bond_denom`, gas prices, and `MsgSend` amounts must use internally. The display symbol (`CSOV`/`ESOV`/`WSOV`) is only for human-facing UI (wallets, explorer) via the `denom_units`/`display` fields in `denom_metadata` — this mapping already exists in `chain/genesis.json` today (see Section 1), just with the old names.

---

## 1. Core chain config — do these first, everything else derives from them

| File | Line(s) | Current | Change to |
|---|---|---|---|
| `chain/genesis.json` | 21, 27 | `"base": "atoken"`, `"denom": "atoken"` | `aesov` |
| `chain/genesis.json` | 32, 37 | `"denom": "evmtoken"`, `"display": "evmtoken"` | `esov` |
| `chain/genesis.json` | 42, 46 | `"base": "utoken"`, `"denom": "utoken"` | `ucsov` |
| `chain/genesis.json` | 51 | `"denom": "token"` (display unit, exponent 6) | `csov` |
| `chain/genesis.json` | 61, 65 | `"base": "usov"`, `"denom": "usov"` | `uwsov` |
| `chain/genesis.json` | 70, 75 | `"denom": "sov"`, `"display": "sov"` | `wsov` |
| `chain/genesis.json` | 138 | `"denom": "utoken"` (mint module params) | `ucsov` |
| `chain/genesis.json` | 162 | `"evm_denom": "atoken"` | `aesov` |
| `chain/genesis.json` | 164 | `"extended_denom": "atoken"` | `aesov` |
| `chain/genesis.json` | 209 | `"denom": "utoken"` (feemarket params) | `ucsov` |
| `chain/genesis.json` | 290 | `"bond_denom": "utoken"` | `ucsov` |
| `scripts/generate_genesis.go` | 45 | `TokenDenom = "utoken"` | `TokenDenom = "ucsov"` |
| `scripts/generate_genesis.go` | 47 | `EVMDenom = "atoken"` | `EVMDenom = "aesov"` |
| `scripts/generate_genesis.go` | 131–133 | invariant check `EVMDenom != "atoken"` | update expected value to `"aesov"` |
| `scripts/generate_genesis.go` | 468, 473, 478–479 | `"denom": "atoken"`, `"denom": "evmtoken"`, `"base": "atoken"`, `"display": "evmtoken"` | `aesov` / `esov` |
| `scripts/generate_genesis.go` | 487, 497 | `"denom": "utoken"`, `"base": "utoken"` | `ucsov` |
| `scripts/generate_genesis.go` | 506, 516 | `"denom": "usov"`, `"base": "usov"` | `uwsov` |
| `scripts/generate_genesis.go` | 51, 58, 60, 699, 700 | comments referencing `utoken` for supply/rewards math | update comment text to `ucsov` (values unaffected) |
| `chain/config/app.toml` | 8 | `minimum-gas-prices = "0atoken"` | `"0aesov"` |
| `chain/cmd/chaind/main.go` | 127 | comment: "staking operates in utoken (6 decimals)" | update comment to `ucsov` |
| `docker-compose.yml` | 205 | `DENOM=atoken` | `DENOM=aesov` |

## 2. Chain modules (`chain/x/...`)

| File | What to change |
|---|---|
| `chain/x/bridge/keeper.go` (lines 43, 203, 248) | `"usov"` → `"uwsov"` (also update the comment "5000 SOV/usov threshold" → "5000 WSOV/uwsov threshold") |
| `chain/x/bridge/keeper_test.go` (170, 199, 200, 226, 244, 253, 254) | `"usov"` → `"uwsov"` in test fixtures/assertions |
| `chain/x/bridge/simulation.go` (60, 108, 117, 166) | `"usov"` → `"uwsov"` |
| `chain/x/milestone/keeper.go` (280) | `"usov"` → decide: milestone disbursements should likely pay out in **CSOV** (`ucsov`), not WSOV, since milestones are a native governance/treasury function, not a bridge mint — confirm with product before renaming; do not default to `uwsov` here |
| `chain/x/settlement/keeper_test.go` (106), `chain/x/settlement/simulation.go` (60, 100, 143) | Same milestone-style question — confirm whether settlement transfers should be CSOV or WSOV before renaming; if these represent internal native transfers, use `ucsov` |
| `chain/x/validator/keeper.go` (206) | `GetBalance(ctx, distrAddr, "usov")` — validator reward distribution should almost certainly be **CSOV** (`ucsov`), since staking rewards are the native token, not the bridge-wrapped one. Flag this as a likely pre-existing bug (rewards were being paid in the bridge-mint denom instead of the staking denom) — fix to `ucsov`, don't just rename. |

## 3. Backend services (`backend/module/...`)

| File | Line(s) | Change |
|---|---|---|
| `backend/module/faucet/main.go` | 66 | `denom = "usov"` → `"uwsov"` — **but confirm**: if the faucet is meant to fund users with the native gas/staking token (CSOV) for testnet use, not the bridge-wrapped token, this should be `ucsov` instead. Current behavior (funding `usov`) suggests the faucet was already faucet-funding the wrong denom relative to `bond_denom` — same class of bug as validator rewards above. |
| `backend/module/faucet/main.go` | 161 | `"--gas-prices", "1000000000atoken"` | `"1000000000aesov"` |
| `backend/module/faucet/main_test.go` | 171, 185, 194, 225, 226 | update `usov`/`atoken` literals to match whichever denom faucet is fixed to use |
| `backend/module/api/main.go` | 408, 951 | `token = "usov"` / `TokenAddress: "usov"` → `"uwsov"` (this is the bridge-tx API, correctly bridge-scoped) |
| `backend/module/api/api_test.go` | 59, 72, 90, 91 | update `"usov"` literals to `"uwsov"` |
| `backend/module/projection/main.go` | 211, 218, 227, 249, 256, 265 | `"usov"` → `"uwsov"` (all bridge lock/release event projections — correctly bridge-scoped) |
| `backend/module/ingestion/ingestion_test.go` | 33 | `"amount":"500usov"` → `"500uwsov"` |
| `backend/module/projection/projection_test.go` | 13, 18, 31, 40, 41 | `usov` literals → `uwsov` |

## 4. Relayer

| File | Line(s) | Change |
|---|---|---|
| `relayer/cmd/relayer/main.go` | 198, 379 | `sdk.NewCoin("usov", ...)` → `sdk.NewCoin("uwsov", ...)` |
| `relayer/relayer_test.go` | 29, 39 | update comments referencing `usov` |

## 5. Frontend (wallet/dashboard app, `frontend/`)

| File | Line(s) | Change |
|---|---|---|
| `frontend/config/wallets.json` | Keplr `currencies`/`feeCurrencies`/`stakeCurrency` | `coinDenom: "SOV"` / `coinMinimalDenom: "usov"` → `coinDenom: "CSOV"` / `coinMinimalDenom: "ucsov"` (staking/fee/gas currency must be CSOV, the actual `bond_denom`, not the bridge token — this is also a pre-existing correctness bug: wallets were configured to show/stake the wrong denom) |
| `frontend/config/wallets.json` | `metamaskSovereignEvm.nativeCurrency` | `symbol: "TOKEN"` → `symbol: "ESOV"`, decimals stays 18 |
| `frontend/app/dashboard/page.tsx` | 270 | `tokenAddress: "usov"` → `"uwsov"` (confirm this is genuinely a bridge-token display; if it's meant to show the user's native balance, use `ucsov`) |
| `frontend/app/page.tsx` | 67 | Same check as above |
| `frontend/app/governance/page.tsx` | 132, 138 | `denom: "atoken"` for governance deposit/voting amounts → should be `ucsov` (native governance token), **not** `aesov` (EVM gas) — this looks like an existing bug: governance deposits were denominated in the EVM gas token instead of the staking/governance token |

## 6. Explorer UI (`explorer/`) and Explorer API (`explorer-api/`)

| File | Line(s) | Change |
|---|---|---|
| `explorer-api/main.go` | 3199 | `"unit": "usov"` → confirm context (appears to be a fee/amount unit label) and set to `uwsov` or `ucsov` accordingly |
| `explorer/app/blocks/page.tsx` | 113, 123 | `"0.025 usov"`, `` `${burnt} usov` `` — these are gas/burn-fee displays, should be `ucsov` (native fee denom), not the bridge token |

## 7. Infra / deployment configs

| File | Line(s) | Change |
|---|---|---|
| `infra/k8s/faucet.yaml` | 30 | `value: "usov"` → matches whatever `backend/module/faucet` is fixed to use (see Section 3) |
| `infra/sovereign-k8s/11.faucet/faucet.yaml` | 31 | Same |
| `chain/entrypoint.sh` | 37, 40, 51 | Update comments referencing `TOKEN`/`utoken`/`atoken`/`usov` amounts to `CSOV`/`ucsov`/`ESOV`/`aesov`/`WSOV`/`uwsov` |
| `scripts/testnet_launch_ceremony.sh` | 32 | `"$TOTAL_SUPPLY usov"` → `"$TOTAL_SUPPLY uwsov"` or `ucsov` depending on which supply this validates (check: this looks like it's validating the bridge `SupplyCap`, so likely `uwsov`) |

## 8. Documentation (must be updated so ADRs stay accurate)

| File | Change |
|---|---|
| `doc/adr/adr-011-evm-denomination.md` | Rewrite to describe three denoms (`ucsov`/`aesov`/`uwsov`) instead of two (`utoken`/`atoken`); document the new WSOV bridge-mint token explicitly |
| `doc/adr/adr-009-evm-chain-id.md` | Line 16: `Currency Symbol: atoken` → `ESOV` (`aesov`) |
| `doc/governance/genesis_parameters.md` | Lines 12–13: update `Base Denomination`/`EVM Denomination` rows to `ucsov`/`CSOV` and `aesov`/`ESOV`; add a new row for `Bridge Denomination` = `uwsov`/`WSOV` |
| `doc/mainnet/chain-registry.json` | Lines 19, 30: `"denom": "atoken"` → `aesov`, `"denom": "utoken"` → `ucsov`; add WSOV entry if the registry lists all chain assets |
| `doc/ops/audit_engagement.json` | Line 84: update denom reference in audit scope description to `aesov` |
| `planned-vs-implemented.md` | Lines 46, 67, 69, 211, 456, 553 — update historical denom references for accuracy (informational only, low priority) |

## 9. Tests — update after production code is renamed, not before

All `_test.go` files listed above (`keeper_test.go`, `simulation.go` test helpers, `e2e/*.go` — `evm_cosmwasm_test.go`, `phase_1` through `phase_10_verification_test.go`, `real_testnet_integration_test.go`, `phase_5_gaps_test.go`) contain matching `usov`/`atoken`/`utoken` literals and hardcoded expected-value assertions (e.g. `phase_9_verification_test.go` checks `genesisContent` contains `"evm_denom": "atoken"` — this must become `"aesov"` or the test will correctly start failing, which is the intended signal that the rename is complete). Update these last, as a verification step: **once every test above is updated to expect `ucsov`/`aesov`/`uwsov` and the full suite passes, the rename is confirmed complete.**

## 10. Miscellaneous / dev scratch scripts

| File | Line(s) | Change |
|---|---|---|
| `scratch/deploy_and_test.sh` | 50, 52, 56 | `denom=="atoken"` filter, comments → `aesov` |

---

## Pre-existing bugs uncovered during this mapping (fix, don't just rename)

While mapping every occurrence, three spots turned up where the **wrong denom category** was used, not just the wrong name — worth flagging to the dev team explicitly since renaming alone would preserve the bug:

1. **`chain/x/validator/keeper.go:206`** — validator reward distribution reads balance in `usov` (bridge token) instead of the staking denom. Should be `ucsov`.
2. **`frontend/config/wallets.json`** — Keplr wallet's `stakeCurrency`/`feeCurrencies` are set to `usov`, meaning users would be shown/staking the bridge token instead of the actual `bond_denom`. Should be `ucsov`.
3. **`frontend/app/governance/page.tsx:132,138`** — governance proposal deposit and vote amounts are denominated in `atoken` (EVM gas token) instead of the governance/staking denom. Should be `ucsov`.

## 11. Additional files found in the full-repo verification pass (not in the original scan — add these)

| File | Line(s) | Current | Change |
|---|---|---|---|
| `chain/app/simulation_test.go` | 330 | comment "EVM ether/atoken transfer" | update comment to `aesov` |
| `contracts/governance/tests/integration_tests.rs` | 204, 205, 216, 411, 435, 461, 478 | `"usov"` (treasury/reserve/vote-deposit test fixtures) | These test a **native governance contract** moving treasury funds — should be `ucsov`, not `uwsov`. Same "wrong denom category" bug pattern as the validator-reward and governance-page findings above. |
| `contracts/reserve-fund/src/contract.rs` | 216, 241, 248, 272, 282, 307 | `"usov"` | Reserve fund is a native treasury contract — should be `ucsov`, not `uwsov`. Flag as a bug, not a pure rename. |
| `contracts/treasury/src/contract.rs` | 192, 216, 229, 250 | `"usov"` | Same — native treasury contract, should be `ucsov`. |
| `doc/adr/adr-007-operational-security.md` | 12, 26, 28 | `usov` base denom description, `$100 usov per unit gas` | Update to reflect final split: base staking/fee denom is `ucsov`; if this ADR is describing EVM gas specifically, that portion should reference `aesov` instead |
| `doc/ops/runbooks.md` | 20, 32, 44 | `--fees 2000usov` (gov proposal fee examples) | `2000ucsov` (governance/tx fees are native, not bridge-wrapped) |
| `doc/testnet/onboarding.md` | 20 | `gentx ... 100000000000usov` (validator self-delegation) | `ucsov` — self-delegation must be the actual `bond_denom` |
| `doc/plans/2026-06-21-cosmwasm-contract-suite.md` | 123, 131 | `"usov"` in planning doc code samples | `ucsov` (matches the treasury contract fix above) |
| `doc/phase_0_audit_report.md` | 90 | `SOV` symbol reference | Update per context — likely `CSOV` |
| `e2e/comprehensive_phases_1_to_7_test.go` | 573, 574, 783, 904, 920, 928, 961, 1122, 1161, 1162, 1214, 1323, 1324, 1380, 1388, 1397 | `"usov"` literals across milestone/settlement/bridge test scenarios | Split by context: milestone/settlement escrow tests → `ucsov`; bridge-in/relayer tests (1380, 1388, 1397) → `uwsov` |
| `e2e/cosmwasm_counter_test.go` | 47, 93, 177, 207, 237, 267, 294, 315, 342 | `"--gas-prices", "0.025atoken"` | `"0.025aesov"` |
| `e2e/contracts/cosmwasm/query_counter.sh` | 29 | `GAS_PRICES="0.025atoken"` | `"0.025aesov"` |
| `e2e/contracts/cosmwasm/store_and_verify.sh` | 30 | `GAS_PRICES="${GAS_PRICES:-0.025atoken}"` | `"${GAS_PRICES:-0.025aesov}"` |
| `e2e/contracts/TEST_FLOW.md` | 236 | `--gas-prices 0.025usovereign` | This is a distinct typo/leftover denom (`usovereign`, not `usov`/`utoken`) — replace with `aesov` (this is a CosmWasm tx, so likely intended `utoken`→now `ucsov`; confirm which gas denom this contract flow actually pays in before finalizing) |

**Excluded as false positives (do not touch):** `scratch/solve_bech32.py` lines 37, 39 — these are bech32 **address-prefix (HRP)** candidates (`"sov"`, `"sovereignvaloper"`), not currency denoms; unrelated to this rename.

## 12. UI display symbol `"SOV"` — much broader than originally scanned; must be reclassified per context, not blanket-renamed

The initial plan under-covered how many places the *display* symbol `SOV` (as opposed to the base denom `usov`) appears in the two frontend apps. A follow-up exhaustive scan found `SOV`/`"sov"` as a display label or context-string in **37 additional files**, almost entirely in `explorer/app/**` and `frontend/app/**`. These are user-facing amount labels (e.g. "10,000 SOV", "0.025 SOV gas", bridge deposit/withdraw amount suffixes, staking amount suffixes, NFT price tags, tx explorer labels).

**Because `SOV` was the single, generic label for "the native asset" before this rename, each occurrence must be classified by what the amount actually represents, not blanket-replaced:**

| If the amount represents… | Replace `SOV` with |
|---|---|
| Staking, governance, gas fees, general account balance | `CSOV` |
| EVM-side balance/gas (MetaMask, EVM contract pages) | `ESOV` |
| Bridge deposit/withdraw amount, bridge tx history, bridge-related balance card | `WSOV` |

Files requiring this contextual pass (grouped by area — the fixing agent should open each and decide CSOV/ESOV/WSOV per amount shown, not do a blind replace):

- **Explorer bridge pages** (definitely `WSOV`): `explorer/app/bridge/page.tsx`, `explorer/app/bridge/deposit/page.tsx`, `explorer/app/bridge/withdraw/page.tsx`, `explorer/app/bridge/tx/[nonce]/page.tsx`
- **Explorer general/staking/gas pages** (likely `CSOV`): `explorer/app/accounts/page.tsx`, `explorer/app/gastracker/page.tsx`, `explorer/app/params/page.tsx`, `explorer/app/stats/page.tsx`, `explorer/app/stat/supply/page.tsx`, `explorer/app/txs/[hash]/page.tsx`, `explorer/app/txs/advanced-filter/page.tsx`, `explorer/app/faucet/page.tsx`, `explorer/app/tools/page.tsx`, `explorer/app/developers/page.tsx`, `explorer/app/token-approvals/page.tsx`, `explorer/components/charts/TallyBar.tsx`, `explorer/components/GlobalHeader.tsx`
- **Explorer NFT page** (verify context — if showing NFT prices denominated in native token): `explorer/app/nfts/page.tsx`
- **Frontend wallet/dashboard app** (mixed — verify each): `frontend/app/dashboard/page.tsx`, `frontend/app/page.tsx`, `frontend/app/governance/page.tsx` (governance → `CSOV`, per the bug noted in Section 5), `frontend/config/wallets.json` (already covered in Section 5)
- **Bridge contract test/deploy naming** (BSC-side token used by `LockBox`, not a UI label): `bridge/src/MockERC20.sol:6` `symbol = "SOV"` — since the real BSC-side asset is whatever `LockBox` locks (real or mock ERC-20), this can stay `SOV` (or be renamed to reflect a real production token symbol later) — it is **not** part of the CSOV/ESOV/WSOV Cosmos-side scheme and should not be forced into that naming.
- **`chain/entrypoint.sh:40`, `chain/x/bridge/keeper.go:43`, `scripts/testnet_launch_ceremony.sh`, `relayer/cmd/relayer/main.go`, `e2e/phase_4_verification_test.go`, `e2e/phase_6_verification_test.go`, `e2e/real_testnet_integration_test.go`, `explorer-api/main.go` (lines 806, 2959, 3779, 3796–3800), `planned-vs-implemented.md`** — comments/log strings referencing `SOV`, update to whichever of CSOV/WSOV applies to that code path (mostly `WSOV` for bridge-keeper/relayer contexts, `CSOV` for genesis/entrypoint funding comments).

## 13. Final verification — run this after all changes, must return zero output

Give this exact command to the fixing agent as the acceptance test. Any line it prints is a leftover that must be fixed before the rename is considered complete:

```bash
grep -rnoI -e '\butoken\b' -e '\batoken\b' -e '\busov\b' -e '\bevmtoken\b' \
  --include="*.go" --include="*.json" --include="*.ts" --include="*.tsx" \
  --include="*.sol" --include="*.rs" --include="*.toml" --include="*.yaml" \
  --include="*.yml" --include="*.sh" --include="*.md" --include="*.env" \
  . 2>/dev/null | grep -viE "node_modules|\.next|\.git/|/lib/forge-std|cache/|out/"
```

A second pass is needed for the bare `SOV`/`"sov"` display symbol, since it's a legitimate substring of `sovereign`/`Sovereign` everywhere else in the repo (project name, chain ID, addresses, package names) — those must **not** be touched. Use this command, which excludes all `sovereign`/`Sovereign` matches, and manually confirm every remaining hit has been resolved to `CSOV`/`ESOV`/`WSOV` per Section 12 (a hit here is not automatically wrong — it just needs a human check that it was intentionally classified, not skipped):

```bash
grep -rnoI -e '\bSOV\b' -e '"sov"' \
  --include="*.go" --include="*.json" --include="*.ts" --include="*.tsx" \
  --include="*.sol" --include="*.rs" --include="*.md" --include="*.py" \
  . 2>/dev/null | grep -viE "node_modules|\.next|\.git/|/lib/forge-std|cache/|out/" \
  | grep -viE "Sovereign|sovereign"
```

Expected remaining hits after a correct rename: only `bridge/src/MockERC20.sol:6` (`symbol = "SOV"`, intentionally excluded per Section 12) and `scratch/solve_bech32.py` (bech32 prefix candidates, unrelated to denom, excluded per Section 11). Anything else in the output means a file was missed or a `SOV` label was left unclassified.

## 14. Address scheme review — current implementation is correct, but `wallets.json` contradicts it

> **⚠️ Superseded by Section 16.** This section originally recommended switching the prefix to `sovereign`. The user has since confirmed the chain should keep the standard `cosmos` prefix instead — see Section 16 for the final, authoritative direction. Read this section only for the general address-scheme background (unified address codec, EVM hex validity, wallet-registration mechanics), and ignore its `"sovereign"` prefix recommendation in the table below.

The user raised a concern that the chain's address scheme ("unified address," `sovereign1...` prefix) might not be wallet-compatible. Findings:

**The address design itself is correct and standards-compliant — no change needed:**
- `chain/app/app.go` uses `evmaddress.NewEvmCodec(...)` from `github.com/cosmos/evm/encoding/address` — the official Cosmos EVM module's address codec, the same pattern used by Evmos, Injective, Kava, Cronos. One 20-byte key, encoded two ways: `sovereign1...` (bech32, for Cosmos-side modules) and `0x...` (hex, for the embedded EVM). This is not a mock and not a bug.
- `sovereign1...` **is** a valid, standard bech32 address — structurally identical to `cosmos1...`/`osmo1...`/`inj1...`. A custom prefix is required, not a defect: `cosmos` is reserved for the Cosmos Hub specifically, so this chain must use its own prefix (`sovereign`) rather than borrowing Cosmos Hub's.
- `0x...` (20-byte hex) is already a fully valid, standard Ethereum address — MetaMask-compatible as-is, no custom encoding involved.
- Wallet support for a non-preloaded chain always requires one-time registration (e.g. Keplr's `experimentalSuggestChain(...)`) — this is normal for every Cosmos chain that isn't the Hub or Osmosis, not something specific or broken about this project.

**The actual bug is in `frontend/config/wallets.json`'s Keplr config — it contradicts the chain's real address/denom scheme:**

| Field | Current (wrong) | Should be |
|---|---|---|
| `bech32Config.bech32PrefixAccAddr` | `"cosmos"` | `"sovereign"` |
| `bech32Config.bech32PrefixAccPub` | `"cosmospub"` | `"sovereignpub"` |
| `bech32Config.bech32PrefixValAddr` | `"cosmosvaloper"` | `"sovereignvaloper"` |
| `bech32Config.bech32PrefixValPub` | `"cosmosvaloperpub"` | `"sovereignvaloperpub"` |
| `bech32Config.bech32PrefixConsAddr` | `"cosmosvalcons"` | `"sovereignvalcons"` |
| `bech32Config.bech32PrefixConsPub` | `"cosmosvalconspub"` | `"sovereignvalconspub"` |
| `currencies[0].coinDenom` / `stakeCurrency.coinDenom` / `feeCurrencies[0].coinDenom` | `"SOV"` | `"CSOV"` (see Section 5) |
| `currencies[0].coinMinimalDenom` / `stakeCurrency.coinMinimalDenom` / `feeCurrencies[0].coinMinimalDenom` | `"usov"` | `"ucsov"` (see Section 5) |
| `bip44.coinType` | `60` (Ethereum's coin type) | Verify intentionally: `60` is correct **only if** this chain wants Keplr to derive Cosmos-side keys using an Ethereum-style HD path (as Evmos/Injective do, to keep the same seed phrase producing the same 20-byte key on both sides). If that's not the intent, this should be `118` (standard Cosmos coin type) — but note doing so would produce **different** bytes for the Cosmos vs. EVM address, breaking the "one key, two representations" unification, since a different coin type derives an entirely different private key from the same seed. Confirm which is wanted before changing; do not change to `118` without also deciding whether unification is being kept (this ties directly into the address-scheme discussion above). |

With this mismatch left uncorrected, Keplr would register the chain expecting `cosmos1...`-prefixed addresses and the `usov`/`SOV` denom — neither of which match what the chain actually produces or (post-rename) will be named — so wallet connect/sign/balance-display would all fail or show wrong data, independent of the denom-rename work.

**`metamaskSovereignEvm` config (lines 43–53)** — separately, `nativeCurrency.symbol: "TOKEN"` here is the same stale placeholder denom flagged in Section 5; per the CSOV/ESOV/WSOV scheme this MetaMask-facing native gas token should read `"ESOV"`, decimals `18` (already correct).

## 15. Two independent, conflicting wallet-connection implementations — must be reconciled, not just denom-renamed

> **⚠️ Prefix target superseded by Section 16.** Everywhere below that says `"sovereign"` is the correct prefix and `"cosmos"` is wrong, read it reversed: per the confirmed decision in Section 16, `"cosmos"` is correct and `explorer/store/wallet.ts`'s current `"sovereign"` value is what needs to change. The rest of this section's findings (chain ID mismatch, coin-type mismatch, EVM chain ID mismatch, `"SLT"` placeholder, missing Leap/Cosmostation in `frontend/`) are unaffected and still apply as written.

The `explorer/` and `frontend/` apps each ship their **own separate** wallet-connect implementation, with different chain IDs, different coin types, and different (in one case wrong) bech32 config. Fixing `frontend/config/wallets.json` alone (Section 14) is not sufficient — `explorer/store/wallet.ts` is a fully independent code path with its own inline chain config that never reads `wallets.json` at all.

| | `explorer/store/wallet.ts` (+ `MultiWalletButton.tsx`) | `frontend/components/WalletConnect.tsx` (+ `config/wallets.json`) |
|---|---|---|
| Wallets supported | Keplr, Leap, Cosmostation, MetaMask | Keplr, MetaMask only |
| Config source | Inline object in `wallet.ts`, built from `NEXT_PUBLIC_*` env vars | Imports `frontend/config/wallets.json` directly |
| Cosmos chain ID | `"sovereign-1"` (env-overridable, defaults to this) | `"sovereign-testnet-1"` (hardcoded) |
| bech32 prefix | `"sovereign"` — already correct | `"cosmos"` — wrong, per Section 14 |
| `bip44.coinType` | `118` | `60` |
| Native symbol/denom | `NEXT_PUBLIC_CURRENCY_SYMBOL` env var, defaults to `"SLT"` — a **third**, undocumented placeholder symbol, not part of the `utoken`/`atoken`/`usov` set already tracked, and not in the CSOV/ESOV/WSOV scheme either | `"SOV"` / `"usov"` from `wallets.json` |
| EVM chain ID | `NEXT_PUBLIC_EVM_CHAIN_ID` env var, defaults to `"7777"` decimal | `walletsConfig.metamaskSovereignEvm.chainId` = `"0x2329"` (8745 decimal) — hardcoded, different value |

**Consequences if left as-is:** since both apps are for the same product (per `chain/genesis.json`'s actual chain ID and `app.go`'s actual bech32/coin-type settings), a user who connects Keplr via `explorer/` registers one chain entry, and a user who connects via `frontend/` registers a **different** chain entry (different ID, different coin type, wrong prefix) — Keplr would treat them as two unrelated chains, and the `frontend/` one would produce/expect addresses that don't match what the real chain (per `app.go`) actually issues.

**Required fixes, in addition to the wallets.json field corrections in Section 14:**
1. Pick one source of truth for chain config (recommend: a single shared config file/package both apps import — do not keep two independently-maintained copies) — chain ID, bech32 prefixes, coin type, EVM chain ID, and CSOV denom must be identical in both apps.
2. Resolve which chain ID is real: `chain/genesis.json` and `chain/entrypoint.sh` should be checked to confirm whether it's `sovereign-1` (mainnet-style) or `sovereign-testnet-1` (testnet-style) — right now the two frontends disagree on this, independent of the denom rename.
3. Resolve the `coinType` 118-vs-60 conflict — per Section 14's analysis, this determines whether Cosmos/EVM address unification actually works; the two apps currently assume different, incompatible answers, so at most one of them is correct today.
4. Reconcile the EVM chain ID (`7777` vs `8745`/`0x2329`) — MetaMask will register two different EVM networks depending on which app the user used.
5. Replace the undocumented `"SLT"` fallback symbol in `explorer/store/wallet.ts` with `CSOV`/`ucsov` per the naming scheme, and drive it from the same source as `frontend/`'s config rather than a separate env var with its own independent default.
6. Once reconciled, re-run this file's Section 13 verification greps — add `explorer/store/wallet.ts` explicitly to the file list checked, since it wasn't caught by the original `utoken`/`atoken`/`usov`/`SOV` literal scan (its placeholder is `"SLT"`, a fourth denom string outside the original grep patterns).
7. **Wallet coverage gap:** `explorer/` already supports all 4 wallets (Keplr, Leap, Cosmostation, MetaMask) via its generic `suggestCosmosChain()` helper, since Leap and Cosmostation both expose Keplr-compatible APIs. `frontend/components/WalletConnect.tsx` currently only implements Keplr and MetaMask. Extend `frontend/`'s implementation with Leap and Cosmostation branches, following the same pattern already working in `explorer/store/wallet.ts`, so both apps offer all four wallets rather than two apps with different wallet coverage.
8. **Address-format summary for the fixing agent to confirm end-to-end:** `sovereign1...` (bech32) must work identically across Keplr, Leap, and Cosmostation once (1)–(4) above are reconciled; `0x...` (hex) must work in MetaMask. Because this chain uses the unified address codec (Section 14), the same 20-byte key should produce the same `sovereign1...` address in all three Cosmos wallets and the same `0x...` address in MetaMask for a given seed phrase — verify this by connecting with each of the four wallets using the same test seed and confirming the addresses match expectations.

## 16. DECISION CONFIRMED — keep the `cosmos1...` bech32 prefix (Cosmos SDK default), do not switch to `sovereign1...`

**Product decision (confirmed by the user):** this chain will intentionally use the standard Cosmos SDK default bech32 prefix `cosmos` for all addresses, not a chain-specific prefix like `sovereign`. Addresses will look like `cosmos1...`, structurally identical in format to Cosmos Hub's own addresses (they remain cryptographically and semantically distinct — same as any two different chains that happen to reuse this default — since chains are still told apart by chain ID, RPC endpoint, and genesis, not by the address text itself).

**Tradeoff accepted:** a `cosmos1...` address from this chain cannot be visually distinguished from a real Cosmos Hub address just by looking at the string — every other major Cosmos chain (Osmosis `osmo1`, Injective `inj1`, Evmos `evmos1`, etc.) avoids this by using its own prefix. This project has explicitly chosen not to follow that convention.

**Practical consequence — this reverses the fix originally proposed here:**
- `chain/cmd/chaind/main.go`'s `setupSDKConfig()` — **no change needed.** The current code (`cfg.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)`) already produces `cosmos1...` by using the SDK's own default constants; leave it exactly as-is.
- `chain/app/app.go`'s `AddressCodec`/`ValidatorAddressCodec` (`evmaddress.NewEvmCodec(sdk.Bech32PrefixAccAddr)`, etc.) — **no change needed**, same reasoning.
- **Section 14's `wallets.json` fix is reversed**: `bech32Config.bech32PrefixAccAddr` should be `"cosmos"` (not `"sovereign"` as Section 14 originally said) — meaning the *current* value in the file (`"cosmos"`) was already correct, and Section 14's finding no longer applies. Do not change this field.
- **Section 15's cross-app reconciliation still applies**, but the target value both apps must agree on is `"cosmos"`/`"cosmospub"`/`"cosmosvaloper"`/etc. — the Cosmos SDK defaults — not the `"sovereign*"` values listed there. `explorer/store/wallet.ts` currently uses `"sovereign"` in its `suggestCosmosChain()` config (lines 32-39) — **this now needs to change to `"cosmos"` to match the chain**, the opposite direction from what was previously assumed.
- The demo address pairs shown earlier in this conversation used a `sovereign1...` HRP for illustration — with this decision, the real addresses this chain produces will look like `cosmos17w0adeg64ky0daxwd2ugyuneellmjgsxkcvw5v` (same bytes, `cosmos` prefix instead).
- `coin_type` (60 vs. 118, discussed in Sections 14–15) is **unaffected by this decision** — that's a separate axis (key derivation, not address text prefix) and still needs its own resolution.

Everything else in Sections 1–15 (denom rename, bug fixes, wallet-count parity across the two apps, final verification greps) is unaffected by this decision and should proceed as written — only the specific bech32-prefix value target changes, from `sovereign` to `cosmos`.

---

## 16a. Original finding (superseded by the decision above — kept for context only)

The section below was the original analysis before the product decision was made. It is no longer the recommended action — Section 16 above is authoritative. Retained so the fixing agent understands *why* `wallets.json` currently says `"cosmos"` and why that is now correct, not a leftover bug.

The chain does not actually produce `sovereign1...` addresses today; it produces `cosmos1...` due to a self-referential config bug

Everything in Sections 14–15, and the demo addresses shown earlier, assumed the chain's real bech32 prefix is `sovereign` (based on `app.go`'s `evmaddress.NewEvmCodec(sdk.Bech32PrefixAccAddr)` and the module path `github.com/sovereign-l1/chain/...`). Verifying this directly against `chain/cmd/chaind/main.go` surfaced a real bug that changes that conclusion:

```go
func setupSDKConfig() {
    cfg := sdk.GetConfig()
    cfg.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, sdk.Bech32PrefixAccPub)
    cfg.SetBech32PrefixForValidator(sdk.Bech32PrefixValAddr, sdk.Bech32PrefixValPub)
    cfg.SetBech32PrefixForConsensusNode(sdk.Bech32PrefixConsAddr, sdk.Bech32PrefixConsPub)
    cfg.Seal()
}
```

`sdk.Bech32PrefixAccAddr` etc. are the **Cosmos SDK's own built-in default constants** — confirmed against the upstream `cosmos-sdk` source (`types/address.go`): `Bech32MainPrefix = "cosmos"`, and `Bech32PrefixAccAddr = Bech32MainPrefix` (i.e. `"cosmos"`) unless a chain defines its own constants and passes those in instead. This code reads the SDK's default value and feeds it right back into itself — a no-op. **As written, this chain's real runtime bech32 prefix is `cosmos`, not `sovereign`.** Every address this chain actually generates today is `cosmos1...`, colliding in form with Cosmos Hub's own address format — the exact problem Section 14 warned against, except it's happening on the chain itself, not just in a frontend config file.

This means the earlier finding in Section 14 needs to be read the other way around: it isn't that `wallets.json` wrongly says `"cosmos"` while the chain correctly uses `"sovereign"` — **both are currently wrong, but they're accidentally consistent with each other** (both currently produce/expect `cosmos1...`). Fixing only `wallets.json` per Section 14 without also fixing this chain-level bug would leave the chain still generating `cosmos1...` addresses.

**Required fix:**
1. Define real project constants (e.g. in `chain/app/app.go` or a small `params` file): `AccountAddressPrefix = "sovereign"`, plus the derived `sovereignpub`/`sovereignvaloper`/`sovereignvaloperpub`/`sovereignvalcons`/`sovereignvalconspub` values.
2. In `setupSDKConfig()`, pass these new constants — not `sdk.Bech32PrefixAccAddr`/`sdk.Bech32PrefixValAddr`/`sdk.Bech32PrefixConsAddr` — into `SetBech32PrefixForAccount`/`SetBech32PrefixForValidator`/`SetBech32PrefixForConsensusNode`.
3. Update the same constants wherever else `sdk.Bech32PrefixAccAddr` is referenced (e.g. `chain/app/app.go:260-261`, `chain/app/app.go:353-354,373-374` per the earlier codec review) so the `AddressCodec`/`ValidatorAddressCodec` construction uses the real prefix too, not the SDK default.
4. This is a **breaking, one-time, genesis-time-only change** — any address, key, or test fixture generated before this fix used `cosmos1...` and will not match addresses generated after it. Must be fixed and locked before mainnet genesis, and ideally before any real testnet keys are distributed.
5. Add an explicit unit/integration test asserting a freshly-generated account address starts with `sovereign1`, not `cosmos1` — this exact bug (SDK-default no-op) is easy to silently reintroduce and cheap to test for.
6. Once fixed, regenerate the demo address pairs shown earlier in this conversation — they were illustrative of the *intended* end-state, not addresses the chain can currently produce.

This finding takes priority over Sections 14–15's wallet-config work — fix the chain-level prefix first, then verify `wallets.json` and both frontend apps' configs actually match what the fixed chain produces.

### Verified against official sources

Checked the claims above against upstream documentation rather than relying on memory:

- **Unified address codec is the real, current Cosmos EVM pattern** — `github.com/cosmos/evm/encoding/address` (the codec actually imported in `chain/app/app.go`) is the maintained upstream package (`cosmos/evm` repo, `encoding/address/address_codec.go`). Cosmos's own EVM docs ("Accounts" page, `docs.cosmos.network/evm/latest/.../accounts`) describe this exact design: "Cosmos EVM accounts are implemented to be compatible with Ethereum type addresses" — one 20-byte key, bech32 and hex are just two encodings of it. There's also an official Bech32 ↔ hex conversion precompile documented at `docs.cosmos.network/evm/latest/.../precompiles/bech32`, confirming lossless two-way conversion is the intended, supported model — matching what `scratch/convert.go` in this repo demonstrates.
- **`coin_type` 60 vs. 118 tradeoff is real, not invented** — Cosmos SDK chains default to BIP-44 coin type `118` (see Keplr's own "Coin Type 118" explainer). Ethermint/Cosmos-EVM chains deliberately override this to `60` (Ethereum's coin type) specifically so the same seed phrase derives the same 20-byte key for both the Cosmos and EVM sides — there's a dedicated historical bugfix in `cosmos/ethermint` (PR #577, "fix Bip44 derivation path") establishing this as the correct, intentional value for unified-address chains. Confirms the plan's warning: don't flip this to `118` without deciding whether you're keeping address unification, since a different coin type derives different keybytes entirely.
- **Bech32 custom-prefix requirement is real, not a limitation of this project** — Cosmos SDK's own address-encoding reference (`docs.cosmos.network/sdk/latest/guides/reference/bech32`) confirms bech32 prefixes are how chains distinguish their address space; `cosmos` is Cosmos Hub's own prefix, not a generic default others should reuse.
- **Wallet chain-registration flow is standard, not project-specific** — Keplr's own docs for `experimentalSuggestChain` (`docs.keplr.app/api/guide/suggest-chain`) confirm this is the documented, expected mechanism for any Cosmos-SDK chain "that isn't natively integrated to Keplr" — exactly the situation this testnet is in, and not something wrong with the chain's design.
- **The `u`-prefix / base-vs-display denom convention is the documented Cosmos SDK standard**, not a project-specific style choice — see Cosmos SDK ADR-024 ("Coin Metadata") and the SDK's `DenomUnit` type (`denom`, `exponent`, `aliases`), which is precisely the `denom_units` structure already present in this repo's `chain/genesis.json`, just using the old denom names.

No corrections to Sections 1–14 were needed as a result of this check — all technical claims held up against the current upstream documentation.

---

## Recommended execution order

1. **Section 1** (genesis + core config) — the source of truth everything else must match.
2. **Section 2** (chain modules) — fix the validator-reward bug here, not just rename.
3. **Sections 3–4** (backend, relayer) — fix the faucet-denom question here.
4. **Section 5** (frontend) — fix the wallet-config and governance-denom bugs here.
5. **Sections 6–7** (explorer, infra) — cosmetic/display alignment.
6. **Section 8** (docs) — keep ADRs accurate.
7. **Section 9** (tests), plus the additional files in **Section 11** — update last; a fully green test suite after this step is your confirmation the rename is complete and consistent everywhere.
8. **Section 12** (UI display symbol reclassification) — do this pass across `explorer/` and `frontend/` once the backend/API amounts are already correctly denominated, so the UI labels can be matched to real data.
9. **Section 13** (final verification) — run both grep commands. The first must return zero output. The second must return only the two explicitly-excluded hits. If either check turns up anything else, the rename is not done — do not mark this complete based on the AI agent's own claim without re-running these two commands yourself.
