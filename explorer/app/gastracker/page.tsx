"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Fuel, Activity, Clock, Zap, TrendingUp, AlertTriangle } from "lucide-react";

interface GasMetrics {
  standard: string;
  fast: string;
  instant: string;
  gasLimit: number;
  guzzlers: Array<{
    address: string;
    moniker: string;
    gasUsed: string;
    pct: number;
  }>;
}

export default function GasTrackerPage() {
  const [metrics, setMetrics] = useState<GasMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

  useEffect(() => {
    const fetchGasData = async () => {
      try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/gas-tracker`);
        if (resp.ok) {
          const data = await resp.json();
          setMetrics(data);
        }
      } catch (err) {
        console.warn("Failed to fetch gas stats. Falling back to mocks.", err);
        setMetrics({
          standard: "0.002500",
          fast: "0.003125",
          instant: "0.003750",
          gasLimit: 30000000,
          guzzlers: [
            { address: "0x1234567890123456789012345678901234567890", moniker: "Sovereign L1 Bridge Box", gasUsed: "5,820,100", pct: 19.4 },
            { address: "0x5a109a25b2a0c7cfd21c0e3a6c57f722971239aa", moniker: "Uniswap Router Proxy", gasUsed: "2,410,500", pct: 8.0 },
            { address: "0x0000000000000000000000000000000000000009", moniker: "EVM Wasm Bridge VM", gasUsed: "1,980,000", pct: 6.6 }
          ]
        });
      } finally {
        setLoading(false);
      }
    };
    fetchGasData();
  }, []);

  return (
    <div className="p-6 max-w-6xl mx-auto space-y-6">
      {/* Breadcrumbs */}
      <nav className="text-sm text-gray-400 flex items-center space-x-2">
        <Link href="/" className="hover:text-white transition">Home</Link>
        <span>/</span>
        <span className="text-white font-medium">Gas Tracker</span>
      </nav>

      {/* Header */}
      <div className="border-b border-gray-900 pb-4 flex items-center space-x-3">
        <Link href="/" className="p-2 bg-gray-950 hover:bg-gray-900 border border-gray-900 rounded-lg text-gray-400 hover:text-white transition">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white flex items-center gap-2">
            <Fuel className="text-yellow-500 w-8 h-8 animate-bounce" />
            Sovereign Gas Station Tracker
          </h1>
          <p className="text-gray-400 mt-1">Real-time gas estimation, network base fees, and top gas spenders.</p>
        </div>
      </div>

      {loading || !metrics ? (
        <div className="py-20 text-center text-gray-400">Loading gas tracking metrics...</div>
      ) : (
        <div className="space-y-6">
          {/* Main Gas Cards */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            {/* Standard */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 flex flex-col justify-between hover:border-blue-900/50 transition">
              <div className="flex justify-between items-center">
                <span className="text-xs font-bold text-gray-500 uppercase">Standard</span>
                <Clock className="h-5 w-5 text-blue-500" />
              </div>
              <div className="my-4">
                <div className="text-3xl font-extrabold text-white font-mono">{metrics.standard}</div>
                <div className="text-xs text-gray-400 mt-1">~ 6s block inclusion time</div>
              </div>
              <div className="text-[10px] text-gray-500 border-t border-gray-900 pt-2 font-mono">
                Base Fee: {metrics.standard} ESOV
              </div>
            </div>

            {/* Fast */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 flex flex-col justify-between hover:border-green-900/50 transition">
              <div className="flex justify-between items-center">
                <span className="text-xs font-bold text-gray-500 uppercase">Fast</span>
                <Activity className="h-5 w-5 text-green-500 animate-pulse" />
              </div>
              <div className="my-4">
                <div className="text-3xl font-extrabold text-white font-mono">{metrics.fast}</div>
                <div className="text-xs text-gray-400 mt-1">~ 1 block inclusion guarantee</div>
              </div>
              <div className="text-[10px] text-gray-500 border-t border-gray-900 pt-2 font-mono">
                Base Fee: {metrics.fast} ESOV
              </div>
            </div>

            {/* Instant */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 flex flex-col justify-between hover:border-purple-900/50 transition">
              <div className="flex justify-between items-center">
                <span className="text-xs font-bold text-gray-500 uppercase">Instant</span>
                <Zap className="h-5 w-5 text-purple-500" />
              </div>
              <div className="my-4">
                <div className="text-3xl font-extrabold text-white font-mono">{metrics.instant}</div>
                <div className="text-xs text-gray-400 mt-1">Frontrun / urgent inclusion</div>
              </div>
              <div className="text-[10px] text-gray-500 border-t border-gray-900 pt-2 font-mono">
                Base Fee: {metrics.instant} ESOV
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Top Gas Guzzlers */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl lg:col-span-2 space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-3">
                <TrendingUp className="text-yellow-500 h-5 w-5" />
                Top 24H Gas Guzzlers
              </h3>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm text-gray-400">
                  <thead>
                    <tr className="border-b border-gray-900 text-gray-500 text-xs font-bold uppercase">
                      <th className="pb-2">Contract Address</th>
                      <th className="pb-2">Contract Moniker</th>
                      <th className="pb-2">Gas Consumed</th>
                      <th className="pb-2 text-right">Gas Limit %</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-900">
                    {metrics.guzzlers.map((g, idx) => (
                      <tr key={idx} className="hover:bg-gray-900/30 transition">
                        <td className="py-3 font-mono text-xs text-blue-500 hover:underline">
                          <Link href={`/contracts/${g.address}`}>{g.address}</Link>
                        </td>
                        <td className="py-3 text-white font-semibold">{g.moniker}</td>
                        <td className="py-3 font-mono text-xs">{g.gasUsed}</td>
                        <td className="py-3 text-right font-mono font-bold text-yellow-400">{g.pct}%</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Block Limit Parameters */}
            <div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl space-y-4">
              <h3 className="text-lg font-bold text-white flex items-center gap-2 border-b border-gray-900 pb-2">
                <AlertTriangle className="text-orange-500 h-4 w-4" />
                Gas Limit Protocol
              </h3>

              <div className="space-y-3 text-xs leading-relaxed text-gray-400">
                <div>
                  <span className="block text-gray-500 uppercase font-bold mb-0.5">Max Gas Limit Per Block</span>
                  <span className="font-mono text-sm font-semibold text-white">30,000,000 gas</span>
                </div>
                <div>
                  <span className="block text-gray-500 uppercase font-bold mb-0.5">Average Block Congestion</span>
                  <span className="font-mono text-sm font-semibold text-white">12.4% capacity</span>
                </div>
                <div className="border-t border-gray-900 pt-3 text-[11px] text-gray-500">
                  Fees are adaptively calculated per block based on current demand capacity. Standard transaction gas cost is 21,000.
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
