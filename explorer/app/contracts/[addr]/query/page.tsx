"use client";

import React, { useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { Play, Code, Database, AlertCircle } from "lucide-react";

export default function ContractQueryPage() {
  const params = useParams();
  const addr = params?.addr ? String(params.addr) : "";
  const [queryMsg, setQueryMsg] = useState('{\n  "balance": {\n    "address": "sovereign1address"\n  }\n}');
  const [response, setResponse] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleQuery = () => {
    setLoading(true);
    setResponse(null);
    setTimeout(() => {
      setLoading(false);
      setResponse(JSON.stringify({ balance: "1000000" }, null, 2));
    }, 1000);
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
        <span className="text-gray-300">Query</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Database className="w-8 h-8 text-blue-500" />
          Contract Query Console
        </h1>
        <p className="text-gray-400 mt-2">Execute read-only queries against CosmWasm contract state.</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="space-y-4">
          <h2 className="text-lg font-bold text-white">Query Message (JSON)</h2>
          <textarea
            rows={10}
            value={queryMsg}
            onChange={(e) => setQueryMsg(e.target.value)}
            className="w-full bg-gray-950 border border-gray-800 rounded-lg p-4 text-white font-mono text-sm focus:outline-none focus:border-blue-500"
          />
          <button
            onClick={handleQuery}
            disabled={loading}
            className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white rounded-lg font-medium transition flex items-center justify-center gap-2"
          >
            <Play className="w-4 h-4 fill-current" />
            {loading ? "Running Query..." : "Execute Query"}
          </button>
        </div>

        <div className="space-y-4">
          <h2 className="text-lg font-bold text-white">Response Output</h2>
          <div className="w-full bg-gray-950 border border-gray-800 rounded-lg p-4 min-h-[250px] font-mono text-sm text-gray-300 overflow-auto whitespace-pre-wrap">
            {response ? response : <span className="text-gray-600">Execute query to see output...</span>}
          </div>
        </div>
      </div>
    </div>
  );
}
