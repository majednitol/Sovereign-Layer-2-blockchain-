# Wallet Integration and Setup Guide

This guide describes how to connect Keplr (Cosmos) and MetaMask (EVM + BSC) to the Sovereign L1 Network.

---

## 1. Keplr Wallet Setup (Cosmos L1)

The Sovereign chain is built using the Cosmos SDK. Keplr is the recommended wallet for managing native `CSOV` (`ucsov`) tokens.

### Experimental Chain Suggestion Configuration

If the chain is not registered, dApps can suggest it programmatically using the following configuration:

```json
{
  "chainId": "sovereign-1",
  "chainName": "Sovereign Mainnet",
  "rpc": "https://rpc.mainnet.sovereign.l1:443",
  "rest": "https://api.mainnet.sovereign.l1:443",
  "bip44": {
    "coinType": 60
  },
  "bech32Config": {
    "bech32PrefixAccAddr": "cosmos",
    "bech32PrefixAccPub": "cosmospub",
    "bech32PrefixValAddr": "cosmosvaloper",
    "bech32PrefixValPub": "cosmosvaloperpub",
    "bech32PrefixConsAddr": "cosmosvalcons",
    "bech32PrefixConsPub": "cosmosvalconspub"
  },
  "currencies": [
    {
      "coinDenom": "CSOV",
      "coinMinimalDenom": "ucsov",
      "coinDecimals": 6
    }
  ],
  "feeCurrencies": [
    {
      "coinDenom": "CSOV",
      "coinMinimalDenom": "ucsov",
      "coinDecimals": 6,
      "gasPriceStep": {
        "low": 0.01,
        "average": 0.025,
        "high": 0.04
      }
    }
  ],
  "stakeCurrency": {
    "coinDenom": "CSOV",
    "coinMinimalDenom": "ucsov",
    "coinDecimals": 6
  }
}
```

---

## 2. MetaMask Wallet Setup (Sovereign EVM)

Sovereign features an integrated EVM compatibility layer. Users can connect to this layer using MetaMask.

### Network Configuration Parameters

- **Network Name:** Sovereign EVM Mainnet
- **New RPC URL:** `https://rpc.mainnet.sovereign.l1:8545/evm-rpc`
- **Chain ID:** `7777` (Hex: `0x1E61`)
- **Currency Symbol:** `ESOV`
- **Block Explorer URL:** `https://explorer.sovereign.l1/blockscout`

*Note: Please ensure the corrected Chain ID matches the L1 on-chain parameter configuration.*

---

## 3. MetaMask Wallet Setup (BSC Mainnet)

To interact with the Sovereign LockBox and deposit/withdraw BSC-side tokens, users must connect to the Binance Smart Chain (BSC) Mainnet.

### Network Configuration Parameters

- **Network Name:** Binance Smart Chain Mainnet
- **New RPC URL:** `https://bsc-dataseed.binance.org/`
- **Chain ID:** `56` (Hex: `0x38`)
- **Currency Symbol:** `BNB`
- **Block Explorer URL:** `https://bscscan.com`

---

## 4. WalletConnect Integration

dApps can support mobile wallets via WalletConnect. The namespace configurations are:

```json
{
  "namespaces": {
    "eip155": {
      "methods": ["eth_sendTransaction", "personal_sign"],
      "chains": ["eip155:56", "eip155:7777"]
    },
    "cosmos": {
      "methods": ["cosmos_signDirect", "cosmos_signAmino"],
      "chains": ["cosmos:sovereign-1"]
    }
  }
}
```
*Note: A valid `projectId` from WalletConnect Cloud is required in frontend builds.*
