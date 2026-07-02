"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, ArrowRight, RefreshCw, FileText } from "lucide-react";
import { useWalletStore } from "@/store/wallet";

interface Tx {
  hash: string;
  height: number;
  time: string;
  type: string;
  msgTypes: string[];
  status: number;
  fee: number;
}

export default function TXSPage() {
  const { walletType, connected, address, connectWallet, disconnectWallet } = useWalletStore();
  const [txs, setTxs] = useState<Tx[]>([]);
  const [loading, setLoading] = useState(true);
  const [cursor, setCursor] = useState("");
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [nextCursor, setNextCursor] = useState("");

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  const fetchTxs = async (targetCursor: string, isNext: boolean = true) => {
    setLoading(true);
    try {
      const url = `${API_BASE}/api/rest/v1/explorer/txs?pagination.limit=10&pagination.cursor=${targetCursor}`;
      const resp = await fetch(url);
      if (resp.ok) {
        const data = await resp.json();
        if (data.txs) {
          setTxs(data.txs.map((t: any) => ({
            hash: t.hash,
            height: Number(t.height),
            time: t.time,
            type: t.type,
            msgTypes: t.msgTypes || [],
            status: Number(t.status || 0),
            fee: Number(t.fee || 0),
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
      console.error("Failed to fetch transactions", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchTxs("");
  }, []);

  const handleNext = () => {
    if (hasMore && nextCursor) {
      fetchTxs(nextCursor, true);
    }
  };

  const handlePrev = () => {
    const historyCopy = [...cursorHistory];
    const prevCursor = historyCopy.pop() || "";
    setCursorHistory(historyCopy);
    fetchTxs(prevCursor, false);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white">Transactions</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4 flex justify-between items-center">
        <div className="flex items-center space-x-3">
          <FileText className="text-blue-500 h-8 w-8" />
          <div>
            <h1 className="text-3xl font-bold tracking-tight text-white">Transactions</h1>
            <p className="text-gray-400 mt-1">Sovereign L1 Transaction ledger</p>
          </div>
        </div>

        <button 
          onClick={() => fetchTxs("")}
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
                  <th className="py-4 px-6">Tx Hash</th>
                  <th className="py-4 px-6">Height</th>
                  <th className="py-4 px-6">Timestamp</th>
                  <th className="py-4 px-6">Type</th>
                  <th className="py-4 px-6">Message Type</th>
                  <th className="py-4 px-6 text-right">Fee</th>
                  <th className="py-4 px-6 text-right">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-900/50 text-sm text-gray-300">
                {txs.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-10 text-center text-gray-500">
                      No transactions found.
                    </td>
                  </tr>
                ) : (
                  txs.map((tx) => (
                    <tr key={tx.hash} className="hover:bg-gray-900/30 transition">
                      <td className="py-4 px-6 font-mono text-xs text-blue-500">
                        <Link href={`/txs/${tx.hash}`} className="hover:underline">
                          {tx.hash.slice(0, 18)}...{tx.hash.slice(-6)}
                        </Link>
                      </td>
                      <td className="py-4 px-6 font-bold text-gray-300">
                        <Link href={`/blocks/${tx.height}`} className="hover:text-blue-400 hover:underline">
                          #{tx.height}
                        </Link>
                      </td>
                      <td className="py-4 px-6 text-gray-400">
                        {new Date(tx.time).toLocaleString()}
                      </td>
                      <td className="py-4 px-6">
                        <span className="capitalize px-2 py-0.5 bg-gray-900 border border-gray-800 text-xs rounded text-gray-400 font-medium">
                          {tx.type}
                        </span>
                      </td>
                      <td className="py-4 px-6 font-mono text-xs text-gray-400">
                        {tx.msgTypes[0] || "Msg"}
                      </td>
                      <td className="py-4 px-6 text-right font-mono text-xs text-gray-500">
                        {tx.fee.toLocaleString()} uSLT
                      </td>
                      <td className="py-4 px-6 text-right">
                        <span className={`inline-block px-2 py-0.5 rounded text-xs font-bold ${tx.status === 0 ? "bg-green-950 text-green-400 border border-green-900" : "bg-red-950 text-red-400 border border-red-900"} border`}>
                          {tx.status === 0 ? "Success" : "Failed"}
                        </span>
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
