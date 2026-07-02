"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeftRight, Clock, ChevronRight } from "lucide-react";

interface EvmTx {
  hash: string;
  height: number;
  time: string;
  from: string;
  to: string;
  value: string;
  status: string;
}

export default function EvmTxsPage() {
  const [txs, setTxs] = useState<EvmTx[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchTxs = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/txs`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.txs) {
            setTxs(data.txs.map((t: any) => ({
              hash: t.hash,
              height: Number(t.height),
              time: t.time,
              from: t.sender || "0x0000000000000000000000000000000000000000",
              to: "0xcontractaddress",
              value: t.value || "0.00",
              status: t.status === 0 ? "success" : "failed",
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated EVM txs", err);
        setTxs([
          { hash: "0x3f5c9e2b1d7a8d05cf5d2eb1...", height: 100, time: new Date().toISOString(), from: "0xsender123...", to: "0xreceiver456...", value: "1.50", status: "success" },
          { hash: "0x4f8a2b9c8d7e123456789abc...", height: 99, time: new Date(Date.now() - 60000).toISOString(), from: "0xsender789...", to: "0xcontract999...", value: "0.00", status: "success" },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchTxs();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading EVM transactions...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">EVM Transactions</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <ArrowLeftRight className="w-8 h-8 text-blue-500" />
          EVM Transactions
        </h1>
        <p className="text-gray-400 mt-2">Latest executed transaction operations on Sovereign EVM.</p>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider font-semibold">
              <tr>
                <th className="p-4">Tx Hash</th>
                <th className="p-4">Height</th>
                <th className="p-4">Value</th>
                <th className="p-4">Status</th>
                <th className="p-4 text-right">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {txs.map((tx) => (
                <tr key={tx.hash} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono font-semibold text-white text-xs">
                    <Link href={`/evm/txs/${tx.hash}`} className="text-blue-500 hover:text-blue-400">
                      {tx.hash.slice(0, 16)}...
                    </Link>
                  </td>
                  <td className="p-4 font-mono">
                    <Link href={`/evm/blocks/${tx.height}`} className="hover:text-white transition">
                      #{tx.height}
                    </Link>
                  </td>
                  <td className="p-4 font-mono">{tx.value} SLT</td>
                  <td className="p-4">
                    <span className={`px-2 py-0.5 text-xs rounded font-semibold uppercase ${
                      tx.status === "success" ? "bg-green-950 text-green-400 border border-green-900" : "bg-red-950 text-red-400 border border-red-900"
                    }`}>
                      {tx.status}
                    </span>
                  </td>
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
