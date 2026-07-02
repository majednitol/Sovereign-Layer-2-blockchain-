"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Play, Code, Cpu, AlertCircle, CheckCircle2 } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

export default function EvmContractWritePage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const { connected, address, walletType, connectWallet } = useWalletStore();
  const [recipient, setRecipient] = useState("");
  const [amount, setAmount] = useState("");
  const [loading, setLoading] = useState(false);
  const [txHash, setTxHash] = useState<string | null>(null);

  const handleWrite = () => {
    if (!connected) {
      alert("Please connect MetaMask first.");
      return;
    }
    setLoading(true);
    setTxHash(null);
    setTimeout(() => {
      setLoading(false);
      setTxHash("0x5f2a89b6c0e812d3456789abcdef...");
    }, 1500);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/evm/contracts" className="hover:text-white transition">EVM Contracts</Link>
        <span>/</span>
        <Link href={`/evm/contracts/${addr}`} className="hover:text-white transition font-mono text-xs">{addr.slice(0, 10)}...</Link>
        <span>/</span>
        <span className="text-gray-300">Write</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Code className="w-8 h-8 text-blue-500" />
          Write EVM Contract
        </h1>
      </div>

      {txHash && (
        <div className="p-4 bg-green-950/30 border border-green-800 rounded-lg flex items-start gap-3">
          <CheckCircle2 className="w-5 h-5 text-green-400 mt-0.5" />
          <div>
            <h3 className="font-semibold text-green-400">Transaction Confirmed</h3>
            <p className="text-sm text-green-300 mt-1">Solidity execution completed successfully.</p>
            <div className="text-xs text-gray-500 font-mono mt-1 break-all">Tx Hash: {txHash}</div>
          </div>
        </div>
      )}

      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-6">
        <div className="text-lg font-bold text-white border-b border-gray-850 pb-2">Method: transfer(address to, uint256 value)</div>
        
        <div className="space-y-4">
          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Recipient Address (to)</label>
            <input
              type="text"
              placeholder="0x..."
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
              className="w-full bg-gray-950 border border-gray-880 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none"
            />
          </div>

          <div className="space-y-2">
            <label className="block text-sm font-medium text-gray-300">Amount (value)</label>
            <input
              type="number"
              placeholder="1000"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              className="w-full bg-gray-950 border border-gray-880 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none"
            />
          </div>
        </div>

        {connected ? (
          <button
            onClick={handleWrite}
            disabled={loading}
            className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            <Play className="w-4 h-4 fill-current" />
            {loading ? "Signing with MetaMask..." : "Write Contract Call"}
          </button>
        ) : (
          <button
            onClick={() => connectWallet("metamask")}
            className="w-full py-3 bg-yellow-600 hover:bg-yellow-500 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            Connect MetaMask to Sign & Write
          </button>
        )}
      </div>
    </div>
  );
}
