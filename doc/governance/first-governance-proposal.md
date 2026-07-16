# Proposal 1: Core Network Parameter Adjustment and Software Upgrade

This document outlines the first governance proposal for the Sovereign L1 Network post-launch.

---

## 1. Metadata and Specifications

- **Proposal Title:** Core Network Parameter Adjustment and Software Upgrade
- **Proposal Type:** ParameterChangeProposal / SoftwareUpgradeProposal
- **Proposer Address:** `cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g`
- **Initial Deposit:** 5,000 CSOV (Minimum 2,500 required for deposit period activation)
- **Voting Period:** 3 days (Mainnet default)

---

## 2. Proposal Body (JSON Payload)

This JSON is formatted to be submitted via the command-line interface or frontend.

```json
{
  "messages": [
    {
      "@type": "/cosmos.gov.v1beta1.MsgSubmitProposal",
      "content": {
        "@type": "/cosmos.params.v1beta1.ParameterChangeProposal",
        "title": "Enable EVM Gas Optimizations and Increase Block Gas Limit",
        "description": "This proposal increases the maximum block gas limit from 30,000,000 to 50,000,000 and reduces base EVM tx costs by 10% to accommodate higher throughput applications.",
        "changes": [
          {
            "subspace": "feemarket",
            "key": "MaxBlockGas",
            "value": "\"50000000\""
          },
          {
            "subspace": "feemarket",
            "key": "MinGasPrice",
            "value": "\"0.0025\""
          }
        ]
      },
      "initial_deposit": [
        {
          "denom": "uwsov",
          "amount": "5000000000"
        }
      ],
      "proposer": "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g"
    }
  ],
  "metadata": "ipfs://QmYwAPJzv5CZ1A6tmMecq1A95g955T33SY23SdfhjA",
  "summary": "Core Network Parameter Adjustment and Software Upgrade to increase EVM block gas limit to 50,000,000."
}
```

---

## 3. Execution Instructions

1. **Submit via CLI:**
   Ensure the wallet key is imported and funded:
   ```bash
   chaind tx gov submit-proposal proposal.json --from my-key --chain-id sovereign-1 --node https://rpc.mainnet.sovereign.l1:443
   ```

2. **Deposit to activate voting:**
   If the initial deposit was insufficient to trigger voting:
   ```bash
   chaind tx gov deposit <proposal-id> 5000000000uwsov --from my-key --chain-id sovereign-1 --node https://rpc.mainnet.sovereign.l1:443
   ```

3. **Vote "Yes":**
   ```bash
   chaind tx gov vote <proposal-id> yes --from my-key --chain-id sovereign-1 --node https://rpc.mainnet.sovereign.l1:443
   ```
