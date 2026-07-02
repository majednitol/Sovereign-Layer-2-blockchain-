"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Database, Clock, ChevronRight } from "lucide-react";

interface EvmBlock {
  height: number;
  time: string;
  txCount: number;
  gasUsed: string;
  miner: string;
}

export default function EvmBlocksPage() {
  const [blocks, setBlocks] = useState<EvmBlock[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchBlocks = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/blocks`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.blocks) {
            setBlocks(data.blocks.map((b: any) => ({
              height: Number(b.height),
              time: b.time,
              txCount: Number(b.txCount || 0),
              gasUsed: b.gasUsed || "0",
              miner: b.proposer || "0x0000000000000000000000000000000000000000",
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated EVM blocks", err);
        setBlocks([
          { height: 100, time: new Date().toISOString(), txCount: 5, gasUsed: "120,530", miner: "0x1234567890abcdef1234567890abcdef12345678" },
          { height: 99, time: new Date(Date.now() - 5000).toISOString(), txCount: 2, gasUsed: "42,000", miner: "0xabcdef1234567890abcdef1234567890abcdef12" },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchBlocks();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading EVM blocks...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">EVM Blocks</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Database className="w-8 h-8 text-blue-500" />
          EVM Blocks
        </h1>
        <p className="text-gray-400 mt-2">Latest mined blocks on Sovereign EVM Ring.</p>
      </div>

      {/* Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider font-semibold">
              <tr>
                <th className="p-4">Height</th>
                <th className="p-4">Gas Used</th>
                <th className="p-4">Tx Count</th>
                <th className="p-4">Miner / Proposer</th>
                <th className="p-4 text-right">Time</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {blocks.map((b) => (
                <tr key={b.height} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono font-semibold text-white">
                    <Link href={`/evm/blocks/${b.height}`} className="text-blue-500 hover:text-blue-400">
                      #{b.height}
                    </Link>
                  </td>
                  <td className="p-4 font-mono">{b.gasUsed}</td>
                  <td className="p-4 font-mono">{b.txCount} txs</td>
                  <td className="p-4 font-mono text-xs text-gray-500">{b.miner}</td>
                  <td className="p-4 text-xs text-gray-500 text-right">{new Date(b.time).toLocaleTimeString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
