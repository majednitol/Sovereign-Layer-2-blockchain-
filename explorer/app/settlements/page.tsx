"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { Layers, CheckCircle2, ChevronRight, Activity, Cpu } from "lucide-react";

interface Settlement {
  id: number;
  witness: string;
  status: string;
  chainId: string;
  txHash: string;
  height: number;
  time: string;
}

export default function SettlementsPage() {
  const [settlements, setSettlements] = useState<Settlement[]>([]);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchSettlements = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/settlements`);
        if (resp.ok) {
          const data = await resp.json();
          if (data.settlements) {
            setSettlements(data.settlements.map((s: any) => ({
              id: Number(s.id),
              witness: s.witness,
              status: s.status,
              chainId: s.chainId,
              txHash: s.txHash,
              height: Number(s.height),
              time: s.time,
            })));
          }
        }
      } catch (err) {
        console.warn("Using simulated settlements list", err);
        setSettlements([
          { id: 1, witness: "sovereign1witness", status: "finalized", chainId: "sovereign-1", txHash: "3f9c8d2a...", height: 120000, time: new Date().toISOString() },
        ]);
      } finally {
        setLoading(false);
      }
    };
    fetchSettlements();
  }, []);

  if (loading) {
    return (
      <div className="p-6 max-w-6xl mx-auto flex items-center justify-center min-h-[400px]">
        <div className="text-gray-400">Loading settlements...</div>
      </div>
    );
  }

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-gray-300">Settlements</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-800 pb-4">
        <h1 className="text-3xl font-bold tracking-tight text-white flex items-center gap-2">
          <Layers className="w-8 h-8 text-blue-500" />
          Batch Settlements Registry
        </h1>
        <p className="text-gray-400 mt-2">Rollup state commitments and witness verification audits.</p>
      </div>

      {/* Settlements Table */}
      <div className="bg-gray-900 border border-gray-800 rounded-xl overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm text-gray-400">
            <thead className="bg-gray-950 text-xs text-gray-500 uppercase tracking-wider">
              <tr>
                <th className="p-4">Settlement ID</th>
                <th className="p-4">Chain ID</th>
                <th className="p-4">Height</th>
                <th className="p-4">Witness</th>
                <th className="p-4">Status</th>
                <th className="p-4 text-right">Details</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-850">
              {settlements.map((s) => (
                <tr key={s.id} className="hover:bg-gray-850/40 transition">
                  <td className="p-4 font-mono font-semibold text-white">#{s.id}</td>
                  <td className="p-4 font-mono text-gray-300">{s.chainId}</td>
                  <td className="p-4 font-mono">#{s.height}</td>
                  <td className="p-4 font-mono text-xs text-gray-500">{s.witness.slice(0, 15)}...</td>
                  <td className="p-4">
                    <span className="px-2.5 py-0.5 text-xs bg-green-950 text-green-400 border border-green-900 rounded font-semibold uppercase">
                      {s.status}
                    </span>
                  </td>
                  <td className="p-4 text-right">
                    <Link
                      href={`/settlements/${s.id}`}
                      className="p-2 bg-gray-850 hover:bg-gray-800 rounded-lg text-gray-400 hover:text-white transition inline-block"
                    >
                      <ChevronRight className="w-4 h-4" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
