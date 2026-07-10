"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Terminal, ShieldAlert, Award, Calendar, Layers, Download, ChevronLeft, ChevronRight } from "lucide-react";

interface Deployment {
  address: string;
  standard: string;
  deployer: string;
  txHash: string;
  blockHeight: number;
  blockTime: string;
  verified: boolean;
}

export default function DeploymentsPage() {
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Pagination cursor states
  const [cursor, setCursor] = useState<string>("");
  const [hasMore, setHasMore] = useState<boolean>(false);
  const [prevCursors, setPrevCursors] = useState<string[]>([]);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchDeployments = async () => {
      setLoading(true);
      setError(null);
      try {
        const url = `${API_BASE}/api/rest/v1/explorer/contracts/deployments?limit=15${cursor ? `&cursor=${cursor}` : ""}`;
        const resp = await fetch(url);
        if (!resp.ok) {
          throw new Error("Failed to load deployments history.");
        }
        const data = await resp.json();
        setDeployments(data.deployments || []);
        setHasMore(data.hasMore || false);
      } catch (err: any) {
        setError(err.message || "Network error occurred.");
      } finally {
        setLoading(false);
      }
    };
    fetchDeployments();
  }, [cursor, API_BASE]);

  const handleNext = () => {
    if (deployments.length > 0 && hasMore) {
      const last = deployments[deployments.length - 1];
      const nextCursorStr = btoa(`${last.blockHeight},${last.address}`);
      setPrevCursors([...prevCursors, cursor]);
      setCursor(nextCursorStr);
    }
  };

  const handlePrev = () => {
    if (prevCursors.length > 0) {
      const prev = prevCursors[prevCursors.length - 1];
      setPrevCursors(prevCursors.slice(0, -1));
      setCursor(prev);
    }
  };

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">Deployments</span>
      </nav>

      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between border-b border-gray-800 pb-6 gap-4">
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Layers className="w-8 h-8 text-blue-500" />
            Verified Deployments
          </h1>
          <p className="text-gray-400 mt-1">Real-time deployments registry of EVM verified contracts and CosmWasm codes.</p>
        </div>
        <div className="flex items-center gap-3">
          <a
            href={`${API_BASE}/api/rest/v1/explorer/contracts/deployments?download=true`}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-2 px-3 py-1.5 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-xs text-gray-300 hover:text-white transition"
          >
            <Download className="h-3.5 w-3.5" /> Download Schema
          </a>
        </div>
      </div>

      {/* Content Area */}
      {loading ? (
        <div className="flex items-center justify-center min-h-[300px]">
          <div className="text-gray-400 font-mono animate-pulse">Loading deployments data...</div>
        </div>
      ) : error ? (
        <div className="bg-red-950/20 border border-red-900 p-6 rounded-xl flex items-start gap-3">
          <ShieldAlert className="h-6 w-6 text-red-500 shrink-0 mt-0.5" />
          <div>
            <h3 className="text-white font-bold">Error loading deployments</h3>
            <p className="text-gray-400 text-sm mt-1">{error}</p>
          </div>
        </div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-2xl shadow-lg overflow-hidden">
          <div className="p-4 border-b border-gray-900 flex items-center justify-between">
            <span className="text-xs text-gray-500 font-mono uppercase tracking-wider font-bold">Deployments List</span>
            <div className="flex items-center space-x-2">
              <button
                onClick={handlePrev}
                disabled={prevCursors.length === 0}
                className="p-1.5 bg-gray-900 border border-gray-800 rounded-lg text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition"
              >
                <ChevronLeft className="h-4 w-4" />
              </button>
              <button
                onClick={handleNext}
                disabled={!hasMore}
                className="p-1.5 bg-gray-900 border border-gray-800 rounded-lg text-gray-400 hover:text-white disabled:opacity-30 disabled:hover:text-gray-400 transition"
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm text-gray-400 font-mono">
              <thead className="bg-black/50 text-xs text-gray-500 uppercase tracking-wider font-bold">
                <tr>
                  <th className="p-4">Contract Address</th>
                  <th className="p-4">Standard</th>
                  <th className="p-4">Deployer</th>
                  <th className="p-4">Tx Hash</th>
                  <th className="p-4">Height</th>
                  <th className="p-4 text-right">Age</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900">
                {deployments.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="p-8 text-center text-gray-500 font-mono">No deployments found in the network yet.</td>
                  </tr>
                ) : (
                  deployments.map((d) => (
                    <tr key={d.address} className="hover:bg-gray-900/30 transition text-xs">
                      <td className="p-4 font-bold text-white flex items-center gap-2">
                        {d.standard === "EVM" ? (
                          <Link href={`/evm/tokens/${d.address}`} className="text-blue-500 hover:underline block truncate max-w-[180px]">
                            {d.address}
                          </Link>
                        ) : (
                          <Link href={`/contracts/${d.address}`} className="text-purple-500 hover:underline block truncate max-w-[180px]">
                            {d.address}
                          </Link>
                        )}
                        {d.verified && (
                          <span className="inline-flex items-center gap-0.5 px-1 py-0.2 bg-green-950 border border-green-900 rounded text-[9px] text-green-400 font-semibold uppercase">
                            <Award className="h-2.5 w-2.5" /> Verified
                          </span>
                        )}
                      </td>
                      <td className="p-4">
                        <span className={`px-2 py-0.5 rounded text-[10px] font-semibold uppercase border ${
                          d.standard === "EVM" ? "bg-blue-950/40 border-blue-900 text-blue-400" : "bg-purple-950/40 border-purple-900 text-purple-400"
                        }`}>
                          {d.standard}
                        </span>
                      </td>
                      <td className="p-4 text-gray-500 truncate max-w-[150px]">
                        <Link href={`/address/${d.deployer}`} className="text-gray-400 hover:text-white transition">
                          {d.deployer}
                        </Link>
                      </td>
                      <td className="p-4">
                        <Link href={`/txs/${d.txHash}`} className="text-blue-500 hover:underline">
                          {d.txHash.slice(0, 14)}...
                        </Link>
                      </td>
                      <td className="p-4 font-bold text-white">{d.blockHeight}</td>
                      <td className="p-4 text-right text-gray-500 flex items-center justify-end gap-1.5">
                        <Calendar className="h-3 w-3" />
                        {new Date(d.blockTime).toLocaleString()}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
