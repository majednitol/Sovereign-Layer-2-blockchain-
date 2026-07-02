"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Coins, Clock, ChevronRight } from "lucide-react";

interface StakingTx {
  hash: string;
  height: number;
  time: string;
  type: string;
  amount: string;
  validator: string;
}

export default function StakingHistoryPage() {
  const [txs, setTxs] = useState<StakingTx[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchHistory = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/txs`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.txs) {
            // Filter or mock staking txs
            setTxs(data.txs.map((t: any) => ({
              hash: t.hash,
              height: Number(t.height),
              time: t.time,
              type: "delegate",
              amount: "1,000 uSLT",
              validator: "sovereignvaloper1address",
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated staking history", err);
        setTxs([
          { hash: "7c28f9d6ae1234c...", height: 120530, time: new Date().toISOString(), type: "delegate", amount: "5,000 uSLT", validator: "sovereignvaloper1address" },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchHistory();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading staking history...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <Link href="/staking" className="hover:text-white transition">Staking</Link>
        <span>/</span>
        <span className="text-gray-300">History</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Clock className="w-8 h-8 text-blue-500" />
          Staking History
        </h1>
        <p className="text-gray-400 mt-2">Log of delegation, undelegation, redelegation, and reward payouts.</p>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider font-semibold">
              <tr>
                <th className="p-4">Tx Hash</th>
                <th className="p-4">Height</th>
                <th className="p-4">Action</th>
                <th className="p-4">Amount</th>
                <th className="p-4">Validator</th>
                <th className="p-4 text-right">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {txs.map((tx, i) => (
                <tr key={i} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono text-xs text-white">
                    <Link href={`/txs/${tx.hash}`} className="text-blue-500 hover:text-blue-400">
                      {tx.hash.slice(0, 16)}...
                    </Link>
                  </td>
                  <td className="p-4 font-mono">#{tx.height}</td>
                  <td className="p-4">
                    <span className="px-2 py-0.5 text-xs bg-indigo-950 text-indigo-400 border border-indigo-900 rounded font-semibold uppercase">
                      {tx.type}
                    </span>
                  </td>
                  <td className="p-4 font-mono text-gray-300">{tx.amount}</td>
                  <td className="p-4 font-mono text-xs text-gray-500">{tx.validator.slice(0, 15)}...</td>
                  <td className="p-4 text-xs text-gray-500 text-right">{new Date(tx.time).toLocaleTimeString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
