"use client";

import React, { useEffect, useState, Suspense } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Search, Loader2, ArrowRight, CornerDownRight, Database, FileText, User, Settings, ShieldAlert, Award, Image as ImageIcon } from "lucide-react";

interface SearchResultItem {
  type: string;
  id: string;
  label: string;
}

function SearchResultsContent() {
  const searchParams = useSearchParams();
  const query = searchParams.get("q") || "";
  const [results, setResults] = useState<SearchResultItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    if (!query) {
      setLoading(false);
      return;
    }
    const fetchResults = async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/search?query=${encodeURIComponent(query)}`);
        if (resp.ok) {
          const data = await resp.json();
          setResults(data.results || []);
        } else {
          throw new Error("Failed to load search results.");
        }
      } catch (err: any) {
        console.error(err);
        setError("Error querying global search. Please try again.");
      } finally {
        setLoading(false);
      }
    };
    fetchResults();
  }, [query]);

  const blocks = results.filter(r => r.type === "block");
  const txs = results.filter(r => r.type === "tx");
  const addresses = results.filter(r => r.type === "address");
  const contracts = results.filter(r => r.type === "contract");
  const validators = results.filter(r => r.type === "validator");
  const proposals = results.filter(r => r.type === "proposal" && !r.label.toLowerCase().includes("mock"));
  const nfts = results.filter(r => r.type === "nft");

  const getEntityLink = (item: SearchResultItem) => {
    switch (item.type) {
      case "block": return `/blocks/${item.id}`;
      case "tx": return `/txs/${item.id}`;
      case "address": return `/address/${item.id}`;
      case "contract": return `/contracts/${item.id}`;
      case "validator": return `/validators/${item.id}`;
      case "proposal": return `/governance/${item.id}`;
      case "nft": return `/evm/nfts/${item.id}`;
      default: return "#";
    }
  };

  return (
    <div className="space-y-8">
      {/* Search status/query display */}
      <div className="bg-gray-950 border border-gray-900 rounded-xl p-5 flex justify-between items-center shadow-lg">
        <div>
          <span className="text-xs text-gray-500 uppercase tracking-wider font-bold">Search Query</span>
          <h2 className="text-xl font-bold text-white mt-1 font-mono">
            &ldquo;{query || "Empty Query"}&rdquo;
          </h2>
        </div>
        <div className="text-sm text-gray-400">
          Found <span className="text-white font-bold">{results.length}</span> results
        </div>
      </div>

      {loading ? (
        <div className="flex flex-col items-center justify-center py-20 space-y-4">
          <Loader2 className="h-8 w-8 text-blue-500 animate-spin" />
          <p className="text-gray-400 text-sm font-semibold">Searching the Sovereign L1 ledger...</p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {/* Blocks */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <Database className="h-5 w-5 text-emerald-400 animate-pulse" />
                Blocks
              </h3>
              <span className="text-xs px-2 py-0.5 bg-emerald-950/50 text-emerald-400 rounded-full font-bold font-mono">
                {blocks.length}
              </span>
            </div>
            {blocks.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching blocks</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {blocks.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-semibold text-white font-mono">Height #{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-mono">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Transactions */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <FileText className="h-5 w-5 text-blue-400 animate-pulse" />
                Transactions
              </h3>
              <span className="text-xs px-2 py-0.5 bg-blue-950/50 text-blue-400 rounded-full font-bold font-mono">
                {txs.length}
              </span>
            </div>
            {txs.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching transactions</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {txs.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-mono text-white truncate max-w-[150px]">{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-mono">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Addresses */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <User className="h-5 w-5 text-purple-400 animate-pulse" />
                Addresses
              </h3>
              <span className="text-xs px-2 py-0.5 bg-purple-950/50 text-purple-400 rounded-full font-bold font-mono">
                {addresses.length}
              </span>
            </div>
            {addresses.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching addresses</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {addresses.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-mono text-white truncate max-w-[150px]">{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-mono">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Contracts */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <Settings className="h-5 w-5 text-amber-400 animate-pulse" />
                Contracts
              </h3>
              <span className="text-xs px-2 py-0.5 bg-amber-950/50 text-amber-400 rounded-full font-bold font-mono">
                {contracts.length}
              </span>
            </div>
            {contracts.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching contracts</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {contracts.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-mono text-white truncate max-w-[150px]">{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-mono">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Validators */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <ShieldAlert className="h-5 w-5 text-red-400 animate-pulse" />
                Validators
              </h3>
              <span className="text-xs px-2 py-0.5 bg-red-950/50 text-red-400 rounded-full font-bold font-mono">
                {validators.length}
              </span>
            </div>
            {validators.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching validators</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {validators.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-mono text-white truncate max-w-[150px]">{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-mono">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* Proposals */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <Award className="h-5 w-5 text-indigo-400 animate-pulse" />
                Proposals
              </h3>
              <span className="text-xs px-2 py-0.5 bg-indigo-950/50 text-indigo-400 rounded-full font-bold font-mono">
                {proposals.length}
              </span>
            </div>
            {proposals.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching proposals</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {proposals.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-semibold text-white">Proposal #{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-sans">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>

          {/* NFTs */}
          <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 space-y-4 shadow-lg shadow-black/40">
            <div className="flex items-center justify-between border-b border-gray-900 pb-3">
              <h3 className="font-bold text-white flex items-center gap-2">
                <ImageIcon className="h-5 w-5 text-pink-400 animate-pulse" />
                NFT Tokens
              </h3>
              <span className="text-xs px-2 py-0.5 bg-pink-950/50 text-pink-400 rounded-full font-bold font-mono">
                {nfts.length}
              </span>
            </div>
            {nfts.length === 0 ? (
              <p className="text-xs text-gray-500 italic">No matching NFT assets</p>
            ) : (
              <div className="space-y-3 max-h-60 overflow-y-auto">
                {nfts.map((item, idx) => (
                  <Link key={idx} href={getEntityLink(item)} className="block p-3 rounded-xl bg-gray-900 hover:bg-gray-850 border border-gray-850 hover:border-gray-800 transition">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-mono text-white truncate max-w-[150px]">{item.id}</span>
                      <CornerDownRight className="h-4 w-4 text-gray-500" />
                    </div>
                    <p className="text-xs text-gray-400 truncate mt-1 font-mono">{item.label}</p>
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default function SEARCHPage() {
  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Search</span>
      </nav>

      <div className="border-b border-gray-900 pb-4">
        <h1 className="text-3xl font-extrabold tracking-tight text-white">Unified Global Search</h1>
        <p className="text-gray-400 mt-1">Cross-referencing blocks, transactions, addresses, validators, proposals, and NFT collections</p>
      </div>

      <Suspense fallback={
        <div className="flex items-center justify-center py-20">
          <Loader2 className="h-8 w-8 text-blue-500 animate-spin" />
        </div>
      }>
        <SearchResultsContent />
      </Suspense>
    </div>
  );
}
