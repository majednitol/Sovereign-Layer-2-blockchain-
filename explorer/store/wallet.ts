import { create } from "zustand";

const COSMOS_CHAIN_ID =
  process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1";
const COSMOS_CHAIN_NAME =
  process.env.NEXT_PUBLIC_COSMOS_CHAIN_NAME || "Sovereign L1";
const COSMOS_RPC =
  process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "http://localhost:26657";
const COSMOS_REST =
  process.env.NEXT_PUBLIC_COSMOS_REST || "http://localhost:1317";
const SYMBOL = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV";
const MINIMAL_DENOM = "ucsov";

interface WalletState {
  walletType: string | null;
  connected: boolean;
  address: string | null;
  connectWallet: (type: "keplr" | "metamask" | "okx" | "cosmostation") => Promise<void>;
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
    bip44: { coinType: 60 },
    bech32Config: {
      bech32PrefixAccAddr: "cosmos",
      bech32PrefixAccPub: "cosmospub",
      bech32PrefixValAddr: "cosmosvaloper",
      bech32PrefixValPub: "cosmosvaloperpub",
      bech32PrefixConsAddr: "cosmosvalcons",
      bech32PrefixConsPub: "cosmosvalconspub",
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
    features: ["eth-secp256k1-cosmos", "eth-address-gen", "eth-key-sign"],
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

    // --- OKX Wallet ---
    if (type === "okx") {
      const okx = (window as any).okxwallet?.keplr || (window as any).okxwallet;
      if (!okx) {
        throw new Error(
          "OKX Wallet extension not found. Please install OKX Wallet."
        );
      }
      await suggestCosmosChain(okx);
      const addr = await getCosmosAddress(okx);
      set({ walletType: "okx", connected: true, address: addr });
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

      // Request accounts FIRST so wallet connects regardless of RPC add policy
      const accounts: string[] = await ethereum.request({
        method: "eth_requestAccounts",
      });
      if (!accounts || accounts.length === 0) {
        throw new Error("MetaMask returned no accounts.");
      }

      const evmChainIdDec =
        process.env.NEXT_PUBLIC_EVM_CHAIN_ID || "7777";
      const evmChainIdHex = "0x" + Number(evmChainIdDec).toString(16);
      const evmChainName =
        process.env.NEXT_PUBLIC_EVM_CHAIN_NAME || "Sovereign L1 EVM";
      const evmRpcUrl =
        process.env.NEXT_PUBLIC_EVM_RPC_URL || "http://localhost:8545";
      const explorerUrl =
        process.env.NEXT_PUBLIC_EXPLORER_URL || "http://localhost:3001";

      // Non-blocking attempt to switch or suggest network
      try {
        await ethereum.request({
          method: "wallet_switchEthereumChain",
          params: [{ chainId: evmChainIdHex }],
        });
      } catch (switchErr: any) {
        if (switchErr.code === 4902) {
          const httpsRpcUrl = evmRpcUrl.startsWith("http://")
            ? evmRpcUrl.replace("http://", "https://")
            : evmRpcUrl;
          const chainParams: any = {
            chainId: evmChainIdHex,
            chainName: evmChainName,
            nativeCurrency: {
              name: SYMBOL,
              symbol: SYMBOL,
              decimals: 18,
            },
            rpcUrls: [httpsRpcUrl, evmRpcUrl],
          };
          if (explorerUrl.startsWith("https://")) {
            chainParams.blockExplorerUrls = [explorerUrl];
          }
          try {
            await ethereum.request({
              method: "wallet_addEthereumChain",
              params: [chainParams],
            });
          } catch (addErr) {
            try {
              await ethereum.request({
                method: "wallet_addEthereumChain",
                params: [{ ...chainParams, rpcUrls: [httpsRpcUrl] }],
              });
            } catch (fallbackAddErr) {
              console.warn("MetaMask programmatic network suggestion skipped:", fallbackAddErr);
            }
          }
        }
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

// Setup live listeners for wallet account changes
if (typeof window !== "undefined") {
  const win = window as any;

  // Listen for MetaMask account changes
  if (win.ethereum && win.ethereum.on) {
    win.ethereum.on("accountsChanged", (accounts: string[]) => {
      const state = useWalletStore.getState();
      if (state.walletType === "metamask") {
        if (accounts && accounts.length > 0) {
          useWalletStore.setState({ address: accounts[0], connected: true });
        } else {
          useWalletStore.getState().disconnectWallet();
        }
      }
    });

    win.ethereum.on("chainChanged", () => {
      window.location.reload();
    });
  }

  // Listen for Keplr / Cosmostation / OKX keystore changes
  window.addEventListener("keplr_keystorechange", async () => {
    const state = useWalletStore.getState();
    if (state.connected && (state.walletType === "keplr" || state.walletType === "okx" || state.walletType === "cosmostation")) {
      let provider: any = null;
      if (state.walletType === "keplr") provider = (window as any).keplr;
      if (state.walletType === "okx") provider = (window as any).okxwallet?.keplr || (window as any).okxwallet;
      if (state.walletType === "cosmostation") provider = (window as any).cosmostation?.providers?.keplr;

      if (provider) {
        try {
          const key = await provider.getKey(COSMOS_CHAIN_ID);
          if (key && key.bech32Address) {
            useWalletStore.setState({ address: key.bech32Address, connected: true });
          }
        } catch (e) {
          console.warn("Failed to update Cosmos address on keystore change:", e);
        }
      }
    }
  });
}
