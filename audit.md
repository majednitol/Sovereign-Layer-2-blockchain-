You are an independent senior blockchain security auditor and adversarial verification engineer.

Your task is to perform a DEEP, INDEPENDENT, CLAIM-BY-CLAIM security audit of this entire repository.

IMPORTANT:
- Do NOT trust any previous audit report.
- Do NOT trust previous AI-generated conclusions.
- Do NOT assume that passing tests prove security.
- Do NOT merely run existing tests and repeat their results.
- Do NOT modify code before first understanding and testing the existing implementation.
- Treat every security claim as UNPROVEN until independently verified.
- You must actively search for vulnerabilities that existing tests do not cover.
- You must distinguish between:
  1. Proven by direct execution
  2. Proven by deterministic source-code reasoning
  3. Partially tested
  4. Operational assumption
  5. Unproven
  6. False or overstated claim

REPOSITORY:
[INSERT REPOSITORY PATH]

SYSTEM ARCHITECTURE:
- Sovereign L1 blockchain
- Cosmos SDK / CometBFT
- EVM compatibility
- Solidity LockBox bridge contract on BSC
- Off-chain relayer
- BSC event indexer
- NATS messaging
- x/bridge
- x/precisebank
- x/erc20
- x/evm
- Native denomination: ucsov
- Wrapped bridge asset: uwsov / WSOV
- EVM representation: 18-decimal token/native unit
- Supply cap: 300,000,000 WSOV
- Relayer quorum-based unlock authorization

==================================================
PHASE 0 — REPOSITORY DISCOVERY
==================================================

Before making any security conclusion:

1. Map the complete repository structure.
2. Identify:
   - All Solidity contracts
   - All Go modules
   - All bridge message types
   - All keeper functions
   - All module accounts
   - All mint/burn paths
   - All EVM ↔ Cosmos denomination conversion paths
   - All relayer components
   - All BSC indexer components
   - All NATS subjects and message flows
   - All configuration and genesis files
   - All deployment scripts
   - All environment variables
   - All tests and fuzz tests

3. Create a trust-boundary diagram in text:

   BSC User
      ↓
   LockBox.sol
      ↓
   BSC Event
      ↓
   Indexer
      ↓
   NATS / Message Bus
      ↓
   Relayer
      ↓
   Relayer Signatures
      ↓
   Sovereign MsgBridgeIn
      ↓
   x/bridge
      ↓
   Mint / Supply Accounting

   Also map the reverse BridgeOut path.

4. Identify every external trust boundary and every component that can be compromised.

==================================================
PHASE 1 — BUILD A SECURITY CLAIM REGISTER
==================================================

Extract every security claim from:
- Existing audit reports
- README files
- Comments
- Test names
- Documentation
- TODOs
- Configuration
- Previous AI conclusions

For every claim, create:

| ID | Claim | Exact Code Location | Security Property | Evidence Required | Current Status |

Do not accept vague claims such as:
- "safe"
- "secure"
- "production-ready"
- "replay protected"
- "atomic"
- "audited"
- "lossless"

Convert each into a precise, falsifiable security property.

Example:

Weak claim:
"Bridge is replay protected."

Precise claims:
- Can the exact same BSC nonce be unlocked twice?
- Can the same event be submitted twice to Sovereign?
- Can the same signature be reused with a different recipient?
- Can the same signature be reused with a different amount?
- Can the same signature be reused on another chain?
- Can the same signature be reused on another LockBox deployment?
- Can nonce storage collide?
- Can nonce arithmetic overflow?
- Can a failed transaction permanently consume a nonce?

==================================================
PHASE 2 — RUN ALL TESTS INDEPENDENTLY
==================================================

Run all relevant tests independently.

Solidity:
- forge build
- forge test -vvv
- forge test --gas-report
- forge test with maximum practical fuzz runs
- invariant tests if available
- mutation testing if feasible

Go:
- go test ./...
- go test -race ./...
- go vet ./...
- static analysis where available
- package-specific tests
- integration tests
- application-level tests

Do NOT only report "tests passed."

For each test:
- Identify exactly what it proves.
- Identify what it does NOT prove.
- Identify whether the test actually exercises production code paths.
- Identify mocks and shortcuts.
- Identify whether the test environment differs from mainnet.

IMPORTANT:
A passing test is evidence only for the exact behavior exercised by that test.

==================================================
PHASE 3 — ADVERSARIAL BRIDGE ANALYSIS
==================================================

Analyze the complete BridgeIn and BridgeOut lifecycle.

Test or reason about:

A. Replay attacks
- Same nonce
- Same event
- Same signature
- Same transaction
- Same payload
- Same payload with reordered signatures
- Same payload with duplicate signatures
- Same payload across different blocks
- Same payload after relayer restart
- Same payload after database rollback

B. Cross-domain replay
- BSC chain A → BSC chain B
- LockBox deployment A → LockBox deployment B
- Testnet → Mainnet
- Old contract → New contract
- BridgeIn signature → BridgeOut
- BridgeOut signature → BridgeIn

C. Signature attacks
- Duplicate signers
- Signature ordering
- ECDSA malleability
- High-S signatures
- Invalid v values
- Zero-address recovery
- Signature truncation
- Signature concatenation
- Wrong signer
- Quorum boundary conditions
- Threshold = 0
- Threshold > signer count
- Duplicate configured relayers
- Relayer removal after signature creation
- Relayer rotation
- Quorum changes during pending messages

D. Nonce attacks
- nonce = 0
- nonce = 1
- nonce = 255
- nonce = 256
- nonce = 257
- max uint256
- nonce bitmap collision
- nonce reuse after upgrade
- nonce storage reset
- nonce storage migration

==================================================
PHASE 4 — SOLIDITY LOCKBOX DEEP AUDIT
==================================================

Audit every function line-by-line.

Check:

1. Reentrancy
2. Checks-effects-interactions
3. ERC20 return-value handling
4. Fee-on-transfer behavior
5. Rebasing token behavior
6. Token decimals mismatch
7. Token replacement
8. Token rescue functions
9. Pause logic
10. Admin compromise
11. Gnosis Safe ownership
12. Upgradeability
13. Initialization
14. Constructor configuration
15. Zero addresses
16. Zero amounts
17. Maximum amounts
18. Integer overflow/underflow
19. Gas griefing
20. Denial of service
21. Block timestamp/block number dependence
22. Rate-limit bypass
23. Rate-limit reset behavior
24. Emergency pause race conditions
25. Token transfer failure behavior
26. Malicious ERC20 callbacks
27. ERC777-style hooks
28. Self-destruct/token destruction edge cases
29. Contract balance mismatch
30. Accounting invariant violations

Explicitly test these invariants:

INVARIANT 1:
totalReleasedToUsers <= totalLocked

INVARIANT 2:
contract token balance >= totalLocked - totalReleasedToUsers

INVARIANT 3:
Every nonce is processed at most once.

INVARIANT 4:
Every successful unlock requires the correct quorum.

INVARIANT 5:
No valid signature can authorize a different:
- chain ID
- contract
- recipient
- amount
- nonce

INVARIANT 6:
A failed unlock must not create partial accounting state.

==================================================
PHASE 5 — COSMOS SDK / BRIDGE MODULE AUDIT
==================================================

Audit:

- Msg validation
- Signer validation
- Address conversion
- Bech32 ↔ EVM conversion
- Account ownership
- Module account permissions
- Mint permissions
- Burn permissions
- Supply cap enforcement
- Supply cap arithmetic
- Integer overflow
- Integer underflow
- Zero-value messages
- Negative values if possible
- Denomination validation
- IBC or cross-module interactions
- BeginBlocker
- EndBlocker
- Genesis import/export
- State migration
- Authority permissions
- Governance parameter changes

Verify:

1. Can a user mint?
2. Can a user burn another user's funds?
3. Can bridge module mint more than the cap?
4. Can bridge module burn without reducing accounting?
5. Can `cosmos_minted` become negative?
6. Can `cosmos_minted` desynchronize from actual supply?
7. Can a failed transaction partially modify supply accounting?
8. Can module permissions change after genesis?
9. Can governance modify bridge security parameters dangerously?

==================================================
PHASE 6 — EVM / PRECISION ACCOUNTING AUDIT
==================================================

Independently derive the mathematical model.

Define:

1 ucsov = 10^12 EVM wei

Then verify all paths:

Cosmos → EVM
EVM → Cosmos
Native transfer
Contract transfer
Gas payment
Fee deduction
Refund
Revert
Out-of-gas
Nested calls
Create contract
Selfdestruct
Precompiles
EVM logs
StateDB rollback
Commit
Multi-block execution

Test:

- 1 wei
- 10^12 - 1 wei
- 10^12 wei
- 10^12 + 1 wei
- maximum uint256
- repeated fractional transfers
- random fuzz transfers
- revert after fractional transfer
- OOG after fractional transfer
- nested transaction failure

Prove or disprove:

GLOBAL SUPPLY CONSERVATION

Total canonical supply =
Cosmos bank supply
+ precisebank fractional reserve
+ EVM represented balance
+ ERC20 escrow
+ bridge escrow
+ other legitimate locations

If you cannot mathematically prove the invariant, classify it as UNPROVEN.

==================================================
PHASE 7 — ERC20 CONVERSION AUDIT
==================================================

Analyze:

ConvertCoin
ConvertERC20
Mint
Burn
Escrow
Release

Check:

- Can ConvertCoin be called twice?
- Can ConvertERC20 release more than escrowed?
- Can ERC20 supply exceed escrow?
- Can native supply exceed canonical supply?
- Can token decimals cause rounding loss?
- Can conversion occur during a revert?
- Can conversion be front-run?
- Can module account balances be bypassed?
- Can a malicious ERC20 contract break assumptions?

Prove:

ERC20 total supply <= escrowed native value

==================================================
PHASE 8 — RELAYER AND INDEXER ADVERSARIAL AUDIT
==================================================

Treat all off-chain components as potentially compromised.

Analyze:

- Private key storage
- Key rotation
- Quorum configuration
- Relayer identity
- Database persistence
- Crash recovery
- Replay processing
- Event confirmation depth
- Chain reorganization
- RPC inconsistency
- RPC censorship
- RPC equivocation
- NATS authentication
- NATS TLS
- Message replay
- Message duplication
- Message tampering
- Message ordering
- Network partition
- Split brain
- Relayer disagreement
- Stale events
- Database rollback

IMPORTANT:

Do not say "50 confirmations are required" merely because a parameter exists.

Prove:
- Where confirmation depth is enforced.
- Whether it is actually executed.
- Whether it is configurable.
- Whether large and small transfers differ.
- Whether the relayer can bypass it.
- Whether a reorg after confirmation is handled.

==================================================
PHASE 9 — CONFIGURATION AND GENESIS AUDIT
==================================================

Search the entire repository for:

aesov
ucsov
uwsov
WSOV
CSOV
EVMDenom
SupplyCap

Check:
- Source code
- Genesis
- Scripts
- Environment files
- Docker
- Kubernetes
- CI/CD
- Test fixtures
- Deployment scripts
- Documentation

Detect:
- Case-sensitive mismatches
- Legacy denominations
- Wrong decimals
- Wrong chain ID
- Wrong contract address
- Wrong quorum
- Wrong supply cap
- Wrong token address
- Wrong RPC endpoint

Do not assume source code and production configuration match.

==================================================
PHASE 10 — PROPERTY-BASED AND FUZZ TESTING
==================================================

Create new tests where necessary.

Do not only test expected success paths.

Generate adversarial properties:

- Any nonce can be processed at most once.
- Any valid unlock requires quorum.
- Duplicate signatures never increase quorum.
- Any failed operation leaves state unchanged.
- Total supply never exceeds cap.
- ERC20 escrow always equals represented supply.
- Fractional accounting never loses value.
- Revert always restores state.
- Out-of-gas always restores state.
- Invalid addresses always fail.
- Invalid denominations always fail.

Use:
- Fuzzing
- Stateful fuzzing
- Invariant testing
- Differential testing
- Randomized sequences
- Boundary testing

==================================================
PHASE 11 — NEGATIVE SECURITY TESTING
==================================================

For every security control, deliberately attempt to bypass it.

Examples:

- Remove one signature.
- Duplicate one signature 100 times.
- Change one byte of the payload.
- Change recipient.
- Change amount.
- Change nonce.
- Change chain ID.
- Change contract address.
- Use maximum integer.
- Use zero.
- Use empty arrays.
- Reorder arrays.
- Replay after restart.
- Replay after rollback.
- Replay after pause/unpause.
- Replay after relayer rotation.
- Trigger failure after state mutation.
- Trigger failure during external call.

Every bypass attempt must be recorded.

==================================================
PHASE 12 — PREVIOUS AUDIT CHALLENGE
==================================================

For every previous audit claim, ask:

1. What exact code proves this?
2. What exact test proves this?
3. Does the test cover production behavior?
4. Is the claim stronger than the evidence?
5. Is the claim actually an operational assumption?
6. Could the claim be false despite all tests passing?

Explicitly identify:
- False claims
- Overstated claims
- Unsupported claims
- Claims proven only under assumptions
- Claims that require external audit

==================================================
PHASE 13 — MAINNET THREAT MODEL
==================================================

Assume the following attackers:

1. Ordinary user
2. Malicious user
3. Malicious relayer
4. Compromised single relayer
5. Compromised quorum of relayers
6. Malicious BSC RPC provider
7. Malicious indexer
8. Compromised NATS server
9. Compromised database
10. Malicious ERC20 token
11. Malicious governance participant
12. Malicious validator
13. Chain reorganization attacker
14. Network attacker
15. Supply/accounting exploit developer

For each attacker, determine:
- What they can do
- What they cannot do
- Maximum financial impact
- Required preconditions
- Detection mechanism
- Recovery mechanism

==================================================
PHASE 14 — DO NOT GIVE A FALSE "PRODUCTION READY"
==================================================

Use this classification logic:

NOT READY:
- Any confirmed exploitable critical/high vulnerability
- Supply conservation is not established
- Bridge replay protection is incomplete
- Quorum bypass exists
- Unauthorized minting exists
- Mainnet configuration is inconsistent

READY ONLY AFTER REMEDIATION:
- Code is secure based on available evidence
- External audit remains outstanding
- Operational controls remain unverified

PRODUCTION READY:
Only if:
- Critical security properties are independently verified
- Mainnet configuration is verified
- Operational controls are actually deployed and tested
- No high-severity unmitigated risks remain

If evidence is insufficient, say:
"UNPROVEN — additional evidence required."

Never upgrade "unproven" to "secure."

==================================================
REQUIRED FINAL REPORT
==================================================

Return ONLY these sections:

# A. Claim-by-Claim Verification

For every major claim:

| Claim | Evidence | What Is Actually Proven | What Is Not Proven | Status |

Use statuses:
- PROVEN BY DIRECT EXECUTION
- PROVEN BY SOURCE CODE
- PARTIALLY PROVEN
- OPERATIONAL ASSUMPTION
- UNPROVEN
- FALSE / OVERSTATED

# B. Newly Discovered or Previously Overlooked Vulnerabilities

For each:

- Vulnerability
- Severity
- Attack Preconditions
- Attack Steps
- Technical Root Cause
- Impact
- Exploitability
- Remediation
- Verification Status

# C. Incorrect or Overstated Claims in Previous Audits

Be direct and specific.

# D. Remaining Unproven Assumptions

Separate:
- Security assumptions
- Infrastructure assumptions
- Operational assumptions
- Governance assumptions
- Third-party dependencies

# E. Required Code Changes

Only include changes genuinely required for security.

# F. Required Operational Remediations

Include:
- Key management
- Finality
- Monitoring
- Alerting
- Incident response
- Backup/recovery
- NATS security
- RPC redundancy
- Deployment controls

# G. Adversarial Attack Matrix

Include at minimum:

| Attack | Preconditions | Attack Steps | Expected | Actual | Status | Evidence | Limitations |

# H. Final Mainnet Readiness Classification

Choose exactly one:

NOT MAINNET READY

PRODUCTION-READY AFTER SPECIFIC REMEDIATIONS

PRODUCTION READY

Then explain the exact reasons.

# I. Top 5 Remaining Risks Ranked by Severity

For each:

1. Risk
2. Severity
3. Likelihood
4. Impact
5. Why it remains
6. Exact remediation
7. Evidence required to close it

FINAL RULE:

Do not produce a reassuring report.

Produce an adversarial report.

Passing tests are evidence, not proof of complete security.

A vulnerability is not "fixed" merely because a test passes.

An operational assumption is not a verified security control.

A source-code parameter is not proof that an off-chain service enforces it.

If the evidence is insufficient, explicitly write:

"UNPROVEN — additional evidence required."

Your goal is to find the strongest attack that can still succeed.