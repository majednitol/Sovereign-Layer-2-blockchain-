# ADR 011: EVM Denomination & Decimal Scaling

## Context & Problem Statement
The EVM standard requires gas tokens to support 18 decimal places for compatibility with wallets (MetaMask, Rainbow) and web3 frameworks. However, standard Cosmos SDK staking and fee calculations typically use 6 decimal places (e.g. `utoken`). We need a solution that accommodates both models without precision loss or display bugs.

## Decision & Design

1. **Dual Denomination Model**:
   - **`utoken` (6 decimals)**: Used for Cosmos SDK transactions, staking, delegations, governance deposits, and native bank commands.
   - **`atoken` (18 decimals)**: Used for EVM transactions, smart contract executions, and displayed as the native network currency in EVM-compatible wallets.
2. **`x/erc20` Conversion**:
   - The `x/erc20` module (bundled with `cosmos/evm`) manages the bi-directional conversion between `utoken` and `atoken`.
   - When a user sends a Cosmos transaction to an EVM address, the funds are scaled up by $10^{12}$ and wrapped. When transferred back, they are scaled down by $10^{12}$ and unwrapped.
   - Genesis is configured with a native token pair registering `utoken` to an ERC-20 token wrapper representation.
3. **Wallet Configuration**:
   - MetaMask and other EVM wallets must register the network currency as `atoken` with 18 decimal places.
   - Users transact in `atoken` on the EVM side (e.g., 1 TOKEN = $10^{18}$ `atoken` / $10^6$ `utoken`).
