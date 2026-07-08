"use client";

import React, { useState } from "react";
import Link from "next/link";
import { ArrowLeft, Search, Filter, RefreshCw, ChevronRight } from "lucide-react";

export default function AdvancedFilterPage() {
  const [sender, setSender] = useState("");
  const [receiver, setReceiver] = useState("");
  const [txType, setTxType] = useState("all");
  const [status, setStatus] = useState("all");
  const [startBlock, setStartBlock] = useState("");
  const [endBlock, setEndBlock] = useState("");

  const [results, setResults] = useState<any[]>([]);
  const [searching, setSearching] = useState(false);

  const handleSearch = () => {
    setSearching(true);
    setTimeout(() => {
      // Mock results matching query criteria
      setResults([
        { hash: "0x8d92a10be43210be892a10be892a10be892a10be892a10be892a10be", height: 120530, type: "EVM", fee: "0.0025 SOV", status: "success", time: new Date().toLocaleTimeString() },
        { hash: "0x5f3a09e0129bcfe170298a09ee09ea090a908a908d098e09fcd09090", height: 120525, type: "CosmWasm", fee: "0.0031 SOV", status: "success", time: new Date(Date.now() - 60000).toLocaleTimeString() }
      ]);
      setSearching(false);
    }, 1000);
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/txs" className="hover:text-white transition">Transactions</Link>
        <span>/</span>
        <span className="text-white font-medium">Advanced Filter</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/txs" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Filter className="text-blue-500 w-8 h-8" />
            Advanced Transaction Filter
          </h1>
          <p className="text-gray-400 mt-1">Deep-search index database using custom constraints and heights.</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Constraints Form */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 lg:col-span-1 shadow-lg">
          <h3 className="text-base font-bold text-white border-b border-gray-900 pb-2">Filter Parameters</h3>
          
          <div className="space-y-4 text-xs">
            <div>
              <label className="block text-gray-400 font-bold uppercase mb-1">Sender Address</label>
              <input 
                type="text" 
                placeholder="sov1... or 0x..." 
                value={sender} 
                onChange={(e) => setSender(e.target.value)}
                className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-mono"
              />
            </div>

            <div>
              <label className="block text-gray-400 font-bold uppercase mb-1">Recipient Address</label>
              <input 
                type="text" 
                placeholder="sov1... or 0x..." 
                value={receiver} 
                onChange={(e) => setReceiver(e.target.value)}
                className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-mono"
              />
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-gray-400 font-bold uppercase mb-1">Start Block</label>
                <input 
                  type="number" 
                  placeholder="Min height" 
                  value={startBlock} 
                  onChange={(e) => setStartBlock(e.target.value)}
                  className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-mono"
                />
              </div>
              <div>
                <label className="block text-gray-400 font-bold uppercase mb-1">End Block</label>
                <input 
                  type="number" 
                  placeholder="Max height" 
                  value={endBlock} 
                  onChange={(e) => setEndBlock(e.target.value)}
                  className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-mono"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-gray-400 font-bold uppercase mb-1">Tx Type</label>
                <select 
                  value={txType} 
                  onChange={(e) => setTxType(e.target.value)}
                  className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-sans cursor-pointer"
                >
                  <option value="all">All Types</option>
                  <option value="evm">EVM Call</option>
                  <option value="wasm">CosmWasm</option>
                  <option value="bridge">Bridge Action</option>
                </select>
              </div>
              <div>
                <label className="block text-gray-400 font-bold uppercase mb-1">Status</label>
                <select 
                  value={status} 
                  onChange={(e) => setStatus(e.target.value)}
                  className="w-full bg-black border border-gray-900 rounded-xl p-3 text-white focus:border-blue-500 outline-none font-sans cursor-pointer"
                >
                  <option value="all">All status</option>
                  <option value="success">Success only</option>
                  <option value="fail">Failures only</option>
                </select>
              </div>
            </div>

            <button 
              onClick={handleSearch}
              disabled={searching}
              className="w-full py-3 bg-blue-600 hover:bg-blue-500 disabled:bg-gray-800 text-white font-bold text-xs uppercase tracking-wider rounded-xl transition flex items-center justify-center gap-1.5"
            >
              {searching ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
              {searching ? "filtering indexes..." : "execute search"}
            </button>
          </div>
        </div>

        {/* Search Results viewport */}
        <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 lg:col-span-2 shadow-lg space-y-4">
          <h3 className="text-base font-bold text-white border-b border-gray-900 pb-2">Matching Transactions</h3>

          {results.length > 0 ? (
            <div className="divide-y divide-gray-900">
              {results.map((tx) => (
                <div key={tx.hash} className="py-4 flex justify-between items-center hover:bg-gray-900/10 px-2 rounded-xl transition">
                  <div className="space-y-1">
                    <Link href={`/txs/${tx.hash}`} className="text-xs font-mono font-bold text-blue-500 hover:underline block truncate max-w-[250px]">
                      {tx.hash}
                    </Link>
                    <div className="flex gap-2 text-[10px] text-gray-500">
                      <span>Height: #{tx.height}</span>
                      <span>•</span>
                      <span>Type: {tx.type}</span>
                    </div>
                  </div>

                  <div className="text-right space-y-1">
                    <span className={`inline-flex px-2 py-0.5 rounded text-[9px] font-bold uppercase ${
                      tx.status === "success" ? "bg-green-950 text-green-400 border border-green-900" : "bg-red-950 text-red-400 border border-red-900"
                    }`}>
                      {tx.status}
                    </span>
                    <div className="text-[10px] text-gray-400 font-mono">Fee: {tx.fee}</div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="py-20 text-center text-gray-500 text-xs">
              Define search conditions and run filter to fetch transaction rows.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
