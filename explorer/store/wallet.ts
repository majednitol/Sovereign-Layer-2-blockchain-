import { create } from "zustand";

const COSMOS_CHAIN_ID =
  process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1";
const COSMOS_CHAIN_NAME =
  process.env.NEXT_PUBLIC_COSMOS_CHAIN_NAME || "Sovereign L1";
const COSMOS_RPC =
  process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "https://rpc.yourchain.io";
const COSMOS_REST =
  process.env.NEXT_PUBLIC_COSMOS_REST || "https://lcd.yourchain.io:1317";
const SYMBOL = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "SLT";
const MINIMAL_DENOM = "u" + SYMBOL.toLowerCase();

interface WalletState {
  walletType: string | null;
  connected: boolean;
  address: string | null;
  connectWallet: (type: "keplr" | "metamask" | "leap" | "cosmostation") => Promise<void>;
  disconnectWallet: () => void;
}

/**
 * Suggest the Sovereign chain to Keplr / Leap / Cosmostation.
 */
async function suggestCosmosChain(keplrLike: any): Promise<void> {
  await keplrLike.experimentalSuggestChain({
    chainId: COSMOS_CHAIN_ID,
    chainName: COSMOS_CHAIN_NAME,
    rpc: COSMOS_RPC,
    rest: COSMOS_REST,
    bip44: { coinType: 118 },
    bech32Config: {
      bech32PrefixAccAddr: "sovereign",
      bech32PrefixAccPub: "sovereignpub",
      bech32PrefixValAddr: "sovereignvaloper",
      bech32PrefixValPub: "sovereignvaloperpub",
      bech32PrefixConsAddr: "sovereignvalcons",
      bech32PrefixConsPub: "sovereignvalconspub",
    },
    currencies: [
      { coinDenom: SYMBOL, coinMinimalDenom: MINIMAL_DENOM, coinDecimals: 6 },
    ],
    feeCurrencies: [
      {
        coinDenom: SYMBOL,
        coinMinimalDenom: MINIMAL_DENOM,
        coinDecimals: 6,
        gasPriceStep: { low: 0.01, average: 0.025, high: 0.04 },
      },
    ],
    stakeCurrency: {
      coinDenom: SYMBOL,
      coinMinimalDenom: MINIMAL_DENOM,
      coinDecimals: 6,
    },
  });
}

/**
 * Enable the chain in a Cosmos wallet and return the first account address.
 */
async function getCosmosAddress(keplrLike: any): Promise<string> {
  await keplrLike.enable(COSMOS_CHAIN_ID);
  const key = await keplrLike.getKey(COSMOS_CHAIN_ID);
  return key.bech32Address;
}

export const useWalletStore = create<WalletState>((set, get) => ({
  walletType: null,
  connected: false,
  address: null,

  connectWallet: async (type) => {
    // --- Keplr ---
    if (type === "keplr") {
      const keplr = (window as any).keplr;
      if (!keplr) {
        throw new Error(
          "Keplr extension not found. Please install Keplr from https://keplr.app"
        );
      }
      await suggestCosmosChain(keplr);
      const addr = await getCosmosAddress(keplr);
      set({ walletType: "keplr", connected: true, address: addr });
      return;
    }

    // --- Leap ---
    if (type === "leap") {
      const leap = (window as any).leap;
      if (!leap) {
        throw new Error(
          "Leap extension not found. Please install Leap from https://leapwallet.io"
        );
      }
      await suggestCosmosChain(leap);
      const addr = await getCosmosAddress(leap);
      set({ walletType: "leap", connected: true, address: addr });
      return;
    }

    // --- Cosmostation ---
    if (type === "cosmostation") {
      const cosmostation = (window as any).cosmostation?.providers?.keplr;
      if (!cosmostation) {
        throw new Error(
          "Cosmostation extension not found. Please install Cosmostation."
        );
      }
      await suggestCosmosChain(cosmostation);
      const addr = await getCosmosAddress(cosmostation);
      set({ walletType: "cosmostation", connected: true, address: addr });
      return;
    }

    // --- MetaMask ---
    if (type === "metamask") {
      const ethereum = (window as any).ethereum;
      if (!ethereum) {
        throw new Error(
          "MetaMask extension not found. Please install MetaMask from https://metamask.io"
        );
      }

      const evmChainIdDec =
        process.env.NEXT_PUBLIC_EVM_CHAIN_ID || "7777";
      const evmChainIdHex = "0x" + Number(evmChainIdDec).toString(16);
      const evmChainName =
        process.env.NEXT_PUBLIC_EVM_CHAIN_NAME || "Sovereign L1 EVM";
      const evmRpcUrl =
        process.env.NEXT_PUBLIC_EVM_RPC_URL || "https://evm-rpc.yourchain.io";
      const explorerUrl =
        process.env.NEXT_PUBLIC_EXPLORER_URL || "http://localhost:3001";

      // Switch or add the Sovereign EVM chain
      try {
        await ethereum.request({
          method: "wallet_switchEthereumChain",
          params: [{ chainId: evmChainIdHex }],
        });
      } catch (switchErr: any) {
        // 4902 = chain not added yet
        if (switchErr.code === 4902) {
          await ethereum.request({
            method: "wallet_addEthereumChain",
            params: [
              {
                chainId: evmChainIdHex,
                chainName: evmChainName,
                nativeCurrency: {
                  name: SYMBOL,
                  symbol: SYMBOL,
                  decimals: 18,
                },
                rpcUrls: [evmRpcUrl],
                blockExplorerUrls: [explorerUrl],
              },
            ],
          });
        } else {
          throw switchErr;
        }
      }

      // Request accounts
      const accounts: string[] = await ethereum.request({
        method: "eth_requestAccounts",
      });
      if (!accounts || accounts.length === 0) {
        throw new Error("MetaMask returned no accounts.");
      }
      set({
        walletType: "metamask",
        connected: true,
        address: accounts[0],
      });
      return;
    }

    throw new Error(`Unsupported wallet type: ${type}`);
  },

  disconnectWallet: () => {
    set({
      walletType: null,
      connected: false,
      address: null,
    });
  },
}));
