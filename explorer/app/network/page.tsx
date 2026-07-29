"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useWalletStore } from "@/store/wallet";
import { Shield, Smartphone, Globe } from "lucide-react";

export default function NetworkPage() {
  const { walletType, connected, address } = useWalletStore();
  const [keplrMessage, setKeplrMessage] = useState("");
  const [cosmostationMessage, setCosmostationMessage] = useState("");
  const [okxMessage, setOkxMessage] = useState("");
  const [mmMessage, setMmMessage] = useState("");

  const addKeplrNetwork = async () => {
    setKeplrMessage("Suggesting chain to Keplr...");
    if (typeof window === "undefined" || !(window as any).keplr) {
      setKeplrMessage("Keplr extension not found. Please copy endpoints manually.");
      return;
    }

    const cosmosChainId = process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1";
    const cosmosChainName = process.env.NEXT_PUBLIC_COSMOS_CHAIN_NAME || "Sovereign L1";
    const cosmosRpc = process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "http://localhost:26657";
    const cosmosRest = process.env.NEXT_PUBLIC_COSMOS_REST || "http://localhost:1317";
    const symbol = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV";
    const minimalDenom = "u" + symbol.toLowerCase();

    try {
      await (window as any).keplr.experimentalSuggestChain({
        chainId: cosmosChainId,
        chainName: cosmosChainName,
        rpc: cosmosRpc,
        rest: cosmosRest,
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
          { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
        ],
        feeCurrencies: [
          { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6, gasPriceStep: { low: 0.01, average: 0.025, high: 0.04 } },
        ],
        stakeCurrency: { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
        features: ["eth-secp256k1-cosmos", "eth-address-gen", "eth-key-sign"],
      });
      setKeplrMessage(`${cosmosChainName} chain added to Keplr!`);
    } catch (err: any) {
      setKeplrMessage(`Failed: ${err.message || err}`);
    }
  };

  const addCosmostationNetwork = async () => {
    setCosmostationMessage("Suggesting chain to Cosmostation...");
    const win = window as any;
    if (!win.cosmostation) {
      setCosmostationMessage("Cosmostation extension not found.");
      return;
    }

    const cosmosChainId = process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1";
    const cosmosChainName = process.env.NEXT_PUBLIC_COSMOS_CHAIN_NAME || "Sovereign L1";
    const cosmosRpc = process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "http://localhost:26657";
    const cosmosRest = process.env.NEXT_PUBLIC_COSMOS_REST || "http://localhost:1317";
    const symbol = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV";
    const minimalDenom = "u" + symbol.toLowerCase();

    try {
      if (win.cosmostation.cosmos?.request) {
        try {
          await win.cosmostation.cosmos.request({
            method: "cos_addChain",
            params: {
              chainId: cosmosChainId,
              chainName: cosmosChainName,
              addressPrefix: "cosmos",
              baseDenom: minimalDenom,
              displayDenom: symbol,
              restURL: cosmosRest,
              coinType: "60",
              decimals: 6,
              gasRate: {
                low: "0.01",
                average: "0.025",
                tiny: "0.001",
              },
            },
          });
        } catch (e) {
          console.warn("cos_addChain skipped/failed:", e);
        }
      }

      const cosmostationProvider = win.cosmostation.providers?.keplr || win.keplr;
      if (cosmostationProvider?.experimentalSuggestChain) {
        await cosmostationProvider.experimentalSuggestChain({
          chainId: cosmosChainId,
          chainName: cosmosChainName,
          rpc: cosmosRpc,
          rest: cosmosRest,
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
            { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
          ],
          feeCurrencies: [
            { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6, gasPriceStep: { low: 0.01, average: 0.025, high: 0.04 } },
          ],
          stakeCurrency: { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
          features: ["eth-secp256k1-cosmos", "eth-address-gen", "eth-key-sign"],
        });
      }
      setCosmostationMessage(`${cosmosChainName} added to Cosmostation!`);
    } catch (err: any) {
      setCosmostationMessage(`Failed: ${err.message || err}`);
    }
  };

  const addOkxNetwork = async () => {
    setOkxMessage("Suggesting chain to OKX Wallet...");
    const win = window as any;
    const okx = win.okxwallet?.keplr || win.okxwallet;
    if (!okx) {
      setOkxMessage("OKX Wallet extension not found.");
      return;
    }

    const cosmosChainId = process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1";
    const cosmosChainName = process.env.NEXT_PUBLIC_COSMOS_CHAIN_NAME || "Sovereign L1";
    const cosmosRpc = process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "http://localhost:26657";
    const cosmosRest = process.env.NEXT_PUBLIC_COSMOS_REST || "http://localhost:1317";
    const symbol = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV";
    const minimalDenom = "u" + symbol.toLowerCase();

    try {
      await okx.experimentalSuggestChain({
        chainId: cosmosChainId,
        chainName: cosmosChainName,
        rpc: cosmosRpc,
        rest: cosmosRest,
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
          { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
        ],
        feeCurrencies: [
          { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6, gasPriceStep: { low: 0.01, average: 0.025, high: 0.04 } },
        ],
        stakeCurrency: { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
        features: ["eth-secp256k1-cosmos", "eth-address-gen", "eth-key-sign"],
      });
      setOkxMessage(`${cosmosChainName} added to OKX Wallet!`);
    } catch (err: any) {
      setOkxMessage(`Failed: ${err.message || err}`);
    }
  };

  const addMetaMaskNetwork = async () => {
    setMmMessage("Adding Sovereign L1 EVM to MetaMask...");
    if (typeof window === "undefined" || !(window as any).ethereum) {
      setMmMessage("MetaMask extension not found. Please copy endpoints manually.");
      return;
    }

    const evmChainIdDec = process.env.NEXT_PUBLIC_EVM_CHAIN_ID || "7777";
    const evmChainIdHex = "0x" + Number(evmChainIdDec).toString(16);
    const evmChainName = process.env.NEXT_PUBLIC_EVM_CHAIN_NAME || "Sovereign L1 EVM";
    const evmRpcUrl = process.env.NEXT_PUBLIC_EVM_RPC_URL || "http://localhost:8545";
    const symbol = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV";
    const explorerUrl = process.env.NEXT_PUBLIC_EXPLORER_URL || "http://localhost:3001";

    const httpsRpcUrl = evmRpcUrl.startsWith("http://")
      ? evmRpcUrl.replace("http://", "https://")
      : evmRpcUrl;

    const chainParams: any = {
      chainId: evmChainIdHex,
      chainName: evmChainName,
      nativeCurrency: { name: symbol, symbol: symbol, decimals: 18 },
      rpcUrls: [httpsRpcUrl, evmRpcUrl],
    };

    if (explorerUrl.startsWith("https://")) {
      chainParams.blockExplorerUrls = [explorerUrl];
    }

    try {
      await (window as any).ethereum.request({
        method: "wallet_addEthereumChain",
        params: [chainParams],
      });
      setMmMessage(`${evmChainName} added to MetaMask!`);
    } catch (err: any) {
      // Fallback try with single httpsRpcUrl if array with http failed
      try {
        await (window as any).ethereum.request({
          method: "wallet_addEthereumChain",
          params: [{ ...chainParams, rpcUrls: [httpsRpcUrl] }],
        });
        setMmMessage(`${evmChainName} added to MetaMask!`);
      } catch (fallbackErr: any) {
        if (fallbackErr.message && fallbackErr.message.includes("HTTPS")) {
          setMmMessage(`MetaMask API requires HTTPS for programmatic add. Please add manually in MetaMask: Network Name: ${evmChainName}, RPC URL: ${evmRpcUrl}, Chain ID: ${evmChainIdDec}, Symbol: ${symbol}`);
        } else {
          setMmMessage(`Failed: ${fallbackErr.message || fallbackErr}`);
        }
      }
    }
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Network Config</span>
      </nav>

      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold tracking-tight text-white">Network Configuration</h1>
        <p className="text-gray-400 mt-1">Official developer connection hub and RPC parameters.</p>
      </div>

      {/* Quick Add Section */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Keplr card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2">
              <Smartphone className="text-blue-500 h-5 w-5" />
              <span>Keplr Wallet</span>
            </h3>
            <p className="text-gray-400 text-sm">
              Add Sovereign L1 Cosmos SDK profile to Keplr.
            </p>
          </div>
          <div className="space-y-3">
            <button
              onClick={addKeplrNetwork}
              className="w-full py-2.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-medium transition"
            >
              Add to Keplr
            </button>
            {keplrMessage && (
              <div className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 rounded-md font-mono">
                {keplrMessage}
              </div>
            )}
          </div>
        </div>

        {/* Cosmostation card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2">
              <Smartphone className="text-purple-500 h-5 w-5" />
              <span>Cosmostation</span>
            </h3>
            <p className="text-gray-400 text-sm">
              Add Sovereign L1 profile to Cosmostation.
            </p>
          </div>
          <div className="space-y-3">
            <button
              onClick={addCosmostationNetwork}
              className="w-full py-2.5 bg-purple-600 hover:bg-purple-500 text-white rounded-lg font-medium transition"
            >
              Add to Cosmostation
            </button>
            {cosmostationMessage && (
              <div className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 rounded-md font-mono">
                {cosmostationMessage}
              </div>
            )}
          </div>
        </div>

        {/* OKX Wallet card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2">
              <Globe className="text-green-500 h-5 w-5" />
              <span>OKX Wallet</span>
            </h3>
            <p className="text-gray-400 text-sm">
              Add Sovereign L1 profile to OKX Web3 Wallet.
            </p>
          </div>
          <div className="space-y-3">
            <button
              onClick={addOkxNetwork}
              className="w-full py-2.5 bg-gray-800 hover:bg-gray-700 text-white border border-gray-700 rounded-lg font-medium transition"
            >
              Add to OKX
            </button>
            {okxMessage && (
              <div className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 rounded-md font-mono">
                {okxMessage}
              </div>
            )}
          </div>
        </div>

        {/* MetaMask card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2">
              <Shield className="text-yellow-500 h-5 w-5" />
              <span>MetaMask</span>
            </h3>
            <p className="text-gray-400 text-sm">
              Add Sovereign L1 EVM profile to MetaMask.
            </p>
          </div>
          <div className="space-y-3">
            <button
              onClick={addMetaMaskNetwork}
              className="w-full py-2.5 bg-yellow-600 hover:bg-yellow-500 text-white rounded-lg font-medium transition"
            >
              Add to MetaMask
            </button>
            {mmMessage && (
              <div className="text-xs px-3 py-2 bg-gray-900 border border-gray-800 text-gray-300 rounded-md font-mono">
                {mmMessage}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Specifications Table */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl overflow-hidden shadow-lg">
        <div className="px-6 py-4 border-b border-gray-900">
          <h3 className="text-lg font-bold text-white">Connection Details</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-black/50 text-gray-400 uppercase text-xs">
              <tr>
                <th className="px-6 py-3">Parameter</th>
                <th className="px-6 py-3">EVM Runtime</th>
                <th className="px-6 py-3">Cosmos Runtime</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-900">
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">RPC URL (HTTP)</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_EVM_RPC_URL || "http://localhost:8545"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "http://localhost:26657"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">RPC URL (WS)</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_EVM_RPC_WS || "ws://localhost:8546"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_RPC_WS || "ws://localhost:26657/websocket"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">gRPC Endpoint</td>
                <td className="px-6 py-4 text-gray-500">—</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_GRPC || "localhost:9090"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">REST / LCD</td>
                <td className="px-6 py-4 text-gray-500">—</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_REST || "http://localhost:1317"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">Chain ID</td>
                <td className="px-6 py-4 font-mono text-xs text-gray-300">
                  {process.env.NEXT_PUBLIC_EVM_CHAIN_ID || "7777"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-gray-300">
                  {process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">Currency Symbol</td>
                <td className="px-6 py-4 font-mono text-xs text-gray-300">
                  {process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-gray-300">
                  u{process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "CSOV"} (micro)
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
