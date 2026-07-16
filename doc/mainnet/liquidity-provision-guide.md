# Mainnet Liquidity Provision Guide

This document guides the project owner through the process of bridging treasury-allocated tokens from Sovereign L1 to Binance Smart Chain (BSC) and adding/locking liquidity on PancakeSwap.

---

## 1. Bridging Treasury-Allocated Tokens

To make the token tradeable on BSC, tokens must first be bridged from the Sovereign L1 treasury account to the BSC lockbox.

1. **Query the treasury allocation:**
   Verify that the treasury balance exists on Sovereign L1:
   ```bash
   chaind query bank balances <treasury_address>
   ```

2. **Execute the bridge transaction:**
   Send tokens via the bridge keeper/MsgBridgeInbound (or similar transaction defined in governance parameters) to your BSC recipient address.
   Example:
   ```bash
   chaind tx bridge lock <amount> <bsc_recipient_address> --from treasury --chain-id sovereign-1
   ```

3. **Verify receipts:**
   Check the relayer logs to confirm the tokens have been processed on the BSC side via `LockBox.sol`.

---

## 2. Adding Liquidity on PancakeSwap

1. **Target Pool:** PancakeSwap V2 (WSOV/BNB or WSOV/BUSD)
2. **Minimum Liquidity:** A minimum of $5,000 USD equivalent in BNB or BUSD paired with the earmarked WSOV tokens.
3. **Execution Steps:**
   - Go to [PancakeSwap Liquidity](https://pancakeswap.finance/liquidity).
   - Select BNB/BUSD and paste the WSOV BSC contract address.
   - Enter the ratio of tokens matching the launch price.
   - Click "Supply" and confirm the transaction via MetaMask.

---

## 3. Locking LP Tokens

LP tokens received from PancakeSwap must be locked for at least 6–12 months to prevent premature liquidity removal (rug-pull protection).

1. **Using Team Finance or UNCX Network:**
   - Go to [UNCX Lockers](https://uncx.network/) or [Team Finance](https://www.team.finance/).
   - Connect the wallet holding the PancakeSwap V2 LP tokens.
   - Input the PancakeSwap LP token contract address.
   - Select the lock duration (minimum 6 months).
   - Approve, lock, and pay the transaction fee.
   - Save the lock receipt and transaction hash.

2. **Alternative: Burning LP Tokens:**
   To permanently burn liquidity, send PancakeSwap V2 LP tokens to the BSC dead address:
   `0x000000000000000000000000000000000000dEaD`
