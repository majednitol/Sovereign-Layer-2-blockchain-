# Custodian Key Fingerprint Registry

> ⚠️ **CONFIDENTIALITY & SECURITY WARNING**: Raw public and private keys must never be committed to this or any public repository. This registry tracks custodian identities and key verify-ability using cryptographic hash fingerprints of their public keys (SHA-256).

## Real Key Verification Command

To generate a SHA-256 fingerprint for a Bechs32 pubkey or JSON keyfile:
```bash
# For a raw public key string:
echo -n "cosmospub1..." | sha256sum | awk '{print $1}'

# For a keyring exported key:
chaind keys parse <key-address> --output json | jq -r '.pubkey' | sha256sum | awk '{print $1}'
```

## Roster Registry Table

| Slot | Role / Entity | Key Fingerprint (SHA-256) | Hardware Device / Custody | Verification Date | Holder Name / Signature |
|:---:|:---|:---|:---|:---|:---|
| **1** | Operations Lead | `[PENDING]` | Ledger Nano S+ / Air-gapped | `[PENDING]` | `[PENDING]` |
| **2** | Security Officer | `[PENDING]` | Ledger Nano X / YubiKey | `[PENDING]` | `[PENDING]` |
| **3** | Technical Architect | `[PENDING]` | YubiHSM 2 / Cloud KMS | `[PENDING]` | `[PENDING]` |
| **4** | Validator Representative 1 | `[PENDING]` | Ledger Nano S+ | `[PENDING]` | `[PENDING]` |
| **5** | Foundation Custodian | `[PENDING]` | Cold wallet offline backup | `[PENDING]` | `[PENDING]` |
| **6** | Validator Representative 2 | `[PENDING]` | Ledger Nano S+ | `[PENDING]` | `[PENDING]` |
| **7** | Legal Trustee | `[PENDING]` | Institutional Custody | `[PENDING]` | `[PENDING]` |

## Recovery & Emergency Workflows

In case of a key rotation or custodian change, a new fingerprint must be registered by executing a `MsgUpdateParams` transaction through active governance (5-of-7 multisig vote). The old fingerprint must be deleted, and a new registry table row added below.

---
*Created: 2026-07-15*
