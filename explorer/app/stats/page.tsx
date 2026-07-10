"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, BarChart2, ShieldAlert, Cpu, Layers } from "lucide-react";

interface StatItem {
  name: string;
  value: string;
  description: string;
}

export default function StatsPage() {
  const [stats, setStats] = useState<StatItem[]>([
    { name: "Total Transactions Indexed", value: "24,582,100", description: "Aggregate sum of EVM and Cosmos SDK transactions." },
    { name: "Average Gas Price", value: "0.002500 CSOV", description: "Average transaction gas price base fee." },
    { name: "Total Verified Contracts", value: "154", description: "Count of verified Solidity and CosmWasm contract nodes." },
    { name: "Active Validator Set", value: "20 Nodes", description: "Active block producers signing slot sequences." },
    { name: "Total Staking Ratio", value: "59.45% Bonded", description: "Percentage of total supply bonded in staking consensus." }
  ]);

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Stats</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <BarChart2 className="text-blue-500 w-8 h-8" />
            Sovereign network stats Overview
          </h1>
          <p className="text-gray-400 mt-1">Summary database ledger aggregates and statistics overview.</p>
        </div>
      </div>

      <div className="bg-gray-950 border border-gray-900 rounded-2xl overflow-hidden shadow-lg p-6 space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6 text-sm">
          {stats.map((s, idx) => (
            <div key={idx} className="bg-gray-900/40 border border-gray-850 p-5 rounded-2xl space-y-2 hover:border-gray-800 transition">
              <div className="text-xs font-bold text-gray-500 uppercase">{s.name}</div>
              <div className="text-2xl font-extrabold text-white font-mono">{s.value}</div>
              <p className="text-xs text-gray-400 leading-relaxed">{s.description}</p>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
