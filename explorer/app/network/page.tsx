"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useWalletStore } from "@/store/wallet";
import { Shield, Smartphone, Globe, Layers, Key, Check } from "lucide-react";

export default function NetworkPage() {
  const { walletType, connected, address } = useWalletStore();
  const [keplrMessage, setKeplrMessage] = useState("");
  const [mmMessage, setMmMessage] = useState("");

  const addKeplrNetwork = async () => {
    setKeplrMessage("Suggesting chain to Keplr...");
    if (typeof window === "undefined" || !(window as any).keplr) {
      setKeplrMessage("Keplr extension not found. Please copy endpoints manually.");
      return;
    }

    const cosmosChainId = process.env.NEXT_PUBLIC_COSMOS_CHAIN_ID || "sovereign-1";
    const cosmosChainName = process.env.NEXT_PUBLIC_COSMOS_CHAIN_NAME || "Sovereign L1";
    const cosmosRpc = process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "https://rpc.yourchain.io";
    const cosmosRest = process.env.NEXT_PUBLIC_COSMOS_REST || "https://lcd.yourchain.io:1317";
    const symbol = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "SLT";
    const minimalDenom = "u" + symbol;

    try {
      await (window as any).keplr.experimentalSuggestChain({
        chainId: cosmosChainId,
        chainName: cosmosChainName,
        rpc: cosmosRpc,
        rest: cosmosRest,
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
          { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
        ],
        feeCurrencies: [
          { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6, gasPriceStep: { low: 0.01, average: 0.025, high: 0.04 } },
        ],
        stakeCurrency: { coinDenom: symbol, coinMinimalDenom: minimalDenom, coinDecimals: 6 },
      });
      setKeplrMessage(`${cosmosChainName} chain added to Keplr!`);
    } catch (err: any) {
      setKeplrMessage(`Failed: ${err.message || err}`);
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
    const evmRpcUrl = process.env.NEXT_PUBLIC_EVM_RPC_URL || "https://evm-rpc.yourchain.io";
    const symbol = process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "SLT";
    const explorerUrl = process.env.NEXT_PUBLIC_EXPLORER_URL || "http://localhost:3001";

    try {
      await (window as any).ethereum.request({
        method: "wallet_addEthereumChain",
        params: [
          {
            chainId: evmChainIdHex,
            chainName: evmChainName,
            nativeCurrency: { name: symbol, symbol: symbol, decimals: 18 },
            rpcUrls: [evmRpcUrl],
            blockExplorerUrls: [explorerUrl],
          },
        ],
      });
      setMmMessage(`${evmChainName} added to MetaMask!`);
    } catch (err: any) {
      setMmMessage(`Failed: ${err.message || err}`);
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
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Keplr card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2">
              <Smartphone className="text-blue-500 h-5 w-5" />
              <span>Cosmos Wallet (Keplr)</span>
            </h3>
            <p className="text-gray-400 text-sm">
              Add the Sovereign L1 Cosmos SDK native profile to your Keplr browser extension.
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

        {/* MetaMask card */}
        <div className="bg-gray-950 border border-gray-900 rounded-xl p-6 flex flex-col justify-between space-y-4">
          <div className="space-y-2">
            <h3 className="text-lg font-bold text-white flex items-center space-x-2">
              <Shield className="text-yellow-500 h-5 w-5" />
              <span>EVM Wallet (MetaMask)</span>
            </h3>
            <p className="text-gray-400 text-sm">
              Add the Sovereign L1 EVM execution environment profile to your MetaMask extension.
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
                  {process.env.NEXT_PUBLIC_EVM_RPC_URL || "https://evm-rpc.yourchain.io"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_RPC_URL || "https://rpc.yourchain.io"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">RPC URL (WS)</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_EVM_RPC_WS || "wss://evm-ws.yourchain.io"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_RPC_WS || "wss://rpc.yourchain.io/websocket"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">gRPC Endpoint</td>
                <td className="px-6 py-4 text-gray-500">—</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_GRPC || "grpc.yourchain.io:9090"}
                </td>
              </tr>
              <tr>
                <td className="px-6 py-4 font-medium text-gray-300">REST / LCD</td>
                <td className="px-6 py-4 text-gray-500">—</td>
                <td className="px-6 py-4 font-mono text-xs text-blue-400">
                  {process.env.NEXT_PUBLIC_COSMOS_REST || "https://lcd.yourchain.io:1317"}
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
                  {process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "SLT"}
                </td>
                <td className="px-6 py-4 font-mono text-xs text-gray-300">
                  u{process.env.NEXT_PUBLIC_CURRENCY_SYMBOL || "SLT"} (micro)
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
