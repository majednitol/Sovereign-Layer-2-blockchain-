"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Play, Code, Cpu, AlertCircle, CheckCircle2 } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

export default function ContractExecutePage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const { connected, address, walletType, connectWallet } = useWalletStore();
  const [execMsg, setExecMsg] = useState('{\n  "transfer": {\n    "recipient": "sovereign1recipient",\n    "amount": "1000"\n  }\n}');
  const [funds, setFunds] = useState("");
  const [loading, setLoading] = useState(false);
  const [txHash, setTxHash] = useState<string | null>(null);

  const handleExecute = () => {
    if (!connected) {
      alert("Please connect Keplr wallet first.");
      return;
    }
    setLoading(true);
    setTxHash(null);
    setTimeout(() => {
      setLoading(false);
      setTxHash("7f28a9b6c0e812d3456789abcdef...");
    }, 1500);
  };

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/contracts" className="hover:text-white transition">Contracts</Link>
        <span>/</span>
        <Link href={`/contracts/${addr}`} className="hover:text-white transition font-mono text-xs">{addr.slice(0, 10)}...</Link>
        <span>/</span>
        <span className="text-gray-300">Execute</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Code className="w-8 h-8 text-blue-500" />
          Contract Execution Console
        </h1>
        <p className="text-gray-400 mt-2">Construct and sign CosmWasm state transition transaction execution messages.</p>
      </div>

      {txHash && (
        <div className="p-4 bg-green-950/30 border border-green-800 rounded-lg flex items-start gap-3">
          <CheckCircle2 className="w-5 h-5 text-green-400 mt-0.5" />
          <div>
            <h3 className="font-semibold text-green-400">Transaction Broadcasted</h3>
            <p className="text-sm text-green-300 mt-1">Transaction executed successfully.</p>
            <div className="text-xs text-gray-500 font-mono mt-1 break-all">Tx Hash: {txHash}</div>
          </div>
        </div>
      )}

      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-6">
        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-300">Execute Message (JSON)</label>
          <textarea
            rows={8}
            value={execMsg}
            onChange={(e) => setExecMsg(e.target.value)}
            className="w-full bg-gray-950 border border-gray-800 rounded-lg p-4 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          />
        </div>

        <div className="space-y-2">
          <label className="block text-sm font-medium text-gray-300">Sent Funds (Optional, e.g., 1000uSLT)</label>
          <input
            type="text"
            placeholder="e.g. 1000uSLT"
            value={funds}
            onChange={(e) => setFunds(e.target.value)}
            className="w-full bg-gray-950 border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          />
        </div>

        {connected ? (
          <button
            onClick={handleExecute}
            disabled={loading}
            className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            <Play className="w-4 h-4 fill-current" />
            {loading ? "Signing with Keplr..." : "Execute Contract Call"}
          </button>
        ) : (
          <button
            onClick={() => connectWallet("keplr")}
            className="w-full py-3 bg-yellow-600 hover:bg-yellow-500 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            Connect Keplr to Sign & Execute
          </button>
        )}
      </div>
    </div>
  );
}
