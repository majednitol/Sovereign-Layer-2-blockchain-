# DEEP IMPLEMENTATION VERIFICATION PROMPT — UNIFIED COSMOS + EVM ACCOUNT

You are a senior blockchain protocol auditor specializing in:

* Cosmos SDK
* Cosmos EVM / Ethermint
* Ethereum cryptography
* Keplr / Leap wallet integration
* MetaMask / EVM JSON-RPC
* BIP-32 / BIP-44 derivation
* `secp256k1` vs `eth_secp256k1`
* Cosmos transaction signing
* EVM transaction signing
* Blockchain security and integration testing

Repository:

`/Users/majedurrahman/Sovereign`

## PRIMARY OBJECTIVE

Do NOT assume that the previous audit is correct.

Your task is to determine whether the following architecture is **TRULY IMPLEMENTED and WORKING in the actual repository**, not merely theoretically possible:

```text
                    ONE USER ACCOUNT
                         │
          ┌──────────────┴──────────────┐
          │                             │
       MetaMask                    Keplr / Leap
          │                             │
      EVM Wallet                  Cosmos Wallet
          │                             │
       0xABC...                 cosmos1...
          │                             │
          └────── SAME 20-BYTE ─────────┘
                  ADDRESS PAYLOAD
```

The target architecture is:

* MetaMask handles EVM transactions.
* Keplr or Leap handles Cosmos SDK transactions.
* Both wallets can represent the same underlying cryptographic account.
* The EVM `0x...` address and Cosmos `cosmos1...` address contain the same 20-byte address payload.
* The user can use both wallets without accidentally controlling different accounts.
* The architecture must work in actual wallet software, not only in theoretical cryptography.

---

# CRITICAL RULE

You must distinguish between:

1. **THEORETICALLY POSSIBLE**
2. **IMPLEMENTED IN SOURCE CODE**
3. **ACTUALLY VERIFIED AT RUNTIME**
4. **ACTUALLY VERIFIED WITH REAL WALLET SOFTWARE**

Never classify something as "PROVEN" merely because a configuration file exists.

A configuration file is NOT proof that Keplr or Leap actually derives the same private key.

---

# PHASE 1 — REPOSITORY DISCOVERY

Before making any conclusion:

1. Inspect the complete repository structure.

2. Find all relevant files related to:

   * Cosmos SDK app initialization
   * EVM initialization
   * Ethermint / Cosmos EVM
   * keyring configuration
   * HD path configuration
   * BIP-44
   * account codecs
   * Bech32
   * `eth_secp256k1`
   * standard Cosmos `secp256k1`
   * Keplr
   * Leap
   * WalletConnect
   * MetaMask
   * EVM JSON-RPC
   * Cosmos REST
   * Cosmos gRPC
   * frontend wallet adapters
   * chain registry
   * wallet suggestions

3. Do not rely only on filenames.

4. Search the entire codebase for:

   * `eth_secp256k1`
   * `EthSecp256k1`
   * `secp256k1`
   * `coinType`
   * `bip44`
   * `eth-address-gen`
   * `eth-key-sign`
   * `hdPath`
   * `m/44'/60'`
   * `m/44'/118'`
   * `NewEvmCodec`
   * `Bech32PrefixAccAddr`
   * `cosmos_signDirect`
   * `cosmos_signAmino`
   * `eth_sendTransaction`
   * `eth_signTypedData_v4`
   * `personal_sign`
   * `wallet_addEthereumChain`
   * `experimentalSuggestChain`
   * `keplr`
   * `leap`

---

# PHASE 2 — CRYPTOGRAPHIC DERIVATION VERIFICATION

Determine EXACTLY how an account is derived.

You must answer:

## A. What private key is used?

Identify:

* private key format
* curve
* public key format
* compressed/uncompressed public key
* derivation path
* coin type
* hash algorithm

## B. What algorithm does the Cosmos chain expect?

Verify whether the chain uses:

```text
Cosmos secp256k1:
SHA256(pubkey)
→ RIPEMD160
→ 20 bytes
```

or:

```text
Ethereum eth_secp256k1:
Keccak256(uncompressed_pubkey_without_0x04)
→ last 20 bytes
```

Do not confuse the following:

```text
Cosmos SDK secp256k1
```

with:

```text
Ethereum eth_secp256k1
```

They are NOT automatically equivalent.

## C. Verify the actual address codec

Inspect the actual implementation of:

```go
evmaddress.NewEvmCodec(...)
```

Do not assume what it does.

Prove:

```text
EVM address:
0xABCDEF...

↓ decode

20-byte payload:
[20 bytes]

↓ Bech32 encode with prefix "cosmos"

Cosmos address:
cosmos1...

↓ decode

same 20-byte payload:
[20 bytes]
```

Create an executable test that proves:

```text
decodeBech32(cosmosAddress) == decodeHex(evmAddress)
```

---

# PHASE 3 — REAL WALLET DERIVATION COMPATIBILITY

This is the MOST IMPORTANT PHASE.

Determine whether the following statement is actually true:

> "The same MetaMask seed/private key can be imported into Keplr or Leap and produce the same EVM address."

Do NOT assume this.

You must verify:

## MetaMask

Determine:

* seed derivation path
* private key derivation
* curve
* public key algorithm
* Ethereum address generation

## Keplr

Determine from actual wallet implementation/documentation:

* whether `coinType: 60` is supported
* whether `eth-address-gen` changes address derivation
* whether `eth-key-sign` changes signing behavior
* whether Keplr can import the SAME mnemonic used by MetaMask
* whether Keplr derives the exact same private key at the same path
* whether Keplr derives the same public key
* whether Keplr derives the same 20-byte Ethereum address

## Leap

Perform the same verification for Leap.

### IMPORTANT

A chain suggestion like:

```json
{
  "bip44": {
    "coinType": 60
  },
  "features": [
    "eth-address-gen",
    "eth-key-sign"
  ]
}
```

is NOT sufficient proof.

You must verify actual wallet behavior.

If real wallet behavior cannot be executed in this environment, explicitly classify the claim as:

`NOT RUNTIME VERIFIED`

Do not call it proven.

---

# PHASE 4 — ACTUAL WALLET IMPORT TEST

If possible, perform a deterministic test using a known test mnemonic.

Use a non-production test mnemonic only.

Example:

```text
test test test test test test test test test test test junk
```

or another clearly marked development-only mnemonic.

Derive the account using:

## MetaMask-compatible Ethereum derivation

```text
m/44'/60'/0'/0/0
```

Then derive using the actual Cosmos wallet library/configuration used by the project.

Compare:

```text
Private Key A
Private Key B

Public Key A
Public Key B

EVM Address A
EVM Address B

Cosmos 20-byte payload A
Cosmos 20-byte payload B
```

The following must be identical:

```text
Private Key A == Private Key B
Public Key A == Public Key B
EVM Address A == EVM Address B
Cosmos raw 20-byte payload == EVM raw 20-byte payload
```

If only the final address matches but the private keys differ, explain why.

If the wallet software cannot be tested, state exactly:

```text
Cryptographically plausible but not runtime verified with real wallet software.
```

---

# PHASE 5 — ACTUAL METAMASK INTEGRATION

Verify whether the frontend can actually:

1. Detect MetaMask.
2. Request accounts.
3. Read the account.
4. Read EVM chain ID.
5. Switch to EVM chain ID `7777`.
6. Send native EVM transaction.
7. Call a smart contract.
8. Sign an EVM message.
9. Sign EIP-712 typed data if claimed.

Test actual JSON-RPC methods:

```text
eth_requestAccounts
eth_accounts
eth_chainId
wallet_switchEthereumChain
eth_sendTransaction
personal_sign
eth_signTypedData_v4
```

For every method:

* Find implementation.
* Execute if possible.
* Record exact result.
* Record exact failure if unsupported.

Do not claim support merely because the chain supports the method.

---

# PHASE 6 — ACTUAL KEPLR / LEAP INTEGRATION

Verify whether the frontend can actually:

1. Detect Keplr.
2. Detect Leap.
3. Suggest the chain.
4. Add the chain.
5. Enable the chain.
6. Derive the account.
7. Read the account.
8. Sign a Cosmos transaction.
9. Broadcast a Cosmos transaction.

Test:

```text
experimentalSuggestChain
keplr.enable
getKey
signDirect
signAmino
sendTx
```

For Leap, test the equivalent APIs.

The final report must clearly distinguish:

```text
Backend support
Frontend support
Wallet API support
Actual browser runtime support
```

---

# PHASE 7 — ACCOUNT MATCHING SECURITY

Implement or verify this exact security property:

```text
MetaMask Address:
0xABCDEF...

Keplr Address:
cosmos1...

↓ decode both

MetaMask raw 20 bytes
==
Keplr Bech32 decoded raw 20 bytes
```

The frontend must refuse to treat the accounts as unified if:

```text
MetaMaskRawAddress != KeplrRawAddress
```

Verify:

* case normalization
* Bech32 decoding
* wrong prefix
* invalid Bech32 checksum
* different account
* different derivation path

Create tests for:

```text
same account → ACCEPT
different account → REJECT
wrong prefix → REJECT or explicitly handle
invalid Bech32 → REJECT
```

---

# PHASE 8 — TRANSACTION ARCHITECTURE

Verify the actual transaction flow.

## MetaMask path

```text
MetaMask
↓
eth_sendTransaction
↓
EVM JSON-RPC
↓
MsgEthereumTx
↓
EVM AnteHandler
↓
EVM execution
↓
StateDB
```

Prove this using actual source code and runtime tests.

## Keplr / Leap path

```text
Keplr / Leap
↓
cosmos_signDirect
↓
SignDoc
↓
Cosmos SDK transaction
↓
AnteHandler
↓
MsgSend / MsgBridgeOut / staking
↓
State transition
```

Prove this using actual source code and runtime tests.

---

# PHASE 9 — SAME ACCOUNT DOES NOT MEAN SAME WALLET

Explicitly verify and explain:

```text
Same cryptographic account
≠
Same wallet software
≠
Same transaction encoding
≠
Same signing API
```

Determine whether this architecture requires:

```text
MetaMask + Keplr/Leap
```

or whether a single wallet can actually perform both transaction types.

Do not recommend custom EIP-712 Cosmos signing unless the repository genuinely implements it and the implementation is tested.

---

# PHASE 10 — CRITICAL FAILURE CONDITIONS

The following findings MUST be reported as failures:

## Failure 1

Keplr/Leap uses standard Cosmos derivation:

```text
m/44'/118'
```

while MetaMask uses:

```text
m/44'/60'
```

and the resulting accounts differ.

## Failure 2

The chain claims `eth_secp256k1` but Cosmos transactions cannot be signed by the corresponding wallet.

## Failure 3

The chain uses an EVM address codec but account creation uses a different cryptographic address derivation.

## Failure 4

The frontend automatically assumes:

```text
MetaMask account == Keplr account
```

without checking the raw 20-byte payload.

## Failure 5

The chain suggestion advertises:

```text
eth-address-gen
eth-key-sign
```

but the actual wallet does not support or honor those features.

## Failure 6

The same mnemonic is imported into MetaMask and Keplr/Leap but produces different accounts while the project claims unified accounts.

---

# REQUIRED TESTS

Create and execute tests for:

1. EVM hex ↔ Cosmos Bech32 conversion.
2. Same 20-byte payload.
3. Different account detection.
4. Coin type 60 derivation.
5. Coin type 118 derivation comparison.
6. `eth_secp256k1` vs standard `secp256k1`.
7. MetaMask EVM transaction.
8. Cosmos `MsgSend`.
9. Cosmos `MsgBridgeOut`.
10. Keplr/Leap account matching.
11. Wrong wallet account rejection.
12. Wrong chain ID rejection.
13. Wrong Cosmos chain ID rejection.
14. Bech32 checksum failure.
15. EVM address checksum normalization.

---

# REQUIRED FINAL REPORT FORMAT

## 1. Executive Verdict

Use exactly one:

```text
FULLY IMPLEMENTED AND RUNTIME VERIFIED
```

```text
IMPLEMENTED BUT NOT FULLY RUNTIME VERIFIED
```

```text
PARTIALLY IMPLEMENTED
```

```text
NOT IMPLEMENTED
```

---

## 2. Truth Table

| Claim | Theoretically Correct | Source Code Exists | Automated Test | Runtime Verified | Real Wallet Verified | Final Status |
| ----- | --------------------- | ------------------ | -------------- | ---------------- | -------------------- | ------------ |

Every claim must have a separate row.

---

## 3. Cryptographic Derivation

Show exact:

```text
Mnemonic
↓
BIP-39
↓
BIP-32
↓
BIP-44 path
↓
Private Key
↓
Public Key
↓
Hash
↓
20-byte address
```

For MetaMask, Keplr, and Leap separately.

---

## 4. Exact Account Comparison

Show:

```text
MetaMask:
Private Key:
Public Key:
EVM Address:

Keplr:
Private Key:
Public Key:
Cosmos Address:

Leap:
Private Key:
Public Key:
Cosmos Address:
```

Mask sensitive values where appropriate.

---

## 5. Wallet Compatibility Verdict

Answer directly:

```text
Can the same MetaMask mnemonic be imported into Keplr?
YES / NO / NOT VERIFIED

Can it produce the same private key?
YES / NO / NOT VERIFIED

Can it produce the same EVM address?
YES / NO / NOT VERIFIED

Can Keplr sign Cosmos SDK transactions?
YES / NO / NOT VERIFIED

Can MetaMask sign Cosmos SDK transactions?
YES / NO / NOT VERIFIED
```

---

## 6. Required Code Changes

List only changes that are genuinely required.

For every change include:

```text
File
Function
Current Problem
Required Fix
Security Impact
Test Required
```

---

## 7. Required Frontend Changes

Verify whether these are implemented:

* wallet detection
* chain suggestion
* wallet connection
* account matching
* mismatch rejection
* EVM transaction flow
* Cosmos transaction flow
* network switching
* error handling

---

## 8. Security Findings

Rank:

```text
CRITICAL
HIGH
MEDIUM
LOW
INFORMATIONAL
```

Do not inflate severity.

---

## 9. Final Answer to the Core Question

Answer in simple language:

> If a user connects MetaMask and Keplr/Leap, do they actually control the same account?

The answer must be one of:

```text
YES — PROVEN WITH RUNTIME WALLET TESTS
```

```text
YES — CRYPTOGRAPHICALLY PROVEN BUT REAL WALLET RUNTIME NOT VERIFIED
```

```text
NO — THEY CONTROL DIFFERENT ACCOUNTS
```

```text
NOT ENOUGH EVIDENCE
```

---

## 10. ABSOLUTE AUDIT RULE

Never write:

```text
PROVEN
```

when you only found:

* a config file
* a chain suggestion
* a theoretical derivation
* a source-code possibility
* a documentation claim

Use:

```text
SOURCE-CODE VERIFIED
```

only when the code implements it.

Use:

```text
AUTOMATED-TEST VERIFIED
```

only when a test actually executes it.

Use:

```text
RUNTIME VERIFIED
```

only when the actual running system executes it.

Use:

```text
REAL-WALLET VERIFIED
```

only when MetaMask, Keplr, or Leap behavior is actually tested.

Your final report must be evidence-based, skeptical, and must identify any difference between the previous audit's claims and what is actually implemented.
