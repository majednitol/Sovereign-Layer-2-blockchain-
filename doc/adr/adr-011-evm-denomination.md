# ADR 011: EVM Denomination & Decimal Scaling

## Context & Problem Statement
The EVM standard requires gas tokens to support 18 decimal places for compatibility with wallets (MetaMask, Rainbow) and web3 frameworks. However, standard Cosmos SDK staking and fee calculations typically use 6 decimal places. We need a solution that accommodates both models without precision loss or display bugs, while distinguishing the native staking token (CSOV), the EVM gas token (ESOV), and the bridge asset (WSOV).

## Decision & Design

1. **Three-Denomination Model**:
   - **CSOV (`ucsov`, 6 decimals)**: The native Cosmos SDK staking, governance, and fee token. Used for Cosmos-side delegations, voting, and tx fees.
   - **ESOV (`aesov`, 18 decimals)**: The native gas token for the embedded EVM. Used for EVM transactions, smart contract executions, and displayed as the network currency in EVM-compatible wallets (MetaMask).
   - **WSOV (`uwsov`, 6 decimals)**: The bridge-minted token, representing BSC-escrowed SOV tokens locked in the bridge lockbox.

2. **`x/erc20` Conversion**:
   - The `x/erc20` module manages the bi-directional conversion between `ucsov` and `aesov`.
   - When a user sends a Cosmos transaction to an EVM address, the funds are scaled up by $10^{12}$ and wrapped. When transferred back, they are scaled down by $10^{12}$ and unwrapped.
   - Genesis is configured with a native token pair registering `ucsov` to an ERC-20 token wrapper representation.

3. **Wallet Configuration**:
   - MetaMask and other EVM wallets must register the network currency as `ESOV` (under base denom `aesov`) with 18 decimal places.
   - Users transact in `aesov` on the EVM side (e.g., 1 CSOV = $10^{18}$ `aesov` / $10^6$ `ucsov`).
