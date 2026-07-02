"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowRight, RefreshCw, Layers } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

interface Block {
  height: number;
  time: string;
  proposer: string;
  txCount: number;
  gasUsed: number;
  gasLimit: number;
  appHash: string;
}

export default function BLOCKSPage() {
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();
  const [blocks, setBlocks] = useState<Block[]>([]);
  const [loading, setLoading] = useState(true);
  const [cursor, setCursor] = useState("");
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState("");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchBlocks = async (targetCursor: string, isNext: boolean = true) => {
    setLoading(true);
    try {
      const url = `${API_BASE}/api/rest/v1/explorer/blocks?pagination.limit=10&pagination.cursor=${targetCursor}`;
      const resp = await fetch(url);
      if (resp.ok) {
        const data = await resp.json();
        if (data.blocks) {
          setBlocks(data.blocks.map((b: any) => ({
            height: Number(b.height),
            time: b.time,
            proposer: b.proposer,
            txCount: Number(b.txCount || 0),
            gasUsed: Number(b.gasUsed || 0),
            gasLimit: Number(b.gasLimit || 0),
            appHash: b.appHash,
          })));
          setHasMore(data.pagination?.hasMore || false);
          setNextCursor(data.pagination?.nextCursor || "");
          
          if (isNext && targetCursor) {
            setCursorHistory((prev) => [...prev, cursor]);
          }
          setCursor(targetCursor);
        }
      }
    } catch (err) {
      console.error("Failed to fetch blocks", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchBlocks("");
  }, []);

  const handleNext = () => {
    if (hasMore && nextCursor) {
      fetchBlocks(nextCursor, true);
    }
  };

  const handlePrev = () => {
    const historyCopy = [...cursorHistory];
    const prevCursor = historyCopy.pop() || "";
    setCursorHistory(historyCopy);
    fetchBlocks(prevCursor, false);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Blocks</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div className="flex items-center space-x-3">
          <Layers className="text-blue-500 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white">Blocks</h1>
            <p className="text-gray-400 mt-1">Sovereign L1 Block Ledger</p>
          </div>
        </div>

        <button 
          onClick={() => fetchBlocks("")}
          className="p-2 bg-gray-900 hover:bg-gray-800 border border-gray-800 rounded-lg text-gray-400 hover:text-white transition"
          title="Reload"
        >
          <RefreshCw className="h-4 w-4" />
        </button>
      </div>

      {/* Main Content Area */}
      {loading ? (
        <div className="flex justify-center items-center py-20">
          <RefreshCw className="h-8 w-8 text-blue-500 animate-spin" />
        </div>
      ) : (
        <div className="bg-gray-950 border border-gray-900 rounded-xl overflow-hidden shadow-xl">
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="bg-gray-900/50 text-gray-400 text-xs font-bold uppercase tracking-wider border-b border-gray-900">
                  <th className="py-4 px-6">Height</th>
                  <th className="py-4 px-6">Timestamp</th>
                  <th className="py-4 px-6">Proposer</th>
                  <th className="py-4 px-6 text-right">Transactions</th>
                  <th className="py-4 px-6 text-right">Gas Used / Limit</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900/50 text-sm text-gray-300">
                {blocks.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="py-10 text-center text-gray-500">
                      No blocks found.
                    </td>
                  </tr>
                ) : (
                  blocks.map((block) => (
                    <tr key={block.height} className="hover:bg-gray-900/30 transition">
                      <td className="py-4 px-6 font-bold text-blue-500">
                        <Link href={`/blocks/${block.height}`} className="hover:underline">
                          #{block.height}
                        </Link>
                      </td>
                      <td className="py-4 px-6 text-gray-400">
                        {new Date(block.time).toLocaleString()}
                      </td>
                      <td className="py-4 px-6 font-mono text-xs text-gray-400">
                        {block.proposer}
                      </td>
                      <td className="py-4 px-6 text-right font-medium text-white">
                        {block.txCount}
                      </td>
                      <td className="py-4 px-6 text-right font-mono text-xs text-gray-500">
                        {block.gasUsed.toLocaleString()} / {block.gasLimit.toLocaleString()}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          <div className="bg-gray-900/20 px-6 py-4 flex items-center justify-between border-t border-gray-900">
            <button
              onClick={handlePrev}
              disabled={cursorHistory.length === 0}
              className="px-4 py-2 bg-gray-900 border border-gray-800 hover:bg-gray-800 disabled:opacity-50 disabled:hover:bg-gray-900 text-sm font-medium text-white rounded-lg flex items-center space-x-2 transition"
            >
              <ArrowLeft className="h-4 w-4" />
              <span>Previous</span>
            </button>
            <button
              onClick={handleNext}
              disabled={!hasMore}
              className="px-4 py-2 bg-gray-900 border border-gray-800 hover:bg-gray-800 disabled:opacity-50 disabled:hover:bg-gray-900 text-sm font-medium text-white rounded-lg flex items-center space-x-2 transition"
            >
              <span>Next</span>
              <ArrowRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
