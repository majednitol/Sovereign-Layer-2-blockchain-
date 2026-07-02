"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Play, Database } from "lucide-react";

export default function EvmContractReadPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const [response, setResponse] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleRead = () => {
    setLoading(true);
    setResponse(null);
    setTimeout(() => {
      setLoading(false);
      setResponse("Sovereign L1 Token");
    }, 1000);
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
        <span className="text-gray-300">Read</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Database className="w-8 h-8 text-blue-500" />
          Read EVM Contract State
        </h1>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-xl p-6 space-y-6">
        <div className="space-y-4">
          <h2 className="text-lg font-bold text-white">Call Method: name()</h2>
          <button
            onClick={handleRead}
            disabled={loading}
            className="py-2.5 px-6 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            <Play className="w-4 h-4 fill-current" />
            {loading ? "Reading State..." : "Query Method"}
          </button>
        </div>

        {response && (
          <div className="space-y-2 pt-4 border-t border-gray-850">
            <div className="text-xs text-gray-500 uppercase">Response Value</div>
            <div className="bg-gray-950 p-4 border border-gray-850 rounded-lg font-mono text-sm text-gray-200">
              {response}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
